package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/yayccc/livebid/services/ws-gateway/internal/config"
	"github.com/yayccc/livebid/services/ws-gateway/internal/repository"
)

func TestConnectResponseIncludesResyncHints(t *testing.T) {
	h := NewWebSocketHandler(WebSocketHandlerOptions{
		Config: config.Config{
			WebSocket: config.WebSocketConfig{
				HeartbeatIntervalSeconds: 30,
				HeartbeatTimeoutSeconds:  90,
			},
			Danmaku: config.DanmakuConfig{
				RecentLimit: 10,
			},
		},
	})
	conn := &Connection{
		ID:          "conn-1",
		RoomID:      2001,
		UserID:      3001,
		IdentityKey: "u:3001",
		ConnectedAt: 1780000000000,
	}

	data := h.connectResponseData(context.Background(), conn)

	if data["reconnect_strategy"] != "http_snapshot" ||
		data["resync_on_connect"] != true ||
		data["snapshot_url"] != "/api/user/live/rooms/2001/auction-snapshot" ||
		data["auction_records_url"] != "/api/user/live/rooms/2001/auction-records" {
		t.Fatalf("unexpected reconnect hints: %#v", data)
	}
	if data["heartbeat_interval_seconds"] != 30 || data["heartbeat_timeout_seconds"] != 90 {
		t.Fatalf("unexpected heartbeat config: %#v", data)
	}
	recent, ok := data["recent_danmaku"].([]repository.RecentDanmaku)
	if !ok || len(recent) != 0 {
		t.Fatalf("unexpected recent danmaku: %#v", data["recent_danmaku"])
	}
}

func TestConnectResponseIncludesRecentDanmaku(t *testing.T) {
	store := &fakeDanmakuStore{
		recent: []repository.RecentDanmaku{
			{MessageID: "dm_1", SenderType: repository.DanmakuSenderTypeUser, UserID: 9001, Nickname: "tester", Content: "hello", ContentType: repository.DanmakuContentTypeText, ServerTime: 1000},
		},
	}
	h := NewWebSocketHandler(WebSocketHandlerOptions{
		Config: config.Config{
			Danmaku: config.DanmakuConfig{
				RecentLimit: 10,
			},
		},
		Danmaku: store,
	})
	data := h.connectResponseData(context.Background(), &Connection{ID: "conn-1", RoomID: 4001, UserID: 9001})

	recent, ok := data["recent_danmaku"].([]repository.RecentDanmaku)
	if !ok || len(recent) != 1 || recent[0].MessageID != "dm_1" {
		t.Fatalf("unexpected recent danmaku: %#v", data["recent_danmaku"])
	}
}

func TestHandleSendDanmakuBroadcastsAndResponds(t *testing.T) {
	store := &fakeDanmakuStore{
		roomStatus: repository.RoomStatusLiving,
		nickname:   "tester",
		allowSend:  true,
	}
	h := NewWebSocketHandler(WebSocketHandlerOptions{
		Config: config.Config{
			Danmaku: config.DanmakuConfig{
				Enabled:           true,
				MaxChars:          30,
				RecentLimit:       10,
				RateLimitSeconds:  1,
				RecentTTLHours:    24,
				RequestTTLSeconds: 60,
			},
		},
		Danmaku: store,
	})
	conn := testConnection(4001, 9001)
	h.Hub().Add(conn)

	h.handleSendDanmaku(context.Background(), conn, ClientMessage{
		Type:      MessageTypeSendDanmaku,
		RequestID: "req-1",
		Data:      json.RawMessage(`{"content":"  hello  "}`),
	})

	broadcastMsg := readQueuedMessage[repository.DanmakuBroadcast](t, conn)
	if broadcastMsg.Type != EventDanmakuCreated ||
		broadcastMsg.RoomID != 4001 ||
		broadcastMsg.Data.UserID != 9001 ||
		broadcastMsg.Data.Nickname != "tester" ||
		broadcastMsg.Data.Content != "hello" {
		t.Fatalf("unexpected broadcast: %#v", broadcastMsg)
	}
	responseMsg := readQueuedMessage[ResponseMessage](t, conn)
	if responseMsg.Code != CodeOK || responseMsg.RequestType != MessageTypeSendDanmaku {
		t.Fatalf("unexpected response: %#v", responseMsg)
	}
	if store.idempotent.MessageID == "" || len(store.appended) != 1 || len(store.published) != 1 || len(store.aiInputs) != 1 {
		t.Fatalf("store side effects missing: %#v", store)
	}
	if store.aiInputs[0].EventType != InteractionEventDanmakuCreated {
		t.Fatalf("unexpected ai input event type: %#v", store.aiInputs[0])
	}
}

func TestHandleSendDanmakuRateLimited(t *testing.T) {
	store := &fakeDanmakuStore{
		roomStatus: repository.RoomStatusLiving,
		allowSend:  false,
	}
	h := NewWebSocketHandler(WebSocketHandlerOptions{
		Config: config.Config{
			Danmaku: config.DanmakuConfig{
				Enabled:          true,
				MaxChars:         30,
				RateLimitSeconds: 1,
			},
		},
		Danmaku: store,
	})
	conn := testConnection(4001, 9001)

	h.handleSendDanmaku(context.Background(), conn, ClientMessage{
		Type:      MessageTypeSendDanmaku,
		RequestID: "req-1",
		Data:      json.RawMessage(`{"content":"hello"}`),
	})

	responseMsg := readQueuedMessage[ResponseMessage](t, conn)
	if responseMsg.Code != CodeTooManyRequests {
		t.Fatalf("expected 429 response, got %#v", responseMsg)
	}
	if len(store.appended) != 0 || len(store.published) != 0 {
		t.Fatalf("rate limited danmaku should not broadcast: %#v", store)
	}
}

func testConnection(roomID int64, userID int64) *Connection {
	return &Connection{
		ID:          "conn-1",
		RoomID:      roomID,
		UserID:      userID,
		IdentityKey: "u:9001",
		writeQueue:  make(chan []byte, 8),
		closeSignal: make(chan struct{}),
	}
}

func readQueuedMessage[T any](t *testing.T, conn *Connection) T {
	t.Helper()
	select {
	case payload := <-conn.writeQueue:
		var out T
		if err := json.Unmarshal(payload, &out); err != nil {
			t.Fatalf("decode queued message %s: %v", payload, err)
		}
		return out
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for queued websocket message")
		var zero T
		return zero
	}
}

type fakeDanmakuStore struct {
	roomStatus string
	nickname   string
	allowSend  bool
	recent     []repository.RecentDanmaku

	idempotent repository.DanmakuIdempotentResult
	appended   []repository.DanmakuBroadcast
	published  []repository.DanmakuBroadcast
	aiInputs   []repository.DanmakuCreatedEvent
}

func (s *fakeDanmakuStore) GetRoomStatus(ctx context.Context, roomID int64) (string, bool, error) {
	if s.roomStatus == "" {
		return "", false, nil
	}
	return s.roomStatus, true, nil
}

func (s *fakeDanmakuStore) SetRoomStatus(ctx context.Context, roomID int64, status string, ttl time.Duration) error {
	s.roomStatus = status
	return nil
}

func (s *fakeDanmakuStore) GetIdempotent(ctx context.Context, roomID int64, userID int64, requestID string) (repository.DanmakuIdempotentResult, bool, error) {
	if s.idempotent.MessageID == "" {
		return repository.DanmakuIdempotentResult{}, false, nil
	}
	return s.idempotent, true, nil
}

func (s *fakeDanmakuStore) SetIdempotent(ctx context.Context, roomID int64, userID int64, requestID string, result repository.DanmakuIdempotentResult, ttl time.Duration) error {
	s.idempotent = result
	return nil
}

func (s *fakeDanmakuStore) AllowSend(ctx context.Context, roomID int64, userID int64, interval time.Duration) (bool, error) {
	return s.allowSend, nil
}

func (s *fakeDanmakuStore) GetNickname(ctx context.Context, userID int64) (string, bool, error) {
	if s.nickname == "" {
		return "", false, nil
	}
	return s.nickname, true, nil
}

func (s *fakeDanmakuStore) SetNickname(ctx context.Context, userID int64, nickname string, ttl time.Duration) error {
	s.nickname = nickname
	return nil
}

func (s *fakeDanmakuStore) AppendRecent(ctx context.Context, roomID int64, message repository.DanmakuBroadcast, limit int, ttl time.Duration) error {
	s.appended = append(s.appended, message)
	return nil
}

func (s *fakeDanmakuStore) GetRecent(ctx context.Context, roomID int64, limit int) ([]repository.RecentDanmaku, error) {
	return s.recent, nil
}

func (s *fakeDanmakuStore) PublishDanmaku(ctx context.Context, message repository.DanmakuBroadcast) error {
	s.published = append(s.published, message)
	return nil
}

func (s *fakeDanmakuStore) SubscribeDanmaku(ctx context.Context, handle func(context.Context, repository.DanmakuBroadcast)) error {
	return nil
}

func (s *fakeDanmakuStore) PublishAIInput(ctx context.Context, event repository.DanmakuCreatedEvent) error {
	s.aiInputs = append(s.aiInputs, event)
	return nil
}

func (s *fakeDanmakuStore) Close() error {
	return nil
}
