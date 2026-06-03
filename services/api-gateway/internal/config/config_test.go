package config

import "testing"

func TestLoadAppliesTargetAndNacosEnvOverrides(t *testing.T) {
	t.Setenv("API_GATEWAY_CONFIG", "not-exists.yaml")
	t.Setenv("API_GATEWAY_SHOP_SERVICE_TARGET", "nacos:///shop-service")
	t.Setenv("API_GATEWAY_GOODS_SERVICE_TARGET", "nacos:///goods-service")
	t.Setenv("API_GATEWAY_NACOS_ENABLED", "true")
	t.Setenv("API_GATEWAY_NACOS_SERVERS", "nacos:8848")
	t.Setenv("API_GATEWAY_CONFIG_CENTER_ENABLED", "true")

	cfg := Load()
	if cfg.ShopService.Target != "nacos:///shop-service" {
		t.Fatalf("shop target = %q", cfg.ShopService.Target)
	}
	if cfg.GoodsService.Target != "nacos:///goods-service" {
		t.Fatalf("goods target = %q", cfg.GoodsService.Target)
	}
	if !cfg.Nacos.Enabled {
		t.Fatalf("expected nacos enabled")
	}
	if len(cfg.Nacos.Servers) != 1 || cfg.Nacos.Servers[0].Host != "nacos" || cfg.Nacos.Servers[0].Port != 8848 {
		t.Fatalf("nacos servers = %+v", cfg.Nacos.Servers)
	}
	if !cfg.ConfigCenter.Enabled {
		t.Fatalf("expected config center enabled")
	}
}
