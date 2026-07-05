package handler

import (
	"encoding/json"
	"time"
)

const (
	MessageTypePing        = "ping"
	MessageTypePlaceBid    = "place_bid"
	MessageTypeSendDanmaku = "send_danmaku"
	MessageTypeRoomLeave   = "room_leave"
	MessageTypeResponse    = "response"

	ResponseTypeConnect   = "connect"
	ResponseTypeMalformed = "malformed"

	EventRoomOnlineChanged    = "room_online_changed"
	EventAuctionStarted       = "auction_started"
	EventBidAccepted          = "bid_accepted"
	EventAuctionFinished      = "auction_finished"
	EventAuctionFailed        = "auction_failed"
	EventAuctionCancelled     = "auction_cancelled"
	EventDanmakuCreated       = "danmaku_created"
	EventAIInteractionCreated = "ai_interaction_created"

	InteractionEventDanmakuCreated = "danmaku.created"
)

const (
	CodeOK              = 0
	CodeBadRequest      = 400
	CodeUnauthenticated = 401
	CodeForbidden       = 403
	CodeNotFound        = 404
	CodeConflict        = 409
	CodeTooManyRequests = 429
	CodeInternal        = 500
)

type ClientMessage struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type ResponseMessage struct {
	Type        string `json:"type"`
	RequestID   string `json:"request_id,omitempty"`
	RequestType string `json:"request_type,omitempty"`
	Code        int    `json:"code"`
	Message     string `json:"message"`
	ServerTime  int64  `json:"server_time"`
	Data        any    `json:"data,omitempty"`
}

type BroadcastMessage struct {
	Type       string `json:"type"`
	EventID    string `json:"event_id,omitempty"`
	RoomID     int64  `json:"room_id"`
	AuctionID  int64  `json:"auction_id,omitempty"`
	Version    int64  `json:"version,omitempty"`
	ServerTime int64  `json:"server_time"`
	Data       any    `json:"data,omitempty"`
}

type PlaceBidData struct {
	AuctionID int64 `json:"auction_id"`
	BidPrice  int64 `json:"bid_price"`
}

type SendDanmakuData struct {
	Content string `json:"content"`
}

type OnlineChangedData struct {
	OnlineUserCount int64 `json:"online_user_count"`
	ConnectionCount int64 `json:"connection_count"`
}

type OnlineChangedEvent struct {
	EventID             string `json:"event_id"`
	RoomID              int64  `json:"room_id"`
	ServerTime          int64  `json:"server_time"`
	OnlineUserCount     int64  `json:"online_user_count"`
	ConnectionCount     int64  `json:"connection_count"`
	ChangedIdentityKey  string `json:"changed_identity_key,omitempty"`
	ChangedConnectionID string `json:"changed_connection_id,omitempty"`
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

func response(requestID string, requestType string, code int, message string, data any) ResponseMessage {
	return ResponseMessage{
		Type:        MessageTypeResponse,
		RequestID:   requestID,
		RequestType: requestType,
		Code:        code,
		Message:     message,
		ServerTime:  nowMillis(),
		Data:        data,
	}
}

func broadcast(eventType string, eventID string, roomID int64, auctionID int64, data any) BroadcastMessage {
	return BroadcastMessage{
		Type:       eventType,
		EventID:    eventID,
		RoomID:     roomID,
		AuctionID:  auctionID,
		ServerTime: nowMillis(),
		Data:       data,
	}
}
