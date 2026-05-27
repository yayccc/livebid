package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	"github.com/yayccc/livebid/pkg/auth"
	"google.golang.org/grpc"
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

type LiveHandler struct {
	liveClient liveServiceClient
	jwt        *auth.JWTManager
	rpcTimeout time.Duration
}

func NewLiveHandler(liveClient liveServiceClient, jwtManager *auth.JWTManager, rpcTimeout time.Duration) *LiveHandler {
	if rpcTimeout <= 0 {
		rpcTimeout = 3 * time.Second
	}
	return &LiveHandler{
		liveClient: liveClient,
		jwt:        jwtManager,
		rpcTimeout: rpcTimeout,
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

type srsCallbackRequest struct {
	Action   string `json:"action"`
	ClientID string `json:"client_id"`
	IP       string `json:"ip"`
	App      string `json:"app"`
	Stream   string `json:"stream"`
	Param    string `json:"param"`
}

func (h *LiveHandler) CreateLiveRoom(c *gin.Context) {
	shopID, ok := h.requireShopID(c)
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

func (h *LiveHandler) StartLive(c *gin.Context) {
	shopID, ok := h.requireShopID(c)
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
	shopID, ok := h.requireShopID(c)
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
	shopID, ok := h.requireShopID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.rpcTimeout)
	defer cancel()

	resp, err := h.liveClient.GetLiveStreamInfo(ctx, &livev1.GetLiveStreamInfoRequest{Id: id, ShopId: shopID})
	if err != nil {
		respondGRPCError(c, err)
		return
	}
	respondOK(c, gin.H{"stream_info": toLiveStreamInfoResponse(resp.GetStreamInfo())})
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

// 从 Authorization header 中解析 JWT token，验证后返回 shopID
func (h *LiveHandler) requireShopID(c *gin.Context) (int64, bool) {
	if h.jwt == nil {
		respondError(c, http.StatusUnauthorized, "missing auth")
		return 0, false
	}
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" || token == authHeader {
		respondError(c, http.StatusUnauthorized, "missing auth")
		return 0, false
	}
	claims, err := h.jwt.Verify(token)
	if err != nil {
		recordRequestError(c, err)
		respondError(c, http.StatusUnauthorized, "invalid token")
		return 0, false
	}
	shopID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || shopID <= 0 {
		if err != nil {
			recordRequestError(c, err)
		}
		respondError(c, http.StatusUnauthorized, "invalid token")
		return 0, false
	}
	return shopID, true
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		if err != nil {
			recordRequestError(c, err)
		}
		respondError(c, http.StatusBadRequest, "invalid request")
		return 0, false
	}
	return id, true
}

func parsePositiveQueryInt(c *gin.Context, name string, fallback int) int {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
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
