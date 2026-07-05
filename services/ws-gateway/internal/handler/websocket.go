package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	auctionv1 "github.com/yayccc/livebid/gen/proto/auction/v1"
	livev1 "github.com/yayccc/livebid/gen/proto/live/v1"
	userv1 "github.com/yayccc/livebid/gen/proto/user/v1"
	"github.com/yayccc/livebid/pkg/auth"
	"github.com/yayccc/livebid/pkg/identity"
	"github.com/yayccc/livebid/pkg/idgen"
	"github.com/yayccc/livebid/services/ws-gateway/internal/config"
	"github.com/yayccc/livebid/services/ws-gateway/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type WebSocketHandler struct {
	cfg        config.Config
	hub        *Hub
	online     repository.OnlineStore
	danmaku    repository.DanmakuStore
	auction    auctionv1.AuctionServiceClient
	live       livev1.LiveServiceClient
	user       userv1.UserServiceClient
	jwt        *auth.JWTManager
	ids        *idgen.Generator
	upgrader   websocket.Upgrader
	log        *zap.Logger
	rpcTimeout time.Duration

	onlineMu       sync.Mutex
	pendingOnline  map[int64]repository.OnlineEvent
	lastOnlineSent map[int64]repository.OnlineStats
	danmakuMu      sync.Mutex
	seenDanmaku    map[string]int64
}

type WebSocketHandlerOptions struct {
	Config     config.Config
	Hub        *Hub
	Online     repository.OnlineStore
	Danmaku    repository.DanmakuStore
	Auction    auctionv1.AuctionServiceClient
	Live       livev1.LiveServiceClient
	User       userv1.UserServiceClient
	JWT        *auth.JWTManager
	IDs        *idgen.Generator
	Log        *zap.Logger
	RPCTimeout time.Duration
}

func NewWebSocketHandler(opts WebSocketHandlerOptions) *WebSocketHandler {
	hub := opts.Hub
	if hub == nil {
		hub = NewHub()
	}
	ids := opts.IDs
	if ids == nil {
		ids = idgen.New(8)
	}
	rpcTimeout := opts.RPCTimeout
	if rpcTimeout <= 0 {
		rpcTimeout = 3 * time.Second
	}
	allowOrigins := parseAllowOrigins(opts.Config.WebSocket.AllowOrigins)
	return &WebSocketHandler{
		cfg:            opts.Config,
		hub:            hub,
		online:         opts.Online,
		danmaku:        opts.Danmaku,
		auction:        opts.Auction,
		live:           opts.Live,
		user:           opts.User,
		jwt:            opts.JWT,
		ids:            ids,
		log:            opts.Log,
		rpcTimeout:     rpcTimeout,
		pendingOnline:  make(map[int64]repository.OnlineEvent),
		lastOnlineSent: make(map[int64]repository.OnlineStats),
		seenDanmaku:    make(map[string]int64),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return originAllowed(r.Header.Get("Origin"), allowOrigins)
			},
		},
	}
}

func (h *WebSocketHandler) Hub() *Hub {
	return h.hub
}

func (h *WebSocketHandler) ServeLive(c *gin.Context) {
	roomID, err := strconv.ParseInt(strings.TrimSpace(c.Query("room_id")), 10, 64)
	if err != nil || roomID <= 0 {
		c.JSON(http.StatusBadRequest, response("", ResponseTypeConnect, CodeBadRequest, "room_id 无效", nil))
		return
	}
	token := strings.TrimSpace(c.Query("token"))
	userID, err := h.verifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, response("", ResponseTypeConnect, CodeUnauthenticated, "token 无效或已过期", nil))
		return
	}
	if err := h.validateLiveRoom(c.Request.Context(), roomID); err != nil {
		code, message := websocketCodeFromError(err)
		c.JSON(httpStatusFromCode(code), response("", ResponseTypeConnect, code, message, nil))
		return
	}

	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if h.log != nil {
			h.log.Warn("upgrade websocket failed", zap.Int64("room_id", roomID), zap.Error(err))
		}
		return
	}

	connectedAt := nowMillis()
	connectionID := fmt.Sprintf("%d", h.ids.Next())
	identityKey, guestID := identityKeyForConnection(userID, connectionID)
	connLog := h.connectionLogger(connectionID, roomID, userID)
	conn := NewConnection(ws, ConnectionOptions{
		ID:          connectionID,
		InstanceID:  h.cfg.WebSocket.InstanceID,
		RoomID:      roomID,
		UserID:      userID,
		IdentityKey: identityKey,
		GuestID:     guestID,
		ConnectedAt: connectedAt,
		WriteQueue:  h.cfg.WebSocket.WriteQueueSize,
		OnMessage:   h.handleMessage,
		OnPong:      h.handlePong,
		OnClose:     h.handleClose,
		Log:         connLog,
	})

	h.hub.Add(conn)
	stats, err := h.online.Enter(c.Request.Context(), h.connectionState(conn))
	if err != nil {
		h.hub.Remove(conn)
		conn.Close()
		if h.log != nil {
			h.log.Error("record websocket enter failed", zap.String("connection_id", connectionID), zap.Error(err))
		}
		return
	}
	h.publishOnlineIfChanged(c.Request.Context(), conn, stats)

	conn.Send(response("", ResponseTypeConnect, CodeOK, "connected", h.connectResponseData(c.Request.Context(), conn)))
	conn.Start(c.Request.Context(), h.cfg.WebSocket.MaxMessageBytes, h.cfg.HeartbeatTimeout(), h.cfg.HeartbeatInterval())
}

func (h *WebSocketHandler) connectResponseData(ctx context.Context, conn *Connection) map[string]any {
	recent := h.recentDanmaku(ctx, conn.RoomID)
	return map[string]any{
		"connection_id":              conn.ID,
		"room_id":                    conn.RoomID,
		"user_id":                    conn.UserID,
		"identity_key":               conn.IdentityKey,
		"heartbeat_interval_seconds": h.cfg.WebSocket.HeartbeatIntervalSeconds,
		"heartbeat_timeout_seconds":  h.cfg.WebSocket.HeartbeatTimeoutSeconds,
		"connected_at":               conn.ConnectedAt,
		"reconnect_strategy":         "http_snapshot",
		"resync_on_connect":          true,
		"snapshot_url":               fmt.Sprintf("/api/user/live/rooms/%d/auction-snapshot", conn.RoomID),
		"auction_records_url":        fmt.Sprintf("/api/user/live/rooms/%d/auction-records", conn.RoomID),
		"recent_danmaku":             recent,
	}
}

func (h *WebSocketHandler) recentDanmaku(ctx context.Context, roomID int64) []repository.RecentDanmaku {
	if h.danmaku == nil || roomID <= 0 || h.cfg.Danmaku.RecentLimit <= 0 {
		return []repository.RecentDanmaku{}
	}
	recent, err := h.danmaku.GetRecent(ctx, roomID, h.cfg.Danmaku.RecentLimit)
	if err != nil {
		if h.log != nil {
			h.log.Warn("get recent danmaku failed", zap.Int64("room_id", roomID), zap.Error(err))
		}
		return []repository.RecentDanmaku{}
	}
	if recent == nil {
		return []repository.RecentDanmaku{}
	}
	return recent
}

func (h *WebSocketHandler) BroadcastOnlineEvent(ctx context.Context, event repository.OnlineEvent) {
	msg := BroadcastMessage{
		Type:       EventRoomOnlineChanged,
		EventID:    event.EventID,
		RoomID:     event.RoomID,
		ServerTime: event.ServerTime,
		Data: OnlineChangedData{
			OnlineUserCount: event.OnlineUserCount,
			ConnectionCount: event.ConnectionCount,
		},
	}
	h.hub.BroadcastRoom(event.RoomID, msg)
}

func (h *WebSocketHandler) QueueOnlineEvent(event repository.OnlineEvent) {
	if event.RoomID <= 0 {
		return
	}
	h.onlineMu.Lock()
	h.pendingOnline[event.RoomID] = event
	h.onlineMu.Unlock()
}

// update: 继续优化在线事件的处理逻辑
func (h *WebSocketHandler) FlushOnlineEvents(ctx context.Context) {
	pending := h.takePendingOnlineEvents()
	for roomID, event := range pending {
		stats := repository.OnlineStats{
			RoomID:          roomID,
			OnlineUserCount: event.OnlineUserCount,
			ConnectionCount: event.ConnectionCount,
			Changed:         true,
		}
		if h.online != nil {
			latest, err := h.online.GetStats(ctx, roomID)
			if err != nil {
				if h.log != nil {
					h.log.Warn("get websocket online stats failed", zap.Int64("room_id", roomID), zap.Error(err))
				}
			} else {
				stats = latest
			}
		}
		if !h.shouldBroadcastOnlineStats(roomID, stats) {
			continue
		}
		h.BroadcastOnlineEvent(ctx, repository.OnlineEvent{
			EventID:         h.eventID("evt_online"),
			RoomID:          roomID,
			ServerTime:      nowMillis(),
			OnlineUserCount: stats.OnlineUserCount,
			ConnectionCount: stats.ConnectionCount,
		})
	}
}

func (h *WebSocketHandler) takePendingOnlineEvents() map[int64]repository.OnlineEvent {
	h.onlineMu.Lock()
	defer h.onlineMu.Unlock()
	if len(h.pendingOnline) == 0 {
		return nil
	}
	pending := h.pendingOnline
	h.pendingOnline = make(map[int64]repository.OnlineEvent)
	return pending
}

func (h *WebSocketHandler) shouldBroadcastOnlineStats(roomID int64, stats repository.OnlineStats) bool {
	h.onlineMu.Lock()
	defer h.onlineMu.Unlock()
	last, ok := h.lastOnlineSent[roomID]
	if ok && last.OnlineUserCount == stats.OnlineUserCount && last.ConnectionCount == stats.ConnectionCount {
		return false
	}
	h.lastOnlineSent[roomID] = repository.OnlineStats{
		RoomID:          roomID,
		OnlineUserCount: stats.OnlineUserCount,
		ConnectionCount: stats.ConnectionCount,
	}
	return true
}

func (h *WebSocketHandler) BroadcastAuctionEvent(ctx context.Context, event AuctionEvent) {
	if event.RoomID <= 0 {
		return
	}
	msg := BroadcastMessage{
		Type:       event.EventType,
		EventID:    event.EventID,
		RoomID:     event.RoomID,
		AuctionID:  event.AuctionID,
		Version:    event.Version,
		ServerTime: event.TimestampMillis(),
		Data:       normalizeEventData(event.Data),
	}
	h.hub.BroadcastRoom(event.RoomID, msg)
}

func (h *WebSocketHandler) CleanupRoom(ctx context.Context, roomID int64) {
	stats, err := h.online.CleanupRoom(ctx, roomID, 256)
	if err != nil {
		if h.log != nil {
			h.log.Warn("cleanup websocket room failed", zap.Int64("room_id", roomID), zap.Error(err))
		}
		return
	}
	if stats.Changed {
		h.publishOnlineEvent(ctx, repository.OnlineEvent{
			EventID:         h.eventID("evt_online"),
			RoomID:          roomID,
			ServerTime:      nowMillis(),
			OnlineUserCount: stats.OnlineUserCount,
			ConnectionCount: stats.ConnectionCount,
		})
	}
}

func (h *WebSocketHandler) handleMessage(ctx context.Context, conn *Connection, message ClientMessage) {
	switch strings.TrimSpace(message.Type) {
	case MessageTypePing:
		h.handlePing(ctx, conn, message)
	case MessageTypePlaceBid:
		h.handlePlaceBid(ctx, conn, message)
	case MessageTypeSendDanmaku:
		h.handleSendDanmaku(ctx, conn, message)
	case MessageTypeRoomLeave:
		conn.Send(response(message.RequestID, MessageTypeRoomLeave, CodeOK, "success", nil))
		conn.Close()
	default:
		requestType := strings.TrimSpace(message.Type)
		if requestType == "" {
			requestType = ResponseTypeMalformed
		}
		conn.Send(response(message.RequestID, requestType, CodeBadRequest, "不支持的消息类型", nil))
	}
}

func (h *WebSocketHandler) handlePing(ctx context.Context, conn *Connection, message ClientMessage) {
	stats, err := h.online.Heartbeat(ctx, h.connectionState(conn))
	if err != nil {
		conn.Send(response(message.RequestID, MessageTypePing, CodeInternal, "心跳刷新失败", nil))
		if h.log != nil {
			h.log.Warn("websocket heartbeat failed", zap.String("connection_id", conn.ID), zap.Error(err))
		}
		return
	}
	if stats.Changed {
		h.publishOnlineIfChanged(ctx, conn, stats)
	}
	conn.Send(response(message.RequestID, MessageTypePing, CodeOK, "pong", nil))
}

func (h *WebSocketHandler) handlePong(ctx context.Context, conn *Connection) {
	stats, err := h.online.Heartbeat(ctx, h.connectionState(conn))
	if err != nil {
		if h.log != nil {
			h.log.Warn("websocket pong heartbeat failed", zap.String("connection_id", conn.ID), zap.Error(err))
		}
		return
	}
	if stats.Changed {
		h.publishOnlineIfChanged(ctx, conn, stats)
	}
}

func (h *WebSocketHandler) handlePlaceBid(ctx context.Context, conn *Connection, message ClientMessage) {
	if conn.UserID <= 0 {
		conn.Send(response(message.RequestID, MessageTypePlaceBid, CodeUnauthenticated, "请先登录后再出价", nil))
		return
	}
	var data PlaceBidData
	if len(message.Data) == 0 {
		conn.Send(response(message.RequestID, MessageTypePlaceBid, CodeBadRequest, "出价参数无效", nil))
		return
	}
	if err := json.Unmarshal(message.Data, &data); err != nil || data.AuctionID <= 0 || data.BidPrice <= 0 || strings.TrimSpace(message.RequestID) == "" {
		conn.Send(response(message.RequestID, MessageTypePlaceBid, CodeBadRequest, "出价参数无效", nil))
		return
	}

	rpcCtx, cancel := context.WithTimeout(ctx, h.rpcTimeout)
	defer cancel()
	rpcCtx = identity.NewOutgoingContext(rpcCtx, identity.Principal{Kind: identity.KindUser, ID: conn.UserID})
	resp, err := h.auction.PlaceBid(rpcCtx, &auctionv1.PlaceBidRequest{
		AuctionId: data.AuctionID,
		UserId:    conn.UserID,
		BidPrice:  data.BidPrice,
		RequestId: message.RequestID,
		RoomId:    conn.RoomID,
	})
	if err != nil {
		code, text := websocketCodeFromError(err)
		conn.Send(response(message.RequestID, MessageTypePlaceBid, code, text, nil))
		return
	}
	conn.Send(response(message.RequestID, MessageTypePlaceBid, CodeOK, "success", map[string]any{
		"accepted":       resp.GetAccepted(),
		"current_price":  resp.GetCurrentPrice(),
		"bid_count":      resp.GetBidCount(),
		"winner_user_id": resp.GetWinnerUserId(),
		"server_time":    protoMillis(resp.GetServerTime()),
		"expire_at":      protoMillis(resp.GetExpireAt()),
	}))
}

func (h *WebSocketHandler) handleSendDanmaku(ctx context.Context, conn *Connection, message ClientMessage) {
	if !h.cfg.Danmaku.Enabled {
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeForbidden, "弹幕功能未启用", nil))
		return
	}
	if h.danmaku == nil {
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeInternal, "弹幕服务不可用", nil))
		return
	}
	if conn.UserID <= 0 {
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeUnauthenticated, "请先登录后再发送弹幕", nil))
		return
	}
	if conn.RoomID <= 0 {
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeBadRequest, "直播间无效", nil))
		return
	}
	requestID := strings.TrimSpace(message.RequestID)
	if requestID == "" || len(message.Data) == 0 {
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeBadRequest, "弹幕参数无效", nil))
		return
	}
	var data SendDanmakuData
	if err := json.Unmarshal(message.Data, &data); err != nil {
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeBadRequest, "弹幕参数无效", nil))
		return
	}
	content := strings.TrimSpace(data.Content)
	if content == "" || len([]rune(content)) > h.cfg.Danmaku.MaxChars {
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeBadRequest, "弹幕内容无效", nil))
		return
	}
	if err := h.ensureRoomLivingForDanmaku(ctx, conn.RoomID); err != nil {
		code, text := websocketCodeFromError(err)
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, code, text, nil))
		return
	}

	idempotent, ok, err := h.danmaku.GetIdempotent(ctx, conn.RoomID, conn.UserID, requestID)
	if err != nil {
		if h.log != nil {
			h.log.Warn("get danmaku idempotent cache failed", zap.Int64("room_id", conn.RoomID), zap.Int64("user_id", conn.UserID), zap.Error(err))
		}
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeInternal, "弹幕发送失败", nil))
		return
	}
	if ok {
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeOK, "success", map[string]any{
			"message_id": idempotent.MessageID,
			"room_id":    idempotent.RoomID,
		}))
		return
	}

	allowed, err := h.danmaku.AllowSend(ctx, conn.RoomID, conn.UserID, h.cfg.DanmakuRateLimit())
	if err != nil {
		if h.log != nil {
			h.log.Warn("check danmaku rate limit failed", zap.Int64("room_id", conn.RoomID), zap.Int64("user_id", conn.UserID), zap.Error(err))
		}
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeInternal, "弹幕发送失败", nil))
		return
	}
	if !allowed {
		if idempotent, ok, err := h.danmaku.GetIdempotent(ctx, conn.RoomID, conn.UserID, requestID); err == nil && ok {
			conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeOK, "success", map[string]any{
				"message_id": idempotent.MessageID,
				"room_id":    idempotent.RoomID,
			}))
			return
		}
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeTooManyRequests, "弹幕发送过于频繁", nil))
		return
	}

	nickname := h.nicknameForDanmaku(ctx, conn)
	messageID := h.eventID("dm")
	eventID := h.eventID("evt_dm")
	serverTime := nowMillis()
	event := repository.DanmakuBroadcast{
		Type:       EventDanmakuCreated,
		EventID:    eventID,
		RoomID:     conn.RoomID,
		ServerTime: serverTime,
		Data: repository.DanmakuData{
			MessageID:   messageID,
			SenderType:  repository.DanmakuSenderTypeUser,
			UserID:      conn.UserID,
			Nickname:    nickname,
			Content:     content,
			ContentType: repository.DanmakuContentTypeText,
			Status:      repository.DanmakuStatusVisible,
		},
	}
	if err := h.danmaku.SetIdempotent(ctx, conn.RoomID, conn.UserID, requestID, repository.DanmakuIdempotentResult{
		MessageID: messageID,
		RoomID:    conn.RoomID,
	}, h.cfg.DanmakuRequestTTL()); err != nil {
		if h.log != nil {
			h.log.Warn("set danmaku idempotent cache failed", zap.Int64("room_id", conn.RoomID), zap.Int64("user_id", conn.UserID), zap.Error(err))
		}
		conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeInternal, "弹幕发送失败", nil))
		return
	}
	if err := h.danmaku.AppendRecent(ctx, conn.RoomID, event, h.cfg.Danmaku.RecentLimit, h.cfg.DanmakuRecentTTL()); err != nil && h.log != nil {
		h.log.Warn("append recent danmaku failed", zap.Int64("room_id", conn.RoomID), zap.Error(err))
	}

	h.markDanmakuSeen(messageID)
	h.hub.BroadcastRoom(conn.RoomID, event)
	if err := h.danmaku.PublishDanmaku(ctx, event); err != nil && h.log != nil {
		h.log.Warn("publish danmaku broadcast failed", zap.Int64("room_id", conn.RoomID), zap.Error(err))
	}
	if err := h.danmaku.PublishAIInput(ctx, repository.DanmakuCreatedEvent{
		EventID:     eventID,
		EventType:   InteractionEventDanmakuCreated,
		MessageID:   messageID,
		RoomID:      conn.RoomID,
		UserID:      conn.UserID,
		Nickname:    nickname,
		Content:     content,
		ContentType: repository.DanmakuContentTypeText,
		ServerTime:  serverTime,
	}); err != nil && h.log != nil {
		h.log.Warn("publish danmaku ai input failed", zap.Int64("room_id", conn.RoomID), zap.Error(err))
	}
	conn.Send(response(message.RequestID, MessageTypeSendDanmaku, CodeOK, "success", map[string]any{
		"message_id": messageID,
		"room_id":    conn.RoomID,
	}))
}

func (h *WebSocketHandler) ensureRoomLivingForDanmaku(ctx context.Context, roomID int64) error {
	if h.danmaku != nil {
		statusValue, ok, err := h.danmaku.GetRoomStatus(ctx, roomID)
		if err == nil && ok {
			if statusValue == repository.RoomStatusLiving {
				return nil
			}
			return status.Error(codes.PermissionDenied, "直播间不允许发送弹幕")
		}
		if err != nil && h.log != nil {
			h.log.Warn("get room status cache failed", zap.Int64("room_id", roomID), zap.Error(err))
		}
	}
	if h.live == nil {
		return status.Error(codes.Unavailable, "live service unavailable")
	}
	rpcCtx, cancel := context.WithTimeout(ctx, h.rpcTimeout)
	defer cancel()
	resp, err := h.live.GetLiveRoom(rpcCtx, &livev1.GetLiveRoomRequest{Id: roomID})
	if err != nil {
		return err
	}
	room := resp.GetLiveRoom()
	if room == nil || room.GetId() <= 0 {
		return status.Error(codes.NotFound, "直播间不存在")
	}
	statusValue := repository.RoomStatusNotLive
	if room.GetStatus() == livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING {
		statusValue = repository.RoomStatusLiving
	}
	if h.danmaku != nil {
		if err := h.danmaku.SetRoomStatus(ctx, roomID, statusValue, h.cfg.RoomStatusCacheTTL()); err != nil && h.log != nil {
			h.log.Warn("set room status cache failed", zap.Int64("room_id", roomID), zap.Error(err))
		}
	}
	if statusValue != repository.RoomStatusLiving {
		return status.Error(codes.PermissionDenied, "直播间不允许发送弹幕")
	}
	return nil
}

func (h *WebSocketHandler) nicknameForDanmaku(ctx context.Context, conn *Connection) string {
	if nickname := strings.TrimSpace(conn.Nickname); nickname != "" {
		return nickname
	}
	if h.danmaku != nil {
		nickname, ok, err := h.danmaku.GetNickname(ctx, conn.UserID)
		if err == nil && ok && nickname != "" {
			conn.Nickname = nickname
			return nickname
		}
		if err != nil && h.log != nil {
			h.log.Warn("get user nickname cache failed", zap.Int64("user_id", conn.UserID), zap.Error(err))
		}
	}
	if h.user != nil {
		rpcCtx, cancel := context.WithTimeout(ctx, h.rpcTimeout)
		defer cancel()
		resp, err := h.user.GetUser(rpcCtx, &userv1.GetUserRequest{Id: conn.UserID})
		if err == nil {
			nickname := strings.TrimSpace(resp.GetUser().GetNickname())
			if nickname != "" {
				conn.Nickname = nickname
				if h.danmaku != nil {
					if err := h.danmaku.SetNickname(ctx, conn.UserID, nickname, h.cfg.NicknameCacheTTL()); err != nil && h.log != nil {
						h.log.Warn("set user nickname cache failed", zap.Int64("user_id", conn.UserID), zap.Error(err))
					}
				}
				return nickname
			}
		} else if h.log != nil {
			h.log.Warn("get user nickname failed", zap.Int64("user_id", conn.UserID), zap.Error(err))
		}
	}
	nickname := fallbackNickname(conn.UserID)
	conn.Nickname = nickname
	return nickname
}

func (h *WebSocketHandler) BroadcastDanmakuEvent(ctx context.Context, event repository.DanmakuBroadcast) {
	if event.RoomID <= 0 || event.Data.MessageID == "" {
		return
	}
	if !h.markDanmakuSeen(event.Data.MessageID) {
		return
	}
	h.hub.BroadcastRoom(event.RoomID, event)
}

func (h *WebSocketHandler) markDanmakuSeen(messageID string) bool {
	if messageID == "" {
		return false
	}
	now := nowMillis()
	h.danmakuMu.Lock()
	defer h.danmakuMu.Unlock()
	if _, ok := h.seenDanmaku[messageID]; ok {
		return false
	}
	if len(h.seenDanmaku) > 2048 {
		cutoff := now - int64(time.Hour/time.Millisecond)
		for id, seenAt := range h.seenDanmaku {
			if seenAt < cutoff {
				delete(h.seenDanmaku, id)
			}
		}
		if len(h.seenDanmaku) > 4096 {
			h.seenDanmaku = make(map[string]int64)
		}
	}
	h.seenDanmaku[messageID] = now
	return true
}

func (h *WebSocketHandler) handleClose(ctx context.Context, conn *Connection) {
	h.hub.Remove(conn)
	cleanupCtx, cancel := context.WithTimeout(context.Background(), h.rpcTimeout)
	defer cancel()
	stats, err := h.online.Leave(cleanupCtx, conn.RoomID, conn.IdentityKey, conn.ID)
	if err != nil {
		if h.log != nil {
			h.log.Warn("record websocket leave failed", zap.String("connection_id", conn.ID), zap.Error(err))
		}
		return
	}
	h.publishOnlineIfChanged(cleanupCtx, conn, stats)
}

func (h *WebSocketHandler) verifyToken(token string) (int64, error) {
	if token == "" {
		return 0, nil
	}
	if h.jwt == nil {
		return 0, auth.ErrMissingSecret
	}
	claims, err := h.jwt.Verify(token)
	if err != nil {
		return 0, err
	}
	principal, ok := identity.NewPrincipal(identity.KindUser, claims.Subject)
	if !ok {
		return 0, auth.ErrInvalidClaims
	}
	return principal.ID, nil
}

func (h *WebSocketHandler) validateLiveRoom(ctx context.Context, roomID int64) error {
	if h.live == nil {
		return errors.New("live service unavailable")
	}
	rpcCtx, cancel := context.WithTimeout(ctx, h.rpcTimeout)
	defer cancel()
	resp, err := h.live.GetLiveRoom(rpcCtx, &livev1.GetLiveRoomRequest{Id: roomID})
	if err != nil {
		return err
	}
	room := resp.GetLiveRoom()
	if room == nil || room.GetId() <= 0 {
		return status.Error(codes.NotFound, "直播间不存在")
	}
	if h.cfg.WebSocket.RequireLivingRoom && room.GetStatus() != livev1.LiveRoomStatus_LIVE_ROOM_STATUS_LIVING {
		return status.Error(codes.FailedPrecondition, "直播间未开播")
	}
	return nil
}

func (h *WebSocketHandler) connectionState(conn *Connection) repository.ConnectionState {
	now := nowMillis()
	return repository.ConnectionState{
		ConnectionID: conn.ID,
		InstanceID:   conn.InstanceID,
		RoomID:       conn.RoomID,
		IdentityKey:  conn.IdentityKey,
		UserID:       conn.UserID,
		ConnectedAt:  conn.ConnectedAt,
		LastSeenAt:   now,
		ExpireAt:     now + h.cfg.HeartbeatTimeout().Milliseconds(),
	}
}

func (h *WebSocketHandler) publishOnlineIfChanged(ctx context.Context, conn *Connection, stats repository.OnlineStats) {
	if !stats.Changed {
		return
	}
	h.publishOnlineEvent(ctx, repository.OnlineEvent{
		EventID:             h.eventID("evt_online"),
		RoomID:              conn.RoomID,
		ServerTime:          nowMillis(),
		OnlineUserCount:     stats.OnlineUserCount,
		ConnectionCount:     stats.ConnectionCount,
		ChangedIdentityKey:  conn.IdentityKey,
		ChangedConnectionID: conn.ID,
	})
}

func (h *WebSocketHandler) publishOnlineEvent(ctx context.Context, event repository.OnlineEvent) {
	h.QueueOnlineEvent(event)
	if h.online == nil {
		return
	}
	if err := h.online.PublishOnlineEvent(ctx, event); err != nil {
		if h.log != nil {
			h.log.Warn("publish websocket online event failed", zap.Int64("room_id", event.RoomID), zap.Error(err))
		}
	}
}

func (h *WebSocketHandler) eventID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, h.ids.Next())
}

func (h *WebSocketHandler) connectionLogger(connectionID string, roomID int64, userID int64) *zap.Logger {
	if h.log == nil {
		return nil
	}
	return h.log.With(
		zap.String("connection_id", connectionID),
		zap.Int64("room_id", roomID),
		zap.Int64("user_id", userID),
	)
}

func identityKeyForConnection(userID int64, connectionID string) (string, string) {
	if userID > 0 {
		return fmt.Sprintf("u:%d", userID), ""
	}
	guestID := "g:" + connectionID
	return guestID, guestID
}

func parseAllowOrigins(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			origins = append(origins, part)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func originAllowed(origin string, allowOrigins []string) bool {
	for _, allowed := range allowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return origin == "" && len(allowOrigins) > 0
}

func websocketCodeFromError(err error) (int, string) {
	if err == nil {
		return CodeOK, "success"
	}
	switch status.Code(err) {
	case codes.InvalidArgument:
		return CodeBadRequest, grpcMessageOrDefault(err, "请求参数无效")
	case codes.Unauthenticated:
		return CodeUnauthenticated, grpcMessageOrDefault(err, "请先登录")
	case codes.PermissionDenied:
		return CodeForbidden, grpcMessageOrDefault(err, "无权执行该操作")
	case codes.NotFound:
		return CodeNotFound, grpcMessageOrDefault(err, "资源不存在")
	case codes.AlreadyExists, codes.FailedPrecondition, codes.Aborted:
		return CodeConflict, grpcMessageOrDefault(err, "请求未被接受")
	case codes.DeadlineExceeded, codes.Unavailable:
		return CodeInternal, "下游服务暂不可用，请稍后重试"
	default:
		return CodeInternal, "服务内部错误"
	}
}

func httpStatusFromCode(code int) int {
	switch code {
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeTooManyRequests:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func grpcMessageOrDefault(err error, fallback string) string {
	message := strings.TrimSpace(status.Convert(err).Message())
	if message == "" {
		return fallback
	}
	return message
}

func protoMillis(ts *timestamppb.Timestamp) int64 {
	if ts == nil || !ts.IsValid() {
		return 0
	}
	return ts.AsTime().UnixMilli()
}

func fallbackNickname(userID int64) string {
	suffix := fmt.Sprintf("%04d", userID%10000)
	if userID < 0 {
		suffix = fmt.Sprintf("%04d", -userID%10000)
	}
	return "用户" + suffix
}
