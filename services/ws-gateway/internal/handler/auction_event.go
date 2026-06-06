package handler

import (
	"bytes"
	"encoding/json"
)

const EventAuctionExpireCheck = "auction_expire_check"

type AuctionEvent struct {
	EventID   string         `json:"event_id"`
	EventType string         `json:"event_type"`
	BizID     string         `json:"biz_id"`
	AuctionID int64          `json:"auction_id"`
	ShopID    int64          `json:"shop_id"`
	RoomID    int64          `json:"room_id"`
	Version   int64          `json:"version"`
	Timestamp int64          `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}

func DecodeAuctionEvent(payload []byte) (AuctionEvent, error) {
	var event AuctionEvent
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&event); err != nil {
		return AuctionEvent{}, err
	}
	return event, nil
}

func (e AuctionEvent) Broadcastable() bool {
	switch e.EventType {
	case EventAuctionStarted, EventBidAccepted, EventAuctionFinished, EventAuctionFailed, EventAuctionCancelled:
		return e.EventID != "" && e.RoomID > 0
	default:
		return false
	}
}

func (e AuctionEvent) TimestampMillis() int64 {
	return ensureMillis(e.Timestamp)
}

func normalizeEventData(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	out := make(map[string]any, len(data))
	for key, value := range data {
		switch key {
		case "timestamp", "bid_time":
			out[key] = ensureMillis(numberToInt64(value))
		default:
			out[key] = normalizeJSONValue(value)
		}
	}
	return out
}

func normalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if n, err := typed.Int64(); err == nil {
			return n
		}
		if f, err := typed.Float64(); err == nil {
			return f
		}
		return typed.String()
	case map[string]any:
		return normalizeEventData(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalizeJSONValue(item)
		}
		return out
	default:
		return value
	}
}

func numberToInt64(value any) int64 {
	switch typed := value.(type) {
	case json.Number:
		n, _ := typed.Int64()
		return n
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func ensureMillis(value int64) int64 {
	if value <= 0 {
		return nowMillis()
	}
	if value < 100000000000 {
		return value * 1000
	}
	return value
}
