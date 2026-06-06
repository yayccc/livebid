package handler

import (
	"testing"
	"time"

	"github.com/yayccc/livebid/pkg/auth"
)

func TestDecodeAuctionEventAndNormalizeTime(t *testing.T) {
	payload := []byte(`{
		"event_id":"evt_1",
		"event_type":"bid_accepted",
		"auction_id":1001,
		"room_id":4001,
		"timestamp":1780000000,
		"data":{
			"bid_time":1780000001,
			"expire_at":1780000016000,
			"current_price":15000
		}
	}`)

	event, err := DecodeAuctionEvent(payload)
	if err != nil {
		t.Fatalf("decode event failed: %v", err)
	}
	if !event.Broadcastable() {
		t.Fatal("event should be broadcastable")
	}
	if event.TimestampMillis() != 1780000000000 {
		t.Fatalf("unexpected timestamp millis: %d", event.TimestampMillis())
	}
	data := normalizeEventData(event.Data)
	if data["bid_time"] != int64(1780000001000) {
		t.Fatalf("bid_time should be converted to millis, got %#v", data["bid_time"])
	}
	if data["expire_at"] != int64(1780000016000) {
		t.Fatalf("expire_at should stay millis, got %#v", data["expire_at"])
	}
	if data["current_price"] != int64(15000) {
		t.Fatalf("current_price should be int64, got %#v", data["current_price"])
	}
}

func TestAuctionExpireCheckIsNotBroadcastable(t *testing.T) {
	event := AuctionEvent{
		EventID:   "evt_expire",
		EventType: EventAuctionExpireCheck,
		AuctionID: 1001,
		RoomID:    4001,
	}
	if event.Broadcastable() {
		t.Fatal("expire check should not be broadcastable")
	}
}

func TestIdentityKeyForConnection(t *testing.T) {
	key, guestID := identityKeyForConnection(9001, "conn1")
	if key != "u:9001" || guestID != "" {
		t.Fatalf("unexpected user identity: %s %s", key, guestID)
	}
	key, guestID = identityKeyForConnection(0, "conn2")
	if key != "g:conn2" || guestID != "g:conn2" {
		t.Fatalf("unexpected guest identity: %s %s", key, guestID)
	}
}

func TestVerifyTokenRequiresUserIssuer(t *testing.T) {
	userJWT, err := auth.NewJWTManager("secret", auth.WithIssuer("livebid-user"))
	if err != nil {
		t.Fatalf("new user jwt manager failed: %v", err)
	}
	shopJWT, err := auth.NewJWTManager("secret", auth.WithIssuer("livebid-shop"))
	if err != nil {
		t.Fatalf("new shop jwt manager failed: %v", err)
	}
	userToken, err := userJWT.Sign(auth.Claims{Subject: "9001", ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("sign user token failed: %v", err)
	}
	shopToken, err := shopJWT.Sign(auth.Claims{Subject: "10001", ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("sign shop token failed: %v", err)
	}

	h := NewWebSocketHandler(WebSocketHandlerOptions{JWT: userJWT})
	userID, err := h.verifyToken(userToken)
	if err != nil {
		t.Fatalf("verify user token failed: %v", err)
	}
	if userID != 9001 {
		t.Fatalf("unexpected user id: %d", userID)
	}
	if _, err := h.verifyToken(shopToken); err == nil {
		t.Fatal("expected shop issuer token to be rejected")
	}
}
