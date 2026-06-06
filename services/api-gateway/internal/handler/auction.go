package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type auctionServiceClient interface {
	CreateAuction(ctx context.Context, in *auctionv1.CreateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CreateAuctionResponse, error)
	GetAuction(ctx context.Context, in *auctionv1.GetAuctionRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionResponse, error)
	GetAuctionByGoods(ctx context.Context, in *auctionv1.GetAuctionByGoodsRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionByGoodsResponse, error)
	ListShopAuctions(ctx context.Context, in *auctionv1.ListShopAuctionsRequest, opts ...grpc.CallOption) (*auctionv1.ListShopAuctionsResponse, error)
	ListMerchantAuctions(ctx context.Context, in *auctionv1.ListMerchantAuctionsRequest, opts ...grpc.CallOption) (*auctionv1.ListMerchantAuctionsResponse, error)
	GetAuctionRuntime(ctx context.Context, in *auctionv1.GetAuctionRuntimeRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionRuntimeResponse, error)
	GetMerchantDashboardSummary(ctx context.Context, in *auctionv1.GetMerchantDashboardSummaryRequest, opts ...grpc.CallOption) (*auctionv1.GetMerchantDashboardSummaryResponse, error)
	UpdateAuction(ctx context.Context, in *auctionv1.UpdateAuctionRequest, opts ...grpc.CallOption) (*auctionv1.UpdateAuctionResponse, error)
	StartAuction(ctx context.Context, in *auctionv1.StartAuctionRequest, opts ...grpc.CallOption) (*auctionv1.StartAuctionResponse, error)
	FinishAuction(ctx context.Context, in *auctionv1.FinishAuctionRequest, opts ...grpc.CallOption) (*auctionv1.FinishAuctionResponse, error)
	CancelAuction(ctx context.Context, in *auctionv1.CancelAuctionRequest, opts ...grpc.CallOption) (*auctionv1.CancelAuctionResponse, error)
	DeleteAuction(ctx context.Context, in *auctionv1.DeleteAuctionRequest, opts ...grpc.CallOption) (*auctionv1.DeleteAuctionResponse, error)
	PlaceBid(ctx context.Context, in *auctionv1.PlaceBidRequest, opts ...grpc.CallOption) (*auctionv1.PlaceBidResponse, error)
	ListBidRecords(ctx context.Context, in *auctionv1.ListBidRecordsRequest, opts ...grpc.CallOption) (*auctionv1.ListBidRecordsResponse, error)
}

type AuctionHandler struct {
	auctionClient auctionServiceClient
	goodsClient   goodsServiceClient
	rpcTimeout    time.Duration
}

func NewAuctionHandler(auctionClient auctionServiceClient, rpcTimeout time.Duration) *AuctionHandler {
	return NewAuctionHandlerWithGoods(auctionClient, nil, rpcTimeout)
}

func NewAuctionHandlerWithGoods(auctionClient auctionServiceClient, goodsClient goodsServiceClient, rpcTimeout time.Duration) *AuctionHandler {
	return &AuctionHandler{
		auctionClient: auctionClient,
		goodsClient:   goodsClient,
		rpcTimeout:    normalizeRPCTimeout(rpcTimeout),
	}
}

type createAuctionRequest struct {
	GoodsID      int64  `json:"goods_id" form:"goods_id" binding:"required"`
	RoomID       int64  `json:"room_id" form:"room_id" binding:"required"`
	StartPrice   int64  `json:"start_price" form:"start_price" binding:"required"`
	BidIncrement int64  `json:"bid_increment" form:"bid_increment" binding:"required"`
	SealPrice    *int64 `json:"seal_price" form:"seal_price"`
	StartTime    string `json:"start_time" form:"start_time"`
	EndTime      string `json:"end_time" form:"end_time"`
}

type updateAuctionRequest struct {
	StartPrice   *int64 `json:"start_price" form:"start_price"`
	BidIncrement *int64 `json:"bid_increment" form:"bid_increment"`
	SealPrice    *int64 `json:"seal_price" form:"seal_price"`
	StartTime    string `json:"start_time" form:"start_time"`
	EndTime      string `json:"end_time" form:"end_time"`
}

type placeBidRequest struct {
	RoomID    int64  `json:"room_id" form:"room_id" binding:"required"`
	BidPrice  int64  `json:"bid_price" form:"bid_price" binding:"required"`
	RequestID string `json:"request_id" form:"request_id" binding:"required"`
}

type auctionResponse struct {
	ID           int64  `json:"id"`
	GoodsID      int64  `json:"goods_id"`
	ShopID       int64  `json:"shop_id"`
	RoomID       int64  `json:"room_id"`
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
	Version      int64  `json:"version"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type auctionListResponse struct {
	Total    int64                     `json:"total"`
	Page     int32                     `json:"page"`
	PageSize int32                     `json:"page_size"`
	List     []auctionListItemResponse `json:"list"`
}

type placeBidResponse struct {
	Accepted     bool   `json:"accepted"`
	CurrentPrice int64  `json:"current_price"`
	BidCount     int64  `json:"bid_count"`
	WinnerUserID int64  `json:"winner_user_id"`
	ServerTime   string `json:"server_time,omitempty"`
	ExpireAt     string `json:"expire_at,omitempty"`
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
	ID            int64  `json:"id"`
	GoodsID       int64  `json:"goods_id"`
	ShopID        int64  `json:"shop_id"`
	GoodsTitle    string `json:"goods_title,omitempty"`
	GoodsCoverURL string `json:"goods_cover_url,omitempty"`
	StartPrice    int64  `json:"start_price"`
	BidIncrement  int64  `json:"bid_increment,omitempty"`
	SealPrice     *int64 `json:"seal_price,omitempty"`
	CurrentPrice  int64  `json:"current_price"`
	BidCount      int64  `json:"bid_count"`
	Status        int32  `json:"status"`
	StartTime     string `json:"start_time,omitempty"`
	EndTime       string `json:"end_time,omitempty"`
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
	RoomID    int64  `json:"room_id"`
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

type auctionRuntimeResponse struct {
	AuctionID    int64  `json:"auction_id"`
	Status       int32  `json:"status"`
	CurrentPrice int64  `json:"current_price"`
	BidCount     int64  `json:"bid_count"`
	WinnerUserID *int64 `json:"winner_user_id,omitempty"`
	ServerTime   string `json:"server_time,omitempty"`
	ExpireAt     string `json:"expire_at,omitempty"`
	Version      int64  `json:"version"`
}

type merchantDashboardSummaryResponse struct {
	GoodsTotal       int64 `json:"goods_total"`
	GoodsOnSale      int64 `json:"goods_on_sale"`
	GoodsOffSale     int64 `json:"goods_off_sale"`
	AuctionTotal     int64 `json:"auction_total"`
	AuctionRunning   int64 `json:"auction_running"`
	AuctionPending   int64 `json:"auction_pending"`
	AuctionDeal      int64 `json:"auction_deal"`
	AuctionFailed    int64 `json:"auction_failed"`
	AuctionCancelled int64 `json:"auction_cancelled"`
	TodayDealAmount  int64 `json:"today_deal_amount"`
	TodayBidCount    int64 `json:"today_bid_count"`
}

func (h *AuctionHandler) Create(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	var req createAuctionRequest
	if err := c.ShouldBind(&req); err != nil {
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
		RoomId:       req.RoomID,
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
	items := toAuctionListItemResponseList(resp.GetList())
	if err := h.fillAuctionGoodsInfo(ctx, items); err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, auctionListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		List:     items,
	})
}

func (h *AuctionHandler) ListMerchant(c *gin.Context) {
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

	resp, err := h.auctionClient.ListMerchantAuctions(ctx, &auctionv1.ListMerchantAuctionsRequest{
		ShopId:   shopID,
		Status:   status,
		Page:     int32ValueOrZero(page),
		PageSize: int32ValueOrZero(pageSize),
		Keyword:  c.Query("keyword"),
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, auctionListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		List:     toMerchantAuctionListItemResponseList(resp.GetList()),
	})
}

func (h *AuctionHandler) GetRuntime(c *gin.Context) {
	auctionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.auctionClient.GetAuctionRuntime(ctx, &auctionv1.GetAuctionRuntimeRequest{AuctionId: auctionID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, toAuctionRuntimeResponse(resp.GetRuntime()))
}

func (h *AuctionHandler) DashboardSummary(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	auctionResp, err := h.auctionClient.GetMerchantDashboardSummary(ctx, &auctionv1.GetMerchantDashboardSummaryRequest{ShopId: shopID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	response := toMerchantDashboardSummaryResponse(auctionResp.GetSummary())
	if h.goodsClient != nil {
		goodsTotal, goodsOnSale, goodsOffSale, err := h.goodsDashboardCounts(ctx, shopID)
		if err != nil {
			respondGRPCError(c, err)
			return
		}
		response.GoodsTotal = goodsTotal
		response.GoodsOnSale = goodsOnSale
		response.GoodsOffSale = goodsOffSale
	}
	respondOK(c, response)
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
	if err := c.ShouldBind(&req); err != nil {
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
	respondOK(c, gin.H{})
}

func (h *AuctionHandler) PlaceBid(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	auctionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req placeBidRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "invalid request")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.auctionClient.PlaceBid(ctx, &auctionv1.PlaceBidRequest{
		AuctionId: auctionID,
		RoomId:    req.RoomID,
		UserId:    userID,
		BidPrice:  req.BidPrice,
		RequestId: req.RequestID,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, gin.H{"bid": toPlaceBidResponse(resp)})
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
		RoomID:       auction.GetRoomId(),
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
		BidIncrement: auction.GetBidIncrement(),
		SealPrice:    auction.SealPrice,
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

func toMerchantAuctionListItemResponse(auction *auctionv1.MerchantAuction) auctionListItemResponse {
	if auction == nil {
		return auctionListItemResponse{}
	}
	return auctionListItemResponse{
		ID:            auction.GetId(),
		GoodsID:       auction.GetGoodsId(),
		ShopID:        auction.GetShopId(),
		GoodsTitle:    auction.GetGoodsTitle(),
		GoodsCoverURL: auction.GetGoodsCoverUrl(),
		StartPrice:    auction.GetStartPrice(),
		BidIncrement:  auction.GetBidIncrement(),
		SealPrice:     auction.SealPrice,
		CurrentPrice:  auction.GetCurrentPrice(),
		BidCount:      auction.GetBidCount(),
		Status:        auction.GetStatus(),
		StartTime:     documentTimeString(auction.GetStartTime()),
		EndTime:       documentTimeString(auction.GetEndTime()),
	}
}

func toMerchantAuctionListItemResponseList(list []*auctionv1.MerchantAuction) []auctionListItemResponse {
	result := make([]auctionListItemResponse, 0, len(list))
	for _, auction := range list {
		result = append(result, toMerchantAuctionListItemResponse(auction))
	}
	return result
}

func toAuctionRuntimeResponse(runtime *auctionv1.AuctionRuntime) auctionRuntimeResponse {
	if runtime == nil {
		return auctionRuntimeResponse{}
	}
	return auctionRuntimeResponse{
		AuctionID:    runtime.GetAuctionId(),
		Status:       runtime.GetStatus(),
		CurrentPrice: runtime.GetCurrentPrice(),
		BidCount:     runtime.GetBidCount(),
		WinnerUserID: runtime.WinnerUserId,
		ServerTime:   documentTimeString(runtime.GetServerTime()),
		ExpireAt:     documentTimeString(runtime.GetExpireAt()),
		Version:      runtime.GetVersion(),
	}
}

func toMerchantDashboardSummaryResponse(summary *auctionv1.MerchantDashboardSummary) merchantDashboardSummaryResponse {
	if summary == nil {
		return merchantDashboardSummaryResponse{}
	}
	return merchantDashboardSummaryResponse{
		AuctionTotal:     summary.GetAuctionTotal(),
		AuctionRunning:   summary.GetAuctionRunning(),
		AuctionPending:   summary.GetAuctionPending(),
		AuctionDeal:      summary.GetAuctionDeal(),
		AuctionFailed:    summary.GetAuctionFailed(),
		AuctionCancelled: summary.GetAuctionCancelled(),
		TodayDealAmount:  summary.GetTodayDealAmount(),
		TodayBidCount:    summary.GetTodayBidCount(),
	}
}

func (h *AuctionHandler) fillAuctionGoodsInfo(ctx context.Context, items []auctionListItemResponse) error {
	if h.goodsClient == nil || len(items) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.GoodsID <= 0 {
			continue
		}
		if _, ok := seen[item.GoodsID]; ok {
			continue
		}
		seen[item.GoodsID] = struct{}{}
		ids = append(ids, item.GoodsID)
	}
	if len(ids) == 0 {
		return nil
	}
	resp, err := h.goodsClient.BatchGetGoods(ctx, &goodsv1.BatchGetGoodsRequest{Ids: ids})
	if err != nil {
		return err
	}
	goodsByID := make(map[int64]*goodsv1.Goods, len(resp.GetList()))
	for _, goods := range resp.GetList() {
		goodsByID[goods.GetId()] = goods
	}
	for index := range items {
		if goods := goodsByID[items[index].GoodsID]; goods != nil {
			items[index].GoodsTitle = goods.GetTitle()
			items[index].GoodsCoverURL = goods.GetCoverUrl()
		}
	}
	return nil
}

func (h *AuctionHandler) goodsDashboardCounts(ctx context.Context, shopID int64) (int64, int64, int64, error) {
	total, err := h.goodsCount(ctx, &goodsv1.ListGoodsRequest{ShopId: &shopID, Page: 1, PageSize: 1})
	if err != nil {
		return 0, 0, 0, err
	}
	onSaleStatus := int32(1)
	onSale, err := h.goodsCount(ctx, &goodsv1.ListGoodsRequest{ShopId: &shopID, Status: &onSaleStatus, Page: 1, PageSize: 1})
	if err != nil {
		return 0, 0, 0, err
	}
	offSaleStatus := int32(0)
	offSale, err := h.goodsCount(ctx, &goodsv1.ListGoodsRequest{ShopId: &shopID, Status: &offSaleStatus, Page: 1, PageSize: 1})
	if err != nil {
		return 0, 0, 0, err
	}
	return total, onSale, offSale, nil
}

func (h *AuctionHandler) goodsCount(ctx context.Context, req *goodsv1.ListGoodsRequest) (int64, error) {
	resp, err := h.goodsClient.ListGoods(ctx, req)
	if err != nil {
		return 0, err
	}
	return resp.GetTotal(), nil
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

func toPlaceBidResponse(resp *auctionv1.PlaceBidResponse) placeBidResponse {
	if resp == nil {
		return placeBidResponse{}
	}
	return placeBidResponse{
		Accepted:     resp.GetAccepted(),
		CurrentPrice: resp.GetCurrentPrice(),
		BidCount:     resp.GetBidCount(),
		WinnerUserID: resp.GetWinnerUserId(),
		ServerTime:   timestampString(resp.GetServerTime()),
		ExpireAt:     timestampString(resp.GetExpireAt()),
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
		RoomID:    record.GetRoomId(),
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
