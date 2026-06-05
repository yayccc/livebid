package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type auctionServiceClient interface {
	CreateAuction(ctx context.Context, in *auctionv1.CreateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CreateAuctionResponse, error)
	GetAuction(ctx context.Context, in *auctionv1.GetAuctionRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionResponse, error)
	GetAuctionByGoods(ctx context.Context, in *auctionv1.GetAuctionByGoodsRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionByGoodsResponse, error)
	ListShopAuctions(ctx context.Context, in *auctionv1.ListShopAuctionsRequest, opts ...grpc.CallOption) (*auctionv1.ListShopAuctionsResponse, error)
	UpdateAuction(ctx context.Context, in *auctionv1.UpdateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.UpdateAuctionResponse, error)
	StartAuction(ctx context.Context, in *auctionv1.StartAuctionRequest, opts ...grpc.CallOption) (*auctionv1.StartAuctionResponse, error)
	FinishAuction(ctx context.Context, in *auctionv1.FinishAuctionRequest, opts ...grpc.CallOption) (*auctionv1.FinishAuctionResponse, error)
	CancelAuction(ctx context.Context, in *auctionv1.CancelAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CancelAuctionResponse, error)
	DeleteAuction(ctx context.Context, in *auctionv1.DeleteAuctionRequest, opts ...grpc.CallOption) (*auctionv1.DeleteAuctionResponse, error)
	ListBidRecords(ctx context.Context, in *auctionv1.ListBidRecordsRequest, opts ...grpc.CallOption) (*auctionv1.ListBidRecordsResponse, error)
}

type AuctionHandler struct {
	auctionClient auctionServiceClient
	rpcTimeout    time.Duration
}

func NewAuctionHandler(auctionClient auctionServiceClient, rpcTimeout time.Duration) *AuctionHandler {
	return &AuctionHandler{
		auctionClient: auctionClient,
		rpcTimeout:    normalizeRPCTimeout(rpcTimeout),
	}
}

type createAuctionRequest struct {
	GoodsID      int64  `json:"goods_id" binding:"required"`
	StartPrice   int64  `json:"start_price" binding:"required"`
	BidIncrement int64  `json:"bid_increment" binding:"required"`
	SealPrice    *int64 `json:"seal_price"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}

type updateAuctionRequest struct {
	StartPrice   *int64 `json:"start_price"`
	BidIncrement *int64 `json:"bid_increment"`
	SealPrice    *int64 `json:"seal_price"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}

type auctionResponse struct {
	ID           int64  `json:"id"`
	GoodsID      int64  `json:"goods_id"`
	ShopID       int64  `json:"shop_id"`
	StartPrice   int64  `json:"start_price"`
	BidIncrement int64  `json:"bid_increment"`
	SealPrice    *int64 `json:"seal_price,omitempty"`
	CurrentPrice int64  `json:"current_price"`
	DealPrice    *int64 `json:"deal_price,omitempty"`
	BidCount     int64  `json:"bid_count"`
	Status       int32  `json:"status"`
	StartTime    string `json:"start_time,omitempty"`
	EndTime      string `json:"end_time,omitempty"`
	WinnerUserID *int64 `json:"winner_user_id,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type auctionByGoodsResponse struct {
	ID           int64  `json:"id"`
	GoodsID      int64  `json:"goods_id"`
	ShopID       int64  `json:"shop_id"`
	StartPrice   int64  `json:"start_price"`
	BidIncrement int64  `json:"bid_increment"`
	SealPrice    *int64 `json:"seal_price,omitempty"`
	CurrentPrice int64  `json:"current_price"`
	BidCount     int64  `json:"bid_count"`
	Status       int32  `json:"status"`
	StartTime    string `json:"start_time,omitempty"`
	EndTime      string `json:"end_time,omitempty"`
}

type auctionListItemResponse struct {
	ID           int64  `json:"id"`
	GoodsID      int64  `json:"goods_id"`
	ShopID       int64  `json:"shop_id"`
	StartPrice   int64  `json:"start_price"`
	CurrentPrice int64  `json:"current_price"`
	BidCount     int64  `json:"bid_count"`
	Status       int32  `json:"status"`
	StartTime    string `json:"start_time,omitempty"`
	EndTime      string `json:"end_time,omitempty"`
}

type auctionListResponse struct {
	Total    int64                     `json:"total"`
	Page     int32                     `json:"page"`
	PageSize int32                     `json:"page_size"`
	List     []auctionListItemResponse `json:"list"`
}

type startAuctionResponse struct {
	ID        int64  `json:"id"`
	Status    int32  `json:"status"`
	StartTime string `json:"start_time,omitempty"`
}

type finishAuctionResponse struct {
	ID           int64  `json:"id"`
	Status       int32  `json:"status"`
	WinnerUserID *int64 `json:"winner_user_id"`
	DealPrice    *int64 `json:"deal_price"`
	EndTime      string `json:"end_time,omitempty"`
}

type cancelAuctionResponse struct {
	ID     int64 `json:"id"`
	Status int32 `json:"status"`
}

type bidRecordResponse struct {
	ID        int64  `json:"id"`
	AuctionID int64  `json:"auction_id"`
	GoodsID   int64  `json:"goods_id"`
	ShopID    int64  `json:"shop_id"`
	UserID    int64  `json:"user_id"`
	BidPrice  int64  `json:"bid_price"`
	BidTime   string `json:"bid_time,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type bidRecordListResponse struct {
	Total    int64               `json:"total"`
	Page     int32               `json:"page"`
	PageSize int32               `json:"page_size"`
	List     []bidRecordResponse `json:"list"`
}

func (h *AuctionHandler) Create(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	var req createAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "创建竞拍参数无效，请检查商品ID、起拍价、加价幅度、封顶价和竞拍时间")
		return
	}
	startTime, ok := parseOptionalTimestamp(c, req.StartTime)
	if !ok {
		return
	}
	endTime, ok := parseOptionalTimestamp(c, req.EndTime)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.auctionClient.CreateAuction(ctx, &auctionv1.CreateAuctionRequest{
		GoodsId:      req.GoodsID,
		ShopId:       shopID,
		StartPrice:   req.StartPrice,
		BidIncrement: req.BidIncrement,
		SealPrice:    req.SealPrice,
		StartTime:    startTime,
		EndTime:      endTime,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, toAuctionResponse(resp.GetAuction()))
}

func (h *AuctionHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.auctionClient.GetAuction(ctx, &auctionv1.GetAuctionRequest{Id: id})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, toAuctionResponse(resp.GetAuction()))
}

func (h *AuctionHandler) GetByGoods(c *gin.Context) {
	goodsID, ok := parseIDParam(c, "goods_id")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.auctionClient.GetAuctionByGoods(ctx, &auctionv1.GetAuctionByGoodsRequest{GoodsId: goodsID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, toAuctionByGoodsResponse(resp.GetAuction()))
}

func (h *AuctionHandler) ListShop(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	status, ok := parseOptionalInt32Query(c, "status")
	if !ok {
		return
	}
	page, ok := parseOptionalInt32Query(c, "page")
	if !ok {
		return
	}
	pageSize, ok := parseOptionalInt32Query(c, "page_size")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.auctionClient.ListShopAuctions(ctx, &auctionv1.ListShopAuctionsRequest{
		ShopId:   shopID,
		Status:   status,
		Page:     int32ValueOrZero(page),
		PageSize: int32ValueOrZero(pageSize),
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, auctionListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		List:     toAuctionListItemResponseList(resp.GetList()),
	})
}

func (h *AuctionHandler) Update(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req updateAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "修改竞拍参数无效，请提交合法的竞拍配置")
		return
	}
	startTime, ok := parseOptionalTimestamp(c, req.StartTime)
	if !ok {
		return
	}
	endTime, ok := parseOptionalTimestamp(c, req.EndTime)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.auctionClient.UpdateAuction(ctx, &auctionv1.UpdateAuctionRequest{
		Id:           id,
		ShopId:       shopID,
		StartPrice:   req.StartPrice,
		BidIncrement: req.BidIncrement,
		SealPrice:    req.SealPrice,
		StartTime:    startTime,
		EndTime:      endTime,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	_ = resp
	respondOK(c, gin.H{})
}

func (h *AuctionHandler) Start(c *gin.Context) {
	h.withShopAction(c, func(ctx context.Context, id int64, shopID int64) (*auctionv1.Auction, error) {
		resp, err := h.auctionClient.StartAuction(ctx, &auctionv1.StartAuctionRequest{Id: id, ShopId: shopID})
		if err != nil {
			return nil, err
		}
		return resp.GetAuction(), err
	}, func(auction *auctionv1.Auction) any {
		return toStartAuctionResponse(auction)
	})
}

func (h *AuctionHandler) Finish(c *gin.Context) {
	h.withShopAction(c, func(ctx context.Context, id int64, shopID int64) (*auctionv1.Auction, error) {
		resp, err := h.auctionClient.FinishAuction(ctx, &auctionv1.FinishAuctionRequest{Id: id, ShopId: shopID})
		if err != nil {
			return nil, err
		}
		return resp.GetAuction(), err
	}, func(auction *auctionv1.Auction) any {
		return toFinishAuctionResponse(auction)
	})
}

func (h *AuctionHandler) Cancel(c *gin.Context) {
	h.withShopAction(c, func(ctx context.Context, id int64, shopID int64) (*auctionv1.Auction, error) {
		resp, err := h.auctionClient.CancelAuction(ctx, &auctionv1.CancelAuctionRequest{Id: id, ShopId: shopID})
		if err != nil {
			return nil, err
		}
		return resp.GetAuction(), err
	}, func(auction *auctionv1.Auction) any {
		return toCancelAuctionResponse(auction)
	})
}

func (h *AuctionHandler) Delete(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	if _, err := h.auctionClient.DeleteAuction(ctx, &auctionv1.DeleteAuctionRequest{
		Id:     id,
		ShopId: shopID,
	}); err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, true)
}

func (h *AuctionHandler) ListBidRecords(c *gin.Context) {
	auctionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	page, ok := parseOptionalInt32Query(c, "page")
	if !ok {
		return
	}
	pageSize, ok := parseOptionalInt32Query(c, "page_size")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.auctionClient.ListBidRecords(ctx, &auctionv1.ListBidRecordsRequest{
		AuctionId: auctionID,
		Page:      int32ValueOrZero(page),
		PageSize:  int32ValueOrZero(pageSize),
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, bidRecordListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		List:     toBidRecordResponseList(resp.GetList()),
	})
}

func (h *AuctionHandler) withShopAction(c *gin.Context, call func(context.Context, int64, int64) (*auctionv1.Auction, error), data func(*auctionv1.Auction) any) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	auction, err := call(ctx, id, shopID)
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, data(auction))
}

func parseOptionalTimestamp(c *gin.Context, value string) (*timestamppb.Timestamp, bool) {
	if value == "" {
		return nil, true
	}
	parsed, err := parseDocumentTime(value)
	if err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "时间格式无效，请使用 YYYY-MM-DD HH:mm:ss 或 RFC3339 格式")
		return nil, false
	}
	return timestamppb.New(parsed), true
}

func parseDocumentTime(value string) (time.Time, error) {
	if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, value)
}

func toAuctionResponse(auction *auctionv1.Auction) auctionResponse {
	if auction == nil {
		return auctionResponse{}
	}
	return auctionResponse{
		ID:           auction.GetId(),
		GoodsID:      auction.GetGoodsId(),
		ShopID:       auction.GetShopId(),
		StartPrice:   auction.GetStartPrice(),
		BidIncrement: auction.GetBidIncrement(),
		SealPrice:    auction.SealPrice,
		CurrentPrice: auction.GetCurrentPrice(),
		DealPrice:    auction.DealPrice,
		BidCount:     auction.GetBidCount(),
		Status:       auction.GetStatus(),
		StartTime:    documentTimeString(auction.GetStartTime()),
		EndTime:      documentTimeString(auction.GetEndTime()),
		WinnerUserID: auction.WinnerUserId,
		CreatedAt:    documentTimeString(auction.GetCreatedAt()),
		UpdatedAt:    documentTimeString(auction.GetUpdatedAt()),
	}
}

func toAuctionByGoodsResponse(auction *auctionv1.Auction) auctionByGoodsResponse {
	if auction == nil {
		return auctionByGoodsResponse{}
	}
	return auctionByGoodsResponse{
		ID:           auction.GetId(),
		GoodsID:      auction.GetGoodsId(),
		ShopID:       auction.GetShopId(),
		StartPrice:   auction.GetStartPrice(),
		BidIncrement: auction.GetBidIncrement(),
		SealPrice:    auction.SealPrice,
		CurrentPrice: auction.GetCurrentPrice(),
		BidCount:     auction.GetBidCount(),
		Status:       auction.GetStatus(),
		StartTime:    documentTimeString(auction.GetStartTime()),
		EndTime:      documentTimeString(auction.GetEndTime()),
	}
}

func toAuctionListItemResponse(auction *auctionv1.Auction) auctionListItemResponse {
	if auction == nil {
		return auctionListItemResponse{}
	}
	return auctionListItemResponse{
		ID:           auction.GetId(),
		GoodsID:      auction.GetGoodsId(),
		ShopID:       auction.GetShopId(),
		StartPrice:   auction.GetStartPrice(),
		CurrentPrice: auction.GetCurrentPrice(),
		BidCount:     auction.GetBidCount(),
		Status:       auction.GetStatus(),
		StartTime:    documentTimeString(auction.GetStartTime()),
		EndTime:      documentTimeString(auction.GetEndTime()),
	}
}

func toAuctionListItemResponseList(list []*auctionv1.Auction) []auctionListItemResponse {
	result := make([]auctionListItemResponse, 0, len(list))
	for _, auction := range list {
		result = append(result, toAuctionListItemResponse(auction))
	}
	return result
}

func toStartAuctionResponse(auction *auctionv1.Auction) startAuctionResponse {
	if auction == nil {
		return startAuctionResponse{}
	}
	return startAuctionResponse{
		ID:        auction.GetId(),
		Status:    auction.GetStatus(),
		StartTime: documentTimeString(auction.GetStartTime()),
	}
}

func toFinishAuctionResponse(auction *auctionv1.Auction) finishAuctionResponse {
	if auction == nil {
		return finishAuctionResponse{}
	}
	return finishAuctionResponse{
		ID:           auction.GetId(),
		Status:       auction.GetStatus(),
		WinnerUserID: auction.WinnerUserId,
		DealPrice:    auction.DealPrice,
		EndTime:      documentTimeString(auction.GetEndTime()),
	}
}

func toCancelAuctionResponse(auction *auctionv1.Auction) cancelAuctionResponse {
	if auction == nil {
		return cancelAuctionResponse{}
	}
	return cancelAuctionResponse{
		ID:     auction.GetId(),
		Status: auction.GetStatus(),
	}
}

func toBidRecordResponse(record *auctionv1.BidRecord) bidRecordResponse {
	if record == nil {
		return bidRecordResponse{}
	}
	return bidRecordResponse{
		ID:        record.GetId(),
		AuctionID: record.GetAuctionId(),
		GoodsID:   record.GetGoodsId(),
		ShopID:    record.GetShopId(),
		UserID:    record.GetUserId(),
		BidPrice:  record.GetBidPrice(),
		BidTime:   documentTimeString(record.GetBidTime()),
		CreatedAt: documentTimeString(record.GetCreatedAt()),
		UpdatedAt: documentTimeString(record.GetUpdatedAt()),
	}
}

func toBidRecordResponseList(list []*auctionv1.BidRecord) []bidRecordResponse {
	result := make([]bidRecordResponse, 0, len(list))
	for _, record := range list {
		result = append(result, toBidRecordResponse(record))
	}
	return result
}
