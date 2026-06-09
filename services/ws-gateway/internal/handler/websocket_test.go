package handler

import (
	"testing"

	"github.com/yayccc/livebid/services/ws-gateway/internal/config"
)

func TestConnectResponseIncludesResyncHints(t *testing.T) {
	h := NewWebSocketHandler(WebSocketHandlerOptions{
		Config: config.Config{
			WebSocket: config.WebSocketConfig{
				HeartbeatIntervalSeconds: 30,
				HeartbeatTimeoutSeconds:  90,
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

	data := h.connectResponseData(conn)

	if data["reconnect_strategy"] != "http_snapshot" ||
		data["resync_on_connect"] != true ||
		data["snapshot_url"] != "/api/user/live/rooms/2001/auction-snapshot" ||
		data["auction_records_url"] != "/api/user/live/rooms/2001/auction-records" {
		t.Fatalf("unexpected reconnect hints: %#v", data)
	}
	if data["heartbeat_interval_seconds"] != 30 || data["heartbeat_timeout_seconds"] != 90 {
		t.Fatalf("unexpected heartbeat config: %#v", data)
	}
}
