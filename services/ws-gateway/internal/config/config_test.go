package config

import "testing"

func TestDefaultConfigAlignsDesign(t *testing.T) {
	cfg := defaultConfig()
	normalize(&cfg)

	if cfg.HTTP.Addr != ":58081" {
		t.Fatalf("unexpected http addr: %s", cfg.HTTP.Addr)
	}
	if cfg.WorkerID != 8 {
		t.Fatalf("unexpected worker id: %d", cfg.WorkerID)
	}
	if cfg.JWT.UserIssuer != "livebid-user" {
		t.Fatalf("unexpected user issuer: %s", cfg.JWT.UserIssuer)
	}
	if cfg.WebSocket.HeartbeatIntervalSeconds != 30 {
		t.Fatalf("unexpected heartbeat interval: %d", cfg.WebSocket.HeartbeatIntervalSeconds)
	}
	if cfg.WebSocket.HeartbeatTimeoutSeconds != 90 {
		t.Fatalf("unexpected heartbeat timeout: %d", cfg.WebSocket.HeartbeatTimeoutSeconds)
	}
	if cfg.WebSocket.OnlineBroadcastIntervalSeconds != 3 {
		t.Fatalf("unexpected online broadcast interval: %d", cfg.WebSocket.OnlineBroadcastIntervalSeconds)
	}
	if cfg.WebSocket.MaxMessageBytes != 16384 {
		t.Fatalf("unexpected max message bytes: %d", cfg.WebSocket.MaxMessageBytes)
	}
	if cfg.RocketMQ.Enabled {
		t.Fatal("rocketmq should be disabled by default without name servers")
	}
	if cfg.Interaction.AIInputMode != "redis_pubsub" {
		t.Fatalf("unexpected ai input mode: %s", cfg.Interaction.AIInputMode)
	}
}

func TestNormalizeHeartbeatTimeout(t *testing.T) {
	cfg := defaultConfig()
	cfg.WebSocket.HeartbeatIntervalSeconds = 20
	cfg.WebSocket.HeartbeatTimeoutSeconds = 10
	normalize(&cfg)

	if cfg.WebSocket.HeartbeatTimeoutSeconds != 60 {
		t.Fatalf("expected timeout to be interval*3, got %d", cfg.WebSocket.HeartbeatTimeoutSeconds)
	}
}

func TestNormalizeAIInputModeFallsBackToRedisPubSub(t *testing.T) {
	cfg := defaultConfig()
	cfg.Interaction.AIInputMode = "rocketmq"
	normalize(&cfg)

	if cfg.Interaction.AIInputMode != "redis_pubsub" {
		t.Fatalf("unexpected ai input mode: %s", cfg.Interaction.AIInputMode)
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV("127.0.0.1:9876, 127.0.0.2:9876,,")
	if len(got) != 2 || got[0] != "127.0.0.1:9876" || got[1] != "127.0.0.2:9876" {
		t.Fatalf("unexpected split result: %#v", got)
	}
}
