package handler

import (
	"encoding/json"
	"testing"
)

func TestResponseIncludesRequestType(t *testing.T) {
	msg := response("req-1", MessageTypePlaceBid, CodeOK, "success", nil)

	if msg.Type != MessageTypeResponse || msg.RequestID != "req-1" || msg.RequestType != MessageTypePlaceBid {
		t.Fatalf("unexpected response message: %#v", msg)
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded["request_type"] != MessageTypePlaceBid {
		t.Fatalf("missing request_type in json payload: %s", payload)
	}
}
