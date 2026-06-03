package config

import "testing"

func TestLoadAppliesEnvOverrides(t *testing.T) {
	t.Setenv("AUCTION_SERVICE_CONFIG", "not-exists.yaml")
	t.Setenv("AUCTION_SERVICE_GRPC_ADDR", ":19003")
	t.Setenv("AUCTION_SERVICE_REDIS_ADDR", "127.0.0.1:16379")
	t.Setenv("AUCTION_SERVICE_GOODS_ADDR", "127.0.0.1:19002")
	t.Setenv("AUCTION_SERVICE_GOODS_TARGET", "nacos:///goods-service")
	t.Setenv("AUCTION_SERVICE_WORKER_ID", "31")
	t.Setenv("AUCTION_SERVICE_REGISTRY_ENABLED", "true")

	cfg := Load()
	if cfg.GRPC.Addr != ":19003" {
		t.Fatalf("grpc addr = %q", cfg.GRPC.Addr)
	}
	if cfg.Redis.Addr != "127.0.0.1:16379" {
		t.Fatalf("redis addr = %q", cfg.Redis.Addr)
	}
	if cfg.Goods.Addr != "127.0.0.1:19002" {
		t.Fatalf("goods addr = %q", cfg.Goods.Addr)
	}
	if cfg.Goods.Target != "nacos:///goods-service" {
		t.Fatalf("goods target = %q", cfg.Goods.Target)
	}
	if !cfg.Registry.Enabled {
		t.Fatalf("expected registry enabled")
	}
	if cfg.WorkerID != 31 {
		t.Fatalf("worker id = %d", cfg.WorkerID)
	}
}
