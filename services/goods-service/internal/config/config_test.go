package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFileAndEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
env: test
workerID: 7
grpc:
  addr: ":9102"
mysql:
  dsn: "root:pass@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
  autoMigrate: false
  maxOpenConns: 3
  maxIdleConns: 2
  connMaxLifetimeSeconds: 60
log:
  level: info
  encoding: json
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("GOODS_SERVICE_CONFIG", path)
	t.Setenv("GOODS_SERVICE_GRPC_ADDR", ":9202")
	t.Setenv("GOODS_SERVICE_MYSQL_AUTO_MIGRATE", "true")

	cfg := Load()
	if cfg.Env != "test" {
		t.Fatalf("env mismatch: %s", cfg.Env)
	}
	if cfg.GRPC.Addr != ":9202" {
		t.Fatalf("grpc addr mismatch: %s", cfg.GRPC.Addr)
	}
	if !cfg.MySQL.AutoMigrate {
		t.Fatalf("expected env override to enable auto migrate")
	}
	if cfg.MySQL.MaxOpenConns != 3 {
		t.Fatalf("max open conns mismatch: %d", cfg.MySQL.MaxOpenConns)
	}
	if cfg.Log.ServiceName != ServiceName {
		t.Fatalf("log service name mismatch: %s", cfg.Log.ServiceName)
	}
}
