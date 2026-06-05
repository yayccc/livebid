package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/services/auction-service/internal/client"
	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"github.com/yayccc/livebid/services/auction-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errInvalidArgument   = errors.New("竞拍请求参数无效，请检查ID、金额、状态、时间范围或请求标识")
	errInvalidCredential = errors.New("未获取到有效登录身份，请先登录")
)

type AuctionGRPCHandler struct {
	auctionv1.UnimplementedAuctionServiceServer
	auctions   repository.AuctionRepository
	bids       repository.BidRecordRepository
	states     repository.AuctionStateStore
	goods      client.GoodsClient
	events     client.EventPublisher
	ids        *idgen.Generator
	delayLevel int
}

func NewAuctionGRPCHandler(
	auctions repository.AuctionRepository,
	bids repository.BidRecordRepository,
	states repository.AuctionStateStore,
	goods client.GoodsClient,
	events client.EventPublisher,
	ids *idgen.Generator,
	delayLevel int,
) *AuctionGRPCHandler {
	if ids == nil {
		ids = idgen.New(0)
	}
	return &AuctionGRPCHandler{
		auctions:   auctions,
		bids:       bids,
		states:     states,
		goods:      goods,
		events:     events,
		ids:        ids,
		delayLevel: delayLevel,
	}
}

func (h *AuctionGRPCHandler) CreateAuction(ctx context.Context, req *auctionv1.CreateAuctionRequest) (*auctionv1.CreateAuctionResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetGoodsId() <= 0 || !validMoney(req.GetStartPrice()) || !validMoney(req.GetBidIncrement()) {
		return nil, toGRPCError(errInvalidArgument)
	}
	if req.SealPrice != nil && req.GetSealPrice() < req.GetStartPrice() {
		return nil, toGRPCError(errInvalidArgument)
	}
	startTime, endTime, err := parseTimeRange(req.GetStartTime(), req.GetEndTime())
	if err != nil {
		return nil, toGRPCError(err)
	}
	if h.goods != nil {
		// 创建竞拍前校验商品存在且属于当前店铺，避免跨店铺绑定竞拍。
		if err := h.goods.ValidateGoodsForShop(ctx, req.GetGoodsId(), shopID); err != nil {
			return nil, toGRPCError(err)
		}
	}

	auction := &model.Auction{
		ID:           h.ids.Next(),
		GoodsID:      req.GetGoodsId(),
		ShopID:       shopID,
		StartPrice:   req.GetStartPrice(),
		BidIncrement: req.GetBidIncrement(),
		SealPrice:    req.SealPrice,
		CurrentPrice: req.GetStartPrice(),
		Status:       model.AuctionStatusPending,
		StartTime:    startTime,
		EndTime:      endTime,
	}
	if err := h.auctions.Create(ctx, auction); err != nil {
		return nil, toGRPCError(err)
	}
	return &auctionv1.CreateAuctionResponse{Auction: toProtoAuction(auction)}, nil
}

func (h *AuctionGRPCHandler) GetAuction(ctx context.Context, req *auctionv1.GetAuctionRequest) (*auctionv1.GetAuctionResponse, error) {
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	auction, err := h.auctions.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &auctionv1.GetAuctionResponse{Auction: toProtoAuction(auction)}, nil
}

func (h *AuctionGRPCHandler) GetAuctionByGoods(ctx context.Context, req *auctionv1.GetAuctionByGoodsRequest) (*auctionv1.GetAuctionByGoodsResponse, error) {
	if req.GetGoodsId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	auction, err := h.auctions.FindByGoodsID(ctx, req.GetGoodsId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &auctionv1.GetAuctionByGoodsResponse{Auction: toProtoAuction(auction)}, nil
}

func (h *AuctionGRPCHandler) ListShopAuctions(ctx context.Context, req *auctionv1.ListShopAuctionsRequest) (*auctionv1.ListShopAuctionsResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	filter := repository.ListAuctionFilter{
		ShopID:   shopID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.Status != nil {
		statusValue, err := parseAuctionStatus(req.GetStatus())
		if err != nil {
			return nil, toGRPCError(err)
		}
		filter.Status = &statusValue
	}
	list, total, err := h.auctions.ListByShop(ctx, filter)
	if err != nil {
		return nil, toGRPCError(err)
	}
	page, pageSize := normalizePagination(int(req.GetPage()), int(req.GetPageSize()), 10)
	return &auctionv1.ListShopAuctionsResponse{
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
		List:     toProtoAuctionList(list),
	}, nil
}

func (h *AuctionGRPCHandler) UpdateAuction(ctx context.Context, req *auctionv1.UpdateAuctionRequest) (*auctionv1.UpdateAuctionResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	auction, err := h.auctions.FindByIDForShop(ctx, req.GetId(), shopID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if auction.Status != model.AuctionStatusPending {
		return nil, toGRPCError(repository.ErrInvalidAuctionState)
	}
	if req.StartPrice != nil {
		if !validMoney(req.GetStartPrice()) {
			return nil, toGRPCError(errInvalidArgument)
		}
		auction.StartPrice = req.GetStartPrice()
		auction.CurrentPrice = req.GetStartPrice()
	}
	if req.BidIncrement != nil {
		if !validMoney(req.GetBidIncrement()) {
			return nil, toGRPCError(errInvalidArgument)
		}
		auction.BidIncrement = req.GetBidIncrement()
	}
	if req.SealPrice != nil {
		if req.GetSealPrice() < auction.StartPrice {
			return nil, toGRPCError(errInvalidArgument)
		}
		auction.SealPrice = req.SealPrice
	}
	if req.StartTime != nil {
		t := req.GetStartTime().AsTime()
		auction.StartTime = &t
	}
	if req.EndTime != nil {
		t := req.GetEndTime().AsTime()
		auction.EndTime = &t
	}
	if auction.StartTime != nil && auction.EndTime != nil && !auction.EndTime.After(*auction.StartTime) {
		return nil, toGRPCError(errInvalidArgument)
	}
	if err := h.auctions.UpdatePendingConfig(ctx, auction); err != nil {
		return nil, toGRPCError(err)
	}
	updated, err := h.auctions.FindByIDForShop(ctx, req.GetId(), shopID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &auctionv1.UpdateAuctionResponse{Auction: toProtoAuction(updated)}, nil
}

func (h *AuctionGRPCHandler) StartAuction(ctx context.Context, req *auctionv1.StartAuctionRequest) (*auctionv1.StartAuctionResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	auction, err := h.auctions.FindByIDForShop(ctx, req.GetId(), shopID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if auction.Status != model.AuctionStatusPending {
		return nil, toGRPCError(repository.ErrInvalidAuctionState)
	}
	now := time.Now()
	if auction.EndTime == nil {
		// 文档允许不传结束时间，这里给运行态一个保守兜底，避免 Redis 中出现无期限竞拍。
		endTime := now.Add(24 * time.Hour)
		auction.EndTime = &endTime
	}
	auction.Status = model.AuctionStatusRunning
	auction.StartTime = &now
	auction.Version++
	if err := h.states.LoadAuction(ctx, auction, now); err != nil {
		return nil, toGRPCError(err)
	}
	// Redis 是运行态事实来源；这里同步写一份 DB 快照，consumer 后续仍会按版本幂等回写。
	if err := h.auctions.UpdateStatusSnapshot(ctx, auction); err != nil && !errors.Is(err, repository.ErrOutdatedAuctionVersion) {
		return nil, toGRPCError(err)
	}
	h.publish(ctx, client.EventAuctionStarted, auction, nil)
	return &auctionv1.StartAuctionResponse{Auction: toProtoAuction(auction)}, nil
}

func (h *AuctionGRPCHandler) FinishAuction(ctx context.Context, req *auctionv1.FinishAuctionRequest) (*auctionv1.FinishAuctionResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if _, err := h.auctions.FindByIDForShop(ctx, req.GetId(), shopID); err != nil {
		return nil, toGRPCError(err)
	}
	// 手动结束先改 Redis，再由事件和本地快照回写 DB，保持与自动结束路径一致。
	state, err := h.states.FinishAuction(ctx, req.GetId(), time.Now())
	if err != nil {
		return nil, toGRPCError(err)
	}
	auction := auctionFromState(state)
	if state.Status == model.AuctionStatusDeal {
		dealPrice := state.CurrentPrice
		auction.DealPrice = &dealPrice
		h.publish(ctx, client.EventAuctionFinished, auction, nil)
	} else {
		h.publish(ctx, client.EventAuctionFailed, auction, nil)
	}
	_ = h.auctions.UpdateStatusSnapshot(ctx, auction)
	return &auctionv1.FinishAuctionResponse{Auction: toProtoAuction(auction)}, nil
}

func (h *AuctionGRPCHandler) CancelAuction(ctx context.Context, req *auctionv1.CancelAuctionRequest) (*auctionv1.CancelAuctionResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	auction, err := h.auctions.FindByIDForShop(ctx, req.GetId(), shopID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if auction.Status == model.AuctionStatusPending {
		// 待开始竞拍尚未加载进 Redis，取消时直接更新 DB 快照即可。
		now := time.Now()
		auction.Status = model.AuctionStatusCanceled
		auction.EndTime = &now
		auction.Version++
		if err := h.auctions.UpdateStatusSnapshot(ctx, auction); err != nil {
			return nil, toGRPCError(err)
		}
		h.publish(ctx, client.EventAuctionCancelled, auction, nil)
		return &auctionv1.CancelAuctionResponse{Auction: toProtoAuction(auction)}, nil
	}
	// 竞拍中取消必须走 Redis，避免绕过实时状态导致仍可出价。
	state, err := h.states.CancelAuction(ctx, req.GetId(), time.Now())
	if err != nil {
		return nil, toGRPCError(err)
	}
	auction = auctionFromState(state)
	h.publish(ctx, client.EventAuctionCancelled, auction, nil)
	_ = h.auctions.UpdateStatusSnapshot(ctx, auction)
	return &auctionv1.CancelAuctionResponse{Auction: toProtoAuction(auction)}, nil
}

func (h *AuctionGRPCHandler) DeleteAuction(ctx context.Context, req *auctionv1.DeleteAuctionRequest) (*auctionv1.DeleteAuctionResponse, error) {
	shopID, err := currentShopID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	if req.GetId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	if err := h.auctions.Delete(ctx, req.GetId(), shopID); err != nil {
		return nil, toGRPCError(err)
	}
	return &auctionv1.DeleteAuctionResponse{}, nil
}

func (h *AuctionGRPCHandler) PlaceBid(ctx context.Context, req *auctionv1.PlaceBidRequest) (*auctionv1.PlaceBidResponse, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	requestID := strings.TrimSpace(req.GetRequestId())
	if req.GetAuctionId() <= 0 || !validMoney(req.GetBidPrice()) || requestID == "" {
		return nil, toGRPCError(errInvalidArgument)
	}
	bidRecordID := h.ids.Next()
	now := time.Now()
	// 出价校验、幂等、最高价更新和倒计时刷新必须在 Redis Lua 中原子完成。
	result, err := h.states.PlaceBid(ctx, req.GetAuctionId(), userID, req.GetBidPrice(), requestID, bidRecordID, now)
	if err != nil {
		return nil, toGRPCError(err)
	}
	record := &model.BidRecord{
		ID:        result.BidRecordID,
		AuctionID: result.State.AuctionID,
		GoodsID:   result.State.GoodsID,
		ShopID:    result.State.ShopID,
		UserID:    userID,
		BidPrice:  req.GetBidPrice(),
		BidTime:   now,
	}
	// 出价成功后立即写审计记录；RocketMQ consumer 会用 bid_record_id 再做一次幂等补偿。
	_ = h.bids.Create(ctx, record)
	auction := auctionFromState(result.State)
	h.publish(ctx, client.EventBidAccepted, auction, map[string]any{
		"bid_record_id": result.BidRecordID,
		"user_id":       userID,
		"bid_price":     req.GetBidPrice(),
		"bid_time":      now.Unix(),
		"current_price": result.State.CurrentPrice,
		"bid_count":     result.State.BidCount,
		"request_id":    requestID,
	})
	h.publishDelay(ctx, auction, map[string]any{
		"expire_at": result.State.ExpireAt.UnixMilli(),
	})
	_ = h.auctions.UpdateStatusSnapshot(ctx, auction)
	return &auctionv1.PlaceBidResponse{
		Accepted:     true,
		CurrentPrice: result.State.CurrentPrice,
		BidCount:     result.State.BidCount,
		WinnerUserId: userID,
		ServerTime:   timestamppb.New(now),
		ExpireAt:     timestamppb.New(result.State.ExpireAt),
	}, nil
}

func (h *AuctionGRPCHandler) ListBidRecords(ctx context.Context, req *auctionv1.ListBidRecordsRequest) (*auctionv1.ListBidRecordsResponse, error) {
	if req.GetAuctionId() <= 0 {
		return nil, toGRPCError(errInvalidArgument)
	}
	list, total, err := h.bids.ListByAuction(ctx, repository.ListBidRecordFilter{
		AuctionID: req.GetAuctionId(),
		Page:      int(req.GetPage()),
		PageSize:  int(req.GetPageSize()),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	page, pageSize := normalizePagination(int(req.GetPage()), int(req.GetPageSize()), 20)
	return &auctionv1.ListBidRecordsResponse{
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
		List:     toProtoBidRecordList(list),
	}, nil
}

func (h *AuctionGRPCHandler) publish(ctx context.Context, eventType string, auction *model.Auction, data map[string]any) {
	if h.events == nil {
		return
	}
	// 事件携带完整快照，consumer 不依赖当前进程内存即可重建 DB 状态。
	data = mergeEventData(snapshotEventData(auction), data)
	event := client.NewAuctionEvent(stringID(h.ids.Next()), eventType, auction.ID, auction.ShopID, auction.Version, data)
	_ = h.events.Publish(ctx, event)
}

func (h *AuctionGRPCHandler) publishDelay(ctx context.Context, auction *model.Auction, data map[string]any) {
	if h.events == nil || h.delayLevel <= 0 {
		return
	}
	// 延迟检查只在 version/expire_at 未变化时生效，新出价会让旧检查消息自然失效。
	data = mergeEventData(snapshotEventData(auction), data)
	event := client.NewAuctionEvent(stringID(h.ids.Next()), client.EventAuctionExpireCheck, auction.ID, auction.ShopID, auction.Version, data)
	_ = h.events.PublishDelay(ctx, event, h.delayLevel)
}

func snapshotEventData(auction *model.Auction) map[string]any {
	// map 结构与服务设计中的事件 data 对齐，避免 consumer 反序列化时依赖 Go 私有类型。
	data := map[string]any{
		"goods_id":       auction.GoodsID,
		"shop_id":        auction.ShopID,
		"start_price":    auction.StartPrice,
		"bid_increment":  auction.BidIncrement,
		"current_price":  auction.CurrentPrice,
		"bid_count":      auction.BidCount,
		"status":         int32(auction.Status),
		"start_time":     timePtrUnixMilli(auction.StartTime),
		"end_time":       timePtrUnixMilli(auction.EndTime),
		"winner_user_id": int64(0),
		"deal_price":     int64(0),
		"seal_price":     int64(0),
	}
	if auction.WinnerUserID != nil {
		data["winner_user_id"] = *auction.WinnerUserID
	}
	if auction.DealPrice != nil {
		data["deal_price"] = *auction.DealPrice
	}
	if auction.SealPrice != nil {
		data["seal_price"] = *auction.SealPrice
	}
	return data
}

func mergeEventData(base map[string]any, extra map[string]any) map[string]any {
	for key, value := range extra {
		base[key] = value
	}
	return base
}

func parseTimeRange(start *timestamppb.Timestamp, end *timestamppb.Timestamp) (*time.Time, *time.Time, error) {
	var startTime *time.Time
	var endTime *time.Time
	if start != nil {
		value := start.AsTime()
		startTime = &value
	}
	if end != nil {
		value := end.AsTime()
		endTime = &value
	}
	if startTime != nil && endTime != nil && !endTime.After(*startTime) {
		return nil, nil, errInvalidArgument
	}
	return startTime, endTime, nil
}

func validMoney(value int64) bool {
	return value > 0
}

func currentShopID(ctx context.Context) (int64, error) {
	shopID, ok := identity.ShopID(ctx)
	if !ok {
		return 0, errInvalidCredential
	}
	return shopID, nil
}

func currentUserID(ctx context.Context) (int64, error) {
	userID, ok := identity.UserID(ctx)
	if !ok {
		return 0, errInvalidCredential
	}
	return userID, nil
}

func parseAuctionStatus(value int32) (model.AuctionStatus, error) {
	statusValue := model.AuctionStatus(value)
	switch statusValue {
	case model.AuctionStatusPending, model.AuctionStatusRunning, model.AuctionStatusDeal, model.AuctionStatusFailed, model.AuctionStatusCanceled:
		return statusValue, nil
	default:
		return model.AuctionStatusPending, errInvalidArgument
	}
}

func normalizePagination(page int, pageSize int, defaultPageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func toProtoAuction(auction *model.Auction) *auctionv1.Auction {
	if auction == nil {
		return nil
	}
	return &auctionv1.Auction{
		Id:           auction.ID,
		GoodsId:      auction.GoodsID,
		ShopId:       auction.ShopID,
		StartPrice:   auction.StartPrice,
		BidIncrement: auction.BidIncrement,
		SealPrice:    auction.SealPrice,
		CurrentPrice: auction.CurrentPrice,
		DealPrice:    auction.DealPrice,
		BidCount:     auction.BidCount,
		Status:       int32(auction.Status),
		StartTime:    timeToProto(auction.StartTime),
		EndTime:      timeToProto(auction.EndTime),
		WinnerUserId: auction.WinnerUserID,
		Version:      auction.Version,
		CreatedAt:    timestamppb.New(auction.CreatedAt),
		UpdatedAt:    timestamppb.New(auction.UpdatedAt),
	}
}

func toProtoAuctionList(list []*model.Auction) []*auctionv1.Auction {
	result := make([]*auctionv1.Auction, 0, len(list))
	for _, auction := range list {
		result = append(result, toProtoAuction(auction))
	}
	return result
}

func toProtoBidRecord(record *model.BidRecord) *auctionv1.BidRecord {
	if record == nil {
		return nil
	}
	return &auctionv1.BidRecord{
		Id:        record.ID,
		AuctionId: record.AuctionID,
		GoodsId:   record.GoodsID,
		ShopId:    record.ShopID,
		UserId:    record.UserID,
		BidPrice:  record.BidPrice,
		BidTime:   timestamppb.New(record.BidTime),
		CreatedAt: timestamppb.New(record.CreatedAt),
		UpdatedAt: timestamppb.New(record.UpdatedAt),
	}
}

func toProtoBidRecordList(list []*model.BidRecord) []*auctionv1.BidRecord {
	result := make([]*auctionv1.BidRecord, 0, len(list))
	for _, record := range list {
		result = append(result, toProtoBidRecord(record))
	}
	return result
}

func timeToProto(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(*value)
}

func timePtrUnixMilli(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.UnixMilli()
}

func auctionFromState(state repository.AuctionState) *model.Auction {
	auction := &model.Auction{
		ID:           state.AuctionID,
		GoodsID:      state.GoodsID,
		ShopID:       state.ShopID,
		StartPrice:   state.StartPrice,
		BidIncrement: state.BidIncrement,
		SealPrice:    state.SealPrice,
		CurrentPrice: state.CurrentPrice,
		BidCount:     state.BidCount,
		Status:       state.Status,
		StartTime:    &state.StartTime,
		EndTime:      &state.EndTime,
		WinnerUserID: state.WinnerUserID,
		Version:      state.Version,
	}
	if state.Status == model.AuctionStatusDeal {
		dealPrice := state.CurrentPrice
		auction.DealPrice = &dealPrice
	}
	return auction
}

func stringID(id int64) string {
	return "evt_" + strconvFormatInt(id)
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, errInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, errInvalidCredential):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, repository.ErrAuctionNotFound), errors.Is(err, repository.ErrBidRecordNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, repository.ErrAuctionDuplicated):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, repository.ErrInvalidAuctionState), errors.Is(err, repository.ErrBidTooLow), errors.Is(err, repository.ErrBidOverSealPrice), errors.Is(err, repository.ErrAuctionExpired), errors.Is(err, repository.ErrDuplicateBidRequest):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, client.ErrGoodsShopMismatch):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "竞拍服务内部错误，请稍后重试")
	}
}
