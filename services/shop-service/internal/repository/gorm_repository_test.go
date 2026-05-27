package repository

import (
	"context"
	"os"
	"testing"

	"github.com/yayccc/livebid/services/shop-service/internal/config"
	"github.com/yayccc/livebid/services/shop-service/internal/model"
	"gorm.io/datatypes"
)

func TestGormShopRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("SHOP_SERVICE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set SHOP_SERVICE_TEST_MYSQL_DSN to run mysql integration test")
	}

	db, err := OpenDB(config.MySQLConfig{
		DSN:                    dsn,
		AutoMigrate:            true,
		MaxOpenConns:           5,
		MaxIdleConns:           2,
		ConnMaxLifetimeSeconds: 60,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	repo := NewGormShopRepository(db)
	ctx := context.Background()
	shop := &model.Shop{
		ID:           9001001,
		Username:     "integration_shop_9001001",
		PasswordHash: "hash",
		ShopName:     "integration shop 9001001",
		Status:       model.ShopStatusEnabled,
		AuditStatus:  model.AuditStatusPending,
		Extra:        datatypes.JSON([]byte("{}")),
	}
	_ = db.WithContext(ctx).Where("id = ?", shop.ID).Delete(&model.Shop{}).Error

	if err := repo.Create(ctx, shop); err != nil {
		t.Fatalf("create shop: %v", err)
	}
	got, err := repo.FindByUsername(ctx, shop.Username)
	if err != nil {
		t.Fatalf("find by username: %v", err)
	}
	if got.ID != shop.ID {
		t.Fatalf("unexpected shop id: got %d want %d", got.ID, shop.ID)
	}
}
