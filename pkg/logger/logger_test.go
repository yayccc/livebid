package logger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNewWritesJSONToFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app.log")

	log, err := New(Config{
		ServiceName: "auction-service",
		Env:         "test",
		Level:       "debug",
		File: FileConfig{
			Filename: file,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	log.Info("bid accepted", RequestID("req-1"), AuctionID(99))
	_ = log.Sync()

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	output := string(data)
	for _, want := range []string{`"service":"auction-service"`, `"env":"test"`, `"msg":"bid accepted"`, `"request_id":"req-1"`, `"auction_id":99`} {
		if !strings.Contains(output, want) {
			t.Fatalf("log output missing %s: %s", want, output)
		}
	}
}

func TestContextFieldsAreCopied(t *testing.T) {
	ctx := WithContextFields(context.Background(), RequestID("req-1"))
	fields := FieldsFromContext(ctx)
	fields[0] = zap.String("changed", "yes")

	got := FieldsFromContext(ctx)
	if got[0].Key != FieldRequestID {
		t.Fatalf("context fields were mutated: %#v", got)
	}
}

func TestInvalidLevelReturnsError(t *testing.T) {
	_, err := New(Config{Level: "chatty"})
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestInvalidEncodingReturnsError(t *testing.T) {
	_, err := New(Config{Encoding: "pretty"})
	if err == nil {
		t.Fatal("expected error for invalid log encoding")
	}
}
