package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	goodsv1 "github.com/yayccc/livebid/gen/proto/goods/v1"
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	shopv1 "github.com/yayccc/livebid/gen/proto/shop/v1"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/services/api-gateway/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type liveServiceClient interface {
	CreateLiveRoom(ctx context.Context, in *livev1.CreateLiveRoomRequest, opts ...grpc.CallOption) (*livev1.CreateLiveRoomResponse, error)
	GetLiveRoom(ctx context.Context, in *livev1.GetLiveRoomRequest, opts ...grpc.CallOption) (*livev1.GetLiveRoomResponse, error)
	ListLiveRooms(ctx context.Context, in *livev1.ListLiveRoomsRequest, opts ...grpc.CallOption) (*livev1.ListLiveRoomsResponse, error)
	StartLive(ctx context.Context, in *livev1.StartLiveRequest, opts ...grpc.CallOption) (*livev1.StartLiveResponse, error)
	EndLive(ctx context.Context, in *livev1.EndLiveRequest, opts ...grpc.CallOption) (*livev1.EndLiveResponse, error)
	GetLiveStreamInfo(ctx context.Context, in *livev1.GetLiveStreamInfoRequest, opts ...grpc.CallOption) (*livev1.GetLiveStreamInfoResponse, error)
	HandleSRSPublishCallback(ctx context.Context, in *livev1.HandleSRSPublishCallbackRequest, opts ...grpc.CallOption) (*livev1.HandleSRSPublishCallbackResponse, error)
	HandleSRSUnpublishCallback(ctx context.Context, in *livev1.HandleSRSUnpublishCallbackRequest, opts ...grpc.CallOption) (*livev1.HandleSRSUnpublishCallbackResponse, error)
}

type userLiveAuctionClient interface {
	GetCurrentAuctionByRoom(ctx context.Context, in *auctionv1.GetCurrentAuctionByRoomRequest, opts ...grpc.CallOption) (*auctionv1.GetCurrentAuctionByRoomResponse, error)
	BatchGetCurrentAuctionsByRoom(ctx context.Context, in *auctionv1.BatchGetCurrentAuctionsByRoomRequest, opts ...grpc.CallOption) (*auctionv1.BatchGetCurrentAuctionsByRoomResponse, error)
	GetAuctionRuntime(ctx context.Context, in *auctionv1.GetAuctionRuntimeRequest, opts ...grpc.CallOption) (*auctionv1.GetAuctionRuntimeResponse, error)
}

type userLiveShopClient interface {
	GetShop(ctx context.Context, in *shopv1.GetShopRequest, opts ...grpc.CallOption) (*shopv1.GetShopResponse, error)
	BatchGetPublicShops(ctx context.Context, in *shopv1.BatchGetPublicShopsRequest, opts ...grpc.CallOption) (*shopv1.BatchGetPublicShopsResponse, error)
}

type LiveHandler struct {
	liveClient    liveServiceClient
	shopClient    userLiveShopClient
	goodsClient   goodsServiceClient
	auctionClient userLiveAuctionClient
	userLive      config.UserLiveConfig
	rpcTimeout    time.Duration
}

func NewLiveHandler(liveClient liveServiceClient, rpcTimeout time.Duration) *LiveHandler {
	return NewLiveHandlerWithAggregates(liveClient, nil, nil, nil, config.UserLiveConfig{}, rpcTimeout)
}

func NewLiveHandlerWithAggregates(liveClient liveServiceClient, shopClient userLiveShopClient, goodsClient goodsServiceClient, auctionClient userLiveAuctionClient, userLive config.UserLiveConfig, rpcTimeout time.Duration) *LiveHandler {
	if userLive.WSURL == "" {
		userLive.WSURL = "/ws/live"
	}
	if userLive.HeartbeatIntervalSeconds <= 0 {
		userLive.HeartbeatIntervalSeconds = 15
	}
	return &LiveHandler{
		liveClient:    liveClient,
		shopClient:    shopClient,
		goodsClient:   goodsClient,
		auctionClient: auctionClient,
		userLive:      userLive,
		rpcTimeout:    normalizeRPCTimeout(rpcTimeout),
	}
}

type createLiveRoomRequest struct {
	Title       string `json:"title" form:"title" binding:"required"`
	Cover       string `json:"cover" form:"cover"`
	Description string `json:"description" form:"description"`
}

type liveRoomResponse struct {
	ID                int64  `json:"id"`
	ShopID            int64  `json:"shop_id"`
	Title             string `json:"title"`
	Cover             string `json:"cover,omitempty"`
	Description       string `json:"description,omitempty"`
	Status            string `json:"status"`
	MediaStreamStatus string `json:"media_stream_status"`
	ActualStartTime   string `json:"actual_start_time,omitempty"`
	ActualEndTime     string `json:"actual_end_time,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

type liveStreamInfoResponse struct {
	StreamName        string `json:"stream_name"`
	RtmpPushURL       string `json:"rtmp_push_url"`
	WebRTCPlayURL     string `json:"webrtc_play_url"`
	MediaStreamStatus string `json:"media_stream_status"`
}

type userLiveStreamInfoResponse struct {
	WebRTCPlayURL     string `json:"webrtc_play_url"`
	MediaStreamStatus string `json:"media_stream_status"`
	PlayStatus        string `json:"play_status"`
}

type userLiveFeedResponse struct {
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	Total      int64                 `json:"total"`
	HasMore    bool                  `json:"has_more"`
	NextCursor string                `json:"next_cursor"`
	List       []liveRoomFeedPreview `json:"list"`
}

type liveRoomFeedPreview struct {
	RoomID             int64                 `json:"room_id"`
	Title              string                `json:"title"`
	Cover              string                `json:"cover,omitempty"`
	Status             int32                 `json:"status"`
	StatusText         string                `json:"status_text"`
	MediaStreamStatus  int32                 `json:"media_stream_status"`
	MediaStreamText    string                `json:"media_stream_status_text"`
	Shop               userLiveShopResponse  `json:"shop"`
	PreviewStream      userLivePreviewStream `json:"preview_stream"`
	Stats              userLiveStatsResponse `json:"stats"`
	CurrentAuctionHint *userLiveAuctionHint  `json:"current_auction_hint"`
}

type userLiveShopResponse struct {
	ShopID      int64  `json:"shop_id"`
	ShopName    string `json:"shop_name"`
	Logo        string `json:"logo,omitempty"`
	Description string `json:"description,omitempty"`
}

type userLivePreviewStream struct {
	WebRTCPlayURL string `json:"webrtc_play_url"`
	PlayStatus    string `json:"play_status"`
}

type userLiveStatsResponse struct {
	OnlineUserCount int64 `json:"online_user_count"`
	Heat            int64 `json:"heat"`
}

type userLiveAuctionHint struct {
	AuctionID     int64  `json:"auction_id"`
	GoodsID       int64  `json:"goods_id"`
	GoodsTitle    string `json:"goods_title,omitempty"`
	GoodsCoverURL string `json:"goods_cover_url,omitempty"`
	CurrentPrice  int64  `json:"current_price"`
	Status        int32  `json:"status"`
}

type userLiveEntryRoomResponse struct {
	RoomID                int64  `json:"room_id"`
	Title                 string `json:"title"`
	Cover                 string `json:"cover,omitempty"`
	Description           string `json:"description,omitempty"`
	Status                int32  `json:"status"`
	StatusText            string `json:"status_text"`
	MediaStreamStatus     int32  `json:"media_stream_status"`
	MediaStreamStatusText string `json:"media_stream_status_text"`
	ActualStartTime       int64  `json:"actual_start_time,omitempty"`
}

type userLiveViewerResponse struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	UserID     int64  `json:"user_id"`
	CanBid     bool   `json:"can_bid"`
	EnterState string `json:"enter_state,omitempty"`
}

type userLiveAuctionResponse struct {
	AuctionID         int64  `json:"auction_id"`
	RoomID            int64  `json:"room_id"`
	GoodsID           int64  `json:"goods_id"`
	Status            int32  `json:"status"`
	StatusText        string `json:"status_text"`
	StartPrice        int64  `json:"start_price"`
	BidIncrement      int64  `json:"bid_increment"`
	SealPrice         *int64 `json:"seal_price,omitempty"`
	CurrentPrice      int64  `json:"current_price"`
	NextBidPrice      int64  `json:"next_bid_price"`
	BidCount          int64  `json:"bid_count"`
	WinnerUserID      *int64 `json:"winner_user_id,omitempty"`
	WinnerDisplayName string `json:"winner_display_name,omitempty"`
	StartTime         int64  `json:"start_time,omitempty"`
	EndTime           int64  `json:"end_time,omitempty"`
}

type userLiveGoodsResponse struct {
	GoodsID     int64  `json:"goods_id"`
	ShopID      int64  `json:"shop_id,omitempty"`
	Title       string `json:"title"`
	CoverURL    string `json:"cover_url,omitempty"`
	Description string `json:"description,omitempty"`
	Status      int32  `json:"status,omitempty"`
}

type userLiveRuntimeResponse struct {
	AuctionID         int64  `json:"auction_id"`
	Status            int32  `json:"status"`
	CurrentPrice      int64  `json:"current_price"`
	NextBidPrice      int64  `json:"next_bid_price"`
	BidCount          int64  `json:"bid_count"`
	WinnerUserID      *int64 `json:"winner_user_id,omitempty"`
	WinnerDisplayName string `json:"winner_display_name,omitempty"`
	ServerTime        int64  `json:"server_time"`
	ExpireAt          int64  `json:"expire_at,omitempty"`
	Version           int64  `json:"version"`
}

type userLiveWSResponse struct {
	URL                      string `json:"url"`
	RoomID                   int64  `json:"room_id"`
	TokenRequiredForBid      bool   `json:"token_required_for_bid"`
	HeartbeatIntervalSeconds int    `json:"heartbeat_interval_seconds"`
}

type srsCallbackRequest struct {
	Action   string `json:"action"`
	ClientID string `json:"client_id"`
	IP       string `json:"ip"`
	App      string `json:"app"`
	Stream   string `json:"stream"`
	Param    string `json:"param"`
}

func (h *LiveHandler) CreateLiveRoom(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}

	var req createLiveRoomRequest
	if err := c.ShouldBind(&req); err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusBadRequest, "invalid request")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.liveClient.CreateLiveRoom(ctx, &livev1.CreateLiveRoomRequest{
		ShopId:      shopID,
		Title:       req.Title,
		Cover:       req.Cover,
		Description: req.Description,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	respondOK(c, gin.H{
		"live_room":           toLiveRoomResponse(resp.GetLiveRoom()),
		"initial_stream_code": resp.GetInitialStreamCode(),
		"rtmp_push_url":       resp.GetRtmpPushUrl(),
		"webrtc_play_url":     resp.GetWebrtcPlayUrl(),
	})
}

func (h *LiveHandler) GetLiveRoom(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.liveClient.GetLiveRoom(ctx, &livev1.GetLiveRoomRequest{Id: id})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, gin.H{"live_room": toLiveRoomResponse(resp.GetLiveRoom())})
}

func (h *LiveHandler) ListLiveRooms(c *gin.Context) {
	page := parsePositiveQueryInt(c, "page", 1)
	pageSize := parsePositiveQueryInt(c, "page_size", 20)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.liveClient.ListLiveRooms(ctx, &livev1.ListLiveRoomsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	rooms := make([]liveRoomResponse, 0, len(resp.GetLiveRooms()))
	for _, room := range resp.GetLiveRooms() {
		rooms = append(rooms, toLiveRoomResponse(room))
	}
	respondOK(c, gin.H{
		"live_rooms": rooms,
		"total":      resp.GetTotal(),
	})
}

func (h *LiveHandler) ListMerchantLiveRooms(c *gin.Context) {
	shopID, ok := currentShopID(c)
	if !ok {
		return
	}
	page := parsePositiveQueryInt(c, "page", 1)
	pageSize := parsePositiveQueryInt(c, "page_size", 10)
	if pageSize > 100 {
		pageSize = 100
	}
	statusFilter, ok := parseLiveRoomStatusQuery(c, "status")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.liveClient.ListLiveRooms(ctx, &livev1.ListLiveRoomsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		ShopId:   &shopID,
		Status:   statusFilter,
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	rooms := make([]liveRoomResponse, 0, len(resp.GetLiveRooms()))
	for _, room := range resp.GetLiveRooms() {
		rooms = append(rooms, toLiveRoomResponse(room))
	}
	respondOK(c, gin.H{
		"total":     resp.GetTotal(),
		"page":      page,
		"page_size": pageSize,
		"list":      rooms,
	})
}

func (h *LiveHandler) GetUserLiveFeed(c *gin.Context) {
	page := parsePositiveQueryInt(c, "page", 1)
	pageSize := parsePositiveQueryInt(c, "page_size", 10)
	if pageSize > 20 {
		pageSize = 20
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.liveClient.ListLiveRooms(ctx, &livev1.ListLiveRoomsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	rooms := resp.GetLiveRooms()
	shopsByID, err := h.batchPublicShops(ctx, liveRoomShopIDs(rooms))
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	auctionsByRoomID, err := h.batchCurrentAuctionsByRoom(ctx, liveRoomIDs(rooms))
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	goodsByID, err := h.batchGoods(ctx, auctionGoodsIDsFromProtoMap(auctionsByRoomID))
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	list := make([]liveRoomFeedPreview, 0, len(rooms))
	for _, room := range rooms {
		streamResp, err := h.liveClient.GetLiveStreamInfo(ctx, &livev1.GetLiveStreamInfoRequest{Id: room.GetId()})
		if err != nil {
			respondGRPCError(c, err)
			return
		}
		auction := auctionsByRoomID[room.GetId()]
		var hint *userLiveAuctionHint
		if auction != nil {
			hint = toUserLiveAuctionHint(auction, goodsByID[auction.GetGoodsId()])
		}
		list = append(list, toLiveRoomFeedPreview(room, streamResp.GetStreamInfo(), shopsByID[room.GetShopId()], hint))
	}

	respondOK(c, userLiveFeedResponse{
		Page:     page,
		PageSize: pageSize,
		Total:    resp.GetTotal(),
		HasMore:  int64(page*pageSize) < resp.GetTotal(),
		List:     list,
	})
}

func (h *LiveHandler) StartLive(c *gin.Context) {
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

	resp, err := h.liveClient.StartLive(ctx, &livev1.StartLiveRequest{Id: id, ShopId: shopID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, gin.H{"live_room": toLiveRoomResponse(resp.GetLiveRoom())})
}

func (h *LiveHandler) EndLive(c *gin.Context) {
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

	resp, err := h.liveClient.EndLive(ctx, &livev1.EndLiveRequest{Id: id, ShopId: shopID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, gin.H{"live_room": toLiveRoomResponse(resp.GetLiveRoom())})
}

func (h *LiveHandler) GetLiveStreamInfo(c *gin.Context) {
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

	roomResp, err := h.liveClient.GetLiveRoom(ctx, &livev1.GetLiveRoomRequest{Id: id})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	if roomResp.GetLiveRoom().GetShopId() != shopID {
		respondError(c, http.StatusForbidden, "无权查看该直播间推流信息")
		return
	}

	resp, err := h.liveClient.GetLiveStreamInfo(ctx, &livev1.GetLiveStreamInfoRequest{Id: id})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, gin.H{"stream_info": toLiveStreamInfoResponse(resp.GetStreamInfo())})
}

func (h *LiveHandler) GetUserLivePreview(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	roomResp, err := h.liveClient.GetLiveRoom(ctx, &livev1.GetLiveRoomRequest{Id: id})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	resp, err := h.liveClient.GetLiveStreamInfo(ctx, &livev1.GetLiveStreamInfoRequest{Id: id})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, gin.H{
		"room":   toUserLiveEntryRoomResponse(roomResp.GetLiveRoom()),
		"stream": toUserLiveStreamInfoResponse(resp.GetStreamInfo()),
	})
}

func (h *LiveHandler) GetUserLiveEntry(c *gin.Context) {
	roomID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	roomResp, err := h.liveClient.GetLiveRoom(ctx, &livev1.GetLiveRoomRequest{Id: roomID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	room := roomResp.GetLiveRoom()
	if room.GetStatus() != livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING {
		respondError(c, http.StatusBadRequest, "直播间当前未开播")
		return
	}

	streamResp, err := h.liveClient.GetLiveStreamInfo(ctx, &livev1.GetLiveStreamInfoRequest{Id: roomID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	shopResp, err := h.shop(ctx, room.GetShopId())
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	auction, err := h.currentAuctionByRoom(ctx, roomID)
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	goods, runtime, err := h.auctionSnapshotParts(ctx, auction)
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	data := gin.H{
		"room":            toUserLiveEntryRoomResponse(room),
		"shop":            toUserLiveShopResponse(shopResp),
		"stream":          toUserLiveStreamInfoResponse(streamResp.GetStreamInfo()),
		"viewer":          toUserLiveViewerResponse(c, "entered"),
		"current_auction": nil,
		"goods":           nil,
		"runtime":         nil,
		"ws":              h.userLiveWS(roomID),
	}
	if auction != nil {
		data["current_auction"] = toUserLiveAuctionResponse(auction, runtime)
		data["goods"] = toUserLiveGoodsResponse(goods)
		data["runtime"] = toUserLiveRuntimeResponse(auction, runtime)
	}
	respondOK(c, data)
}

func (h *LiveHandler) GetUserLiveAuctionSnapshot(c *gin.Context) {
	roomID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	if _, err := h.liveClient.GetLiveRoom(ctx, &livev1.GetLiveRoomRequest{Id: roomID}); err != nil {
		respondGRPCError(c, err)
		return
	}
	auction, err := h.currentAuctionByRoom(ctx, roomID)
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	goods, runtime, err := h.auctionSnapshotParts(ctx, auction)
	if err != nil {
		respondGRPCError(c, err)
		return
	}

	data := gin.H{
		"room_id":         roomID,
		"viewer":          toUserLiveViewerResponse(c, ""),
		"current_auction": nil,
		"goods":           nil,
		"runtime":         nil,
	}
	if auction != nil {
		data["current_auction"] = toUserLiveAuctionResponse(auction, runtime)
		data["goods"] = toUserLiveGoodsResponse(goods)
		data["runtime"] = toUserLiveRuntimeResponse(auction, runtime)
	}
	respondOK(c, data)
}

func (h *LiveHandler) HandleSRSPublishCallback(c *gin.Context) {
	var req srsCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		c.String(http.StatusForbidden, "1")
		return
	}

	streamCode := streamCodeFromSRSParam(req.Param)
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	_, err := h.liveClient.HandleSRSPublishCallback(ctx, &livev1.HandleSRSPublishCallbackRequest{
		StreamName: req.Stream,
		StreamCode: streamCode,
		ClientId:   req.ClientID,
		Ip:         req.IP,
	})
	if err != nil {
		recordRequestError(c, err)
		c.String(http.StatusForbidden, "1")
		return
	}
	c.String(http.StatusOK, "0")
}

func (h *LiveHandler) HandleSRSUnpublishCallback(c *gin.Context) {
	var req srsCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		recordRequestError(c, err)
		c.String(http.StatusOK, "0")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	if _, err := h.liveClient.HandleSRSUnpublishCallback(ctx, &livev1.HandleSRSUnpublishCallbackRequest{
		StreamName: req.Stream,
		ClientId:   req.ClientID,
		Ip:         req.IP,
	}); err != nil {
		recordRequestError(c, err)
	}
	c.String(http.StatusOK, "0")
}

func (h *LiveHandler) batchPublicShops(ctx context.Context, ids []int64) (map[int64]*shopv1.Shop, error) {
	result := make(map[int64]*shopv1.Shop)
	ids = uniquePositiveIDs(ids)
	if len(ids) == 0 || h.shopClient == nil {
		return result, nil
	}
	resp, err := h.shopClient.BatchGetPublicShops(ctx, &shopv1.BatchGetPublicShopsRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	for _, shop := range resp.GetList() {
		if shop.GetId() > 0 {
			result[shop.GetId()] = shop
		}
	}
	return result, nil
}

func (h *LiveHandler) shop(ctx context.Context, id int64) (*shopv1.Shop, error) {
	if id <= 0 || h.shopClient == nil {
		return nil, nil
	}
	resp, err := h.shopClient.GetShop(ctx, &shopv1.GetShopRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return resp.GetShop(), nil
}

func (h *LiveHandler) batchCurrentAuctionsByRoom(ctx context.Context, roomIDs []int64) (map[int64]*auctionv1.Auction, error) {
	result := make(map[int64]*auctionv1.Auction)
	roomIDs = uniquePositiveIDs(roomIDs)
	if len(roomIDs) == 0 || h.auctionClient == nil {
		return result, nil
	}
	resp, err := h.auctionClient.BatchGetCurrentAuctionsByRoom(ctx, &auctionv1.BatchGetCurrentAuctionsByRoomRequest{RoomIds: roomIDs})
	if err != nil {
		return nil, err
	}
	for _, auction := range resp.GetList() {
		if auction.GetRoomId() > 0 {
			result[auction.GetRoomId()] = auction
		}
	}
	return result, nil
}

func (h *LiveHandler) currentAuctionByRoom(ctx context.Context, roomID int64) (*auctionv1.Auction, error) {
	if roomID <= 0 || h.auctionClient == nil {
		return nil, nil
	}
	resp, err := h.auctionClient.GetCurrentAuctionByRoom(ctx, &auctionv1.GetCurrentAuctionByRoomRequest{RoomId: roomID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	return resp.GetAuction(), nil
}

func (h *LiveHandler) batchGoods(ctx context.Context, ids []int64) (map[int64]*goodsv1.Goods, error) {
	result := make(map[int64]*goodsv1.Goods)
	ids = uniquePositiveIDs(ids)
	if len(ids) == 0 || h.goodsClient == nil {
		return result, nil
	}
	resp, err := h.goodsClient.BatchGetGoods(ctx, &goodsv1.BatchGetGoodsRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	for _, goods := range resp.GetList() {
		if goods.GetId() > 0 {
			result[goods.GetId()] = goods
		}
	}
	return result, nil
}

func (h *LiveHandler) auctionSnapshotParts(ctx context.Context, auction *auctionv1.Auction) (*goodsv1.Goods, *auctionv1.AuctionRuntime, error) {
	if auction == nil {
		return nil, nil, nil
	}
	var goods *goodsv1.Goods
	if h.goodsClient != nil && auction.GetGoodsId() > 0 {
		goodsResp, err := h.goodsClient.GetGoods(ctx, &goodsv1.GetGoodsRequest{Id: auction.GetGoodsId()})
		if err != nil {
			return nil, nil, err
		}
		goods = goodsResp.GetGoods()
	}

	var runtime *auctionv1.AuctionRuntime
	if h.auctionClient != nil && auction.GetId() > 0 {
		runtimeResp, err := h.auctionClient.GetAuctionRuntime(ctx, &auctionv1.GetAuctionRuntimeRequest{AuctionId: auction.GetId()})
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return nil, nil, err
			}
		} else {
			runtime = runtimeResp.GetRuntime()
		}
	}
	return goods, runtime, nil
}

func liveRoomIDs(rooms []*livev1.LiveRoom) []int64 {
	ids := make([]int64, 0, len(rooms))
	for _, room := range rooms {
		if room.GetId() > 0 {
			ids = append(ids, room.GetId())
		}
	}
	return uniquePositiveIDs(ids)
}

func liveRoomShopIDs(rooms []*livev1.LiveRoom) []int64 {
	ids := make([]int64, 0, len(rooms))
	for _, room := range rooms {
		if room.GetShopId() > 0 {
			ids = append(ids, room.GetShopId())
		}
	}
	return uniquePositiveIDs(ids)
}

func auctionGoodsIDsFromProtoMap(auctionsByRoomID map[int64]*auctionv1.Auction) []int64 {
	ids := make([]int64, 0, len(auctionsByRoomID))
	for _, auction := range auctionsByRoomID {
		if auction.GetGoodsId() > 0 {
			ids = append(ids, auction.GetGoodsId())
		}
	}
	return uniquePositiveIDs(ids)
}

func uniquePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}

func streamCodeFromSRSParam(param string) string {
	values := strings.TrimPrefix(strings.TrimSpace(param), "?")
	for _, part := range strings.Split(values, "&") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch key {
		case "token", "stream_code", "code":
			return value
		}
	}
	return ""
}

func parseLiveRoomStatusQuery(c *gin.Context, key string) (*livev1.LiveRoomStatus, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	var statusFilter livev1.LiveRoomStatus
	switch raw {
	case "not_live":
		statusFilter = livev1.LiveRoomStatus_LIVE_ROOM_STATUS_NOT_LIVE
	case "living":
		statusFilter = livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING
	default:
		respondError(c, http.StatusBadRequest, "直播间状态无效，请使用 not_live 或 living")
		return nil, false
	}
	return &statusFilter, true
}

func toLiveRoomResponse(room *livev1.LiveRoom) liveRoomResponse {
	if room == nil {
		return liveRoomResponse{}
	}
	return liveRoomResponse{
		ID:                room.GetId(),
		ShopID:            room.GetShopId(),
		Title:             room.GetTitle(),
		Cover:             room.GetCover(),
		Description:       room.GetDescription(),
		Status:            liveRoomStatusString(room.GetStatus()),
		MediaStreamStatus: mediaStreamStatusString(room.GetMediaStreamStatus()),
		ActualStartTime:   timestampString(room.GetActualStartTime()),
		ActualEndTime:     timestampString(room.GetActualEndTime()),
		CreatedAt:         timestampString(room.GetCreatedAt()),
		UpdatedAt:         timestampString(room.GetUpdatedAt()),
	}
}

func toLiveStreamInfoResponse(info *livev1.LiveStreamInfo) liveStreamInfoResponse {
	if info == nil {
		return liveStreamInfoResponse{}
	}
	return liveStreamInfoResponse{
		StreamName:        info.GetStreamName(),
		RtmpPushURL:       info.GetRtmpPushUrl(),
		WebRTCPlayURL:     info.GetWebrtcPlayUrl(),
		MediaStreamStatus: mediaStreamStatusString(info.GetMediaStreamStatus()),
	}
}

func toLiveRoomFeedPreview(room *livev1.LiveRoom, stream *livev1.LiveStreamInfo, shop *shopv1.Shop, hint *userLiveAuctionHint) liveRoomFeedPreview {
	if room == nil {
		return liveRoomFeedPreview{}
	}
	return liveRoomFeedPreview{
		RoomID:             room.GetId(),
		Title:              room.GetTitle(),
		Cover:              room.GetCover(),
		Status:             int32(room.GetStatus()),
		StatusText:         liveRoomStatusText(room.GetStatus()),
		MediaStreamStatus:  int32(room.GetMediaStreamStatus()),
		MediaStreamText:    mediaStreamStatusText(room.GetMediaStreamStatus()),
		Shop:               toUserLiveShopResponse(shop),
		PreviewStream:      toUserLivePreviewStream(stream),
		Stats:              userLiveStatsResponse{},
		CurrentAuctionHint: hint,
	}
}

func toUserLivePreviewStream(info *livev1.LiveStreamInfo) userLivePreviewStream {
	if info == nil {
		return userLivePreviewStream{PlayStatus: "unavailable"}
	}
	playStatus := "unavailable"
	if info.GetWebrtcPlayUrl() != "" && info.GetMediaStreamStatus() == livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE {
		playStatus = "available"
	}
	return userLivePreviewStream{
		WebRTCPlayURL: info.GetWebrtcPlayUrl(),
		PlayStatus:    playStatus,
	}
}

func toUserLiveStreamInfoResponse(info *livev1.LiveStreamInfo) userLiveStreamInfoResponse {
	if info == nil {
		return userLiveStreamInfoResponse{PlayStatus: "unavailable"}
	}
	playStatus := "unavailable"
	if info.GetWebrtcPlayUrl() != "" && info.GetMediaStreamStatus() == livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE {
		playStatus = "available"
	}
	return userLiveStreamInfoResponse{
		WebRTCPlayURL:     info.GetWebrtcPlayUrl(),
		MediaStreamStatus: mediaStreamStatusString(info.GetMediaStreamStatus()),
		PlayStatus:        playStatus,
	}
}

func toUserLiveEntryRoomResponse(room *livev1.LiveRoom) userLiveEntryRoomResponse {
	if room == nil {
		return userLiveEntryRoomResponse{}
	}
	return userLiveEntryRoomResponse{
		RoomID:                room.GetId(),
		Title:                 room.GetTitle(),
		Cover:                 room.GetCover(),
		Description:           room.GetDescription(),
		Status:                int32(room.GetStatus()),
		StatusText:            liveRoomStatusText(room.GetStatus()),
		MediaStreamStatus:     int32(room.GetMediaStreamStatus()),
		MediaStreamStatusText: mediaStreamStatusText(room.GetMediaStreamStatus()),
		ActualStartTime:       timestampMillis(room.GetActualStartTime()),
	}
}

func toUserLiveShopResponse(shop *shopv1.Shop) userLiveShopResponse {
	if shop == nil {
		return userLiveShopResponse{}
	}
	return userLiveShopResponse{
		ShopID:      shop.GetId(),
		ShopName:    shop.GetShopName(),
		Logo:        shop.GetLogo(),
		Description: shop.GetDescription(),
	}
}

func toUserLiveViewerResponse(c *gin.Context, enterState string) userLiveViewerResponse {
	userID, ok := identity.UserID(c.Request.Context())
	viewer := userLiveViewerResponse{
		IsLoggedIn: ok,
		UserID:     userID,
		CanBid:     ok,
		EnterState: enterState,
	}
	if !ok {
		viewer.UserID = 0
	}
	return viewer
}

func toUserLiveAuctionHint(auction *auctionv1.Auction, goods *goodsv1.Goods) *userLiveAuctionHint {
	if auction == nil {
		return nil
	}
	hint := &userLiveAuctionHint{
		AuctionID:    auction.GetId(),
		GoodsID:      auction.GetGoodsId(),
		CurrentPrice: auction.GetCurrentPrice(),
		Status:       auction.GetStatus(),
	}
	if goods != nil {
		hint.GoodsTitle = goods.GetTitle()
		hint.GoodsCoverURL = goods.GetCoverUrl()
	}
	return hint
}

func toUserLiveAuctionResponse(auction *auctionv1.Auction, runtime *auctionv1.AuctionRuntime) *userLiveAuctionResponse {
	if auction == nil {
		return nil
	}
	currentPrice := auction.GetCurrentPrice()
	bidCount := auction.GetBidCount()
	winnerUserID := auction.WinnerUserId
	if runtime != nil {
		currentPrice = runtime.GetCurrentPrice()
		bidCount = runtime.GetBidCount()
		winnerUserID = runtime.WinnerUserId
	}
	return &userLiveAuctionResponse{
		AuctionID:         auction.GetId(),
		RoomID:            auction.GetRoomId(),
		GoodsID:           auction.GetGoodsId(),
		Status:            auction.GetStatus(),
		StatusText:        auctionStatusText(auction.GetStatus()),
		StartPrice:        auction.GetStartPrice(),
		BidIncrement:      auction.GetBidIncrement(),
		SealPrice:         auction.SealPrice,
		CurrentPrice:      currentPrice,
		NextBidPrice:      nextBidPrice(currentPrice, auction.GetBidIncrement(), auction.SealPrice),
		BidCount:          bidCount,
		WinnerUserID:      winnerUserID,
		WinnerDisplayName: winnerDisplayName(winnerUserID),
		StartTime:         timestampMillis(auction.GetStartTime()),
		EndTime:           timestampMillis(auction.GetEndTime()),
	}
}

func toUserLiveGoodsResponse(goods *goodsv1.Goods) *userLiveGoodsResponse {
	if goods == nil {
		return nil
	}
	return &userLiveGoodsResponse{
		GoodsID:     goods.GetId(),
		ShopID:      goods.GetShopId(),
		Title:       goods.GetTitle(),
		CoverURL:    goods.GetCoverUrl(),
		Description: goods.GetDescription(),
		Status:      goods.GetStatus(),
	}
}

func toUserLiveRuntimeResponse(auction *auctionv1.Auction, runtime *auctionv1.AuctionRuntime) *userLiveRuntimeResponse {
	if auction == nil {
		return nil
	}
	if runtime == nil {
		return &userLiveRuntimeResponse{
			AuctionID:         auction.GetId(),
			Status:            auction.GetStatus(),
			CurrentPrice:      auction.GetCurrentPrice(),
			NextBidPrice:      nextBidPrice(auction.GetCurrentPrice(), auction.GetBidIncrement(), auction.SealPrice),
			BidCount:          auction.GetBidCount(),
			WinnerUserID:      auction.WinnerUserId,
			WinnerDisplayName: winnerDisplayName(auction.WinnerUserId),
			ServerTime:        time.Now().UnixMilli(),
			ExpireAt:          timestampMillis(auction.GetEndTime()),
			Version:           auction.GetVersion(),
		}
	}
	return &userLiveRuntimeResponse{
		AuctionID:         runtime.GetAuctionId(),
		Status:            runtime.GetStatus(),
		CurrentPrice:      runtime.GetCurrentPrice(),
		NextBidPrice:      nextBidPrice(runtime.GetCurrentPrice(), auction.GetBidIncrement(), auction.SealPrice),
		BidCount:          runtime.GetBidCount(),
		WinnerUserID:      runtime.WinnerUserId,
		WinnerDisplayName: winnerDisplayName(runtime.WinnerUserId),
		ServerTime:        timestampMillis(runtime.GetServerTime()),
		ExpireAt:          timestampMillis(runtime.GetExpireAt()),
		Version:           runtime.GetVersion(),
	}
}

func (h *LiveHandler) userLiveWS(roomID int64) userLiveWSResponse {
	return userLiveWSResponse{
		URL:                      h.userLive.WSURL,
		RoomID:                   roomID,
		TokenRequiredForBid:      true,
		HeartbeatIntervalSeconds: h.userLive.HeartbeatIntervalSeconds,
	}
}

func liveRoomStatusString(value livev1.LiveRoomStatus) string {
	switch value {
	case livev1.LiveRoomStatus_LIVE_ROOM_STATUS_NOT_LIVE:
		return "not_live"
	case livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING:
		return "living"
	default:
		return "unspecified"
	}
}

func liveRoomStatusText(value livev1.LiveRoomStatus) string {
	switch value {
	case livev1.LiveRoomStatus_LIVE_ROOM_STATUS_NOT_LIVE:
		return "未开播"
	case livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING:
		return "直播中"
	default:
		return "未知"
	}
}

func mediaStreamStatusString(value livev1.MediaStreamStatus) string {
	switch value {
	case livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_OFFLINE:
		return "offline"
	case livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE:
		return "online"
	default:
		return "unspecified"
	}
}

func mediaStreamStatusText(value livev1.MediaStreamStatus) string {
	switch value {
	case livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_OFFLINE:
		return "未推流"
	case livev1.MediaStreamStatus_MEDIA_STREAM_STATUS_ONLINE:
		return "推流中"
	default:
		return "未知"
	}
}

func auctionStatusText(value int32) string {
	switch value {
	case 0:
		return "待开始"
	case 1:
		return "竞拍中"
	case 2:
		return "已成交"
	case 3:
		return "已流拍"
	case 4:
		return "已取消"
	default:
		return "未知"
	}
}

func timestampMillis(ts *timestamppb.Timestamp) int64 {
	if ts == nil {
		return 0
	}
	return ts.AsTime().UnixMilli()
}

func nextBidPrice(currentPrice int64, bidIncrement int64, sealPrice *int64) int64 {
	next := currentPrice + bidIncrement
	if sealPrice != nil && *sealPrice > 0 && next > *sealPrice {
		return *sealPrice
	}
	return next
}

func winnerDisplayName(userID *int64) string {
	if userID == nil || *userID <= 0 {
		return ""
	}
	return "用户" + strconv.FormatInt(*userID, 10)
}
