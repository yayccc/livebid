package repository

import (
	"context"
	"os"
	"testing"

	"github.com/yayccc/livebid/services/goods-service/internal/config"
	"github.com/yayccc/livebid/services/goods-service/internal/model"
)

func TestGormGoodsRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("GOODS_SERVICE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set GOODS_SERVICE_TEST_MYSQL_DSN to run mysql integration test")
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

	repo := NewGormGoodsRepository(db)
	ctx := context.Background()
	goods := &model.Goods{
		ID:          9002001,
		ShopID:      9001,
		Title:       "integration goods 9002001",
		CoverURL:    "https://example.com/goods.jpg",
		Description: "integration goods",
		Status:      model.GoodsStatusOffSale,
	}
	_ = db.WithContext(ctx).Where("id = ?", goods.ID).Delete(&model.Goods{}).Error

	if err := repo.Create(ctx, goods); err != nil {
		t.Fatalf("create goods: %v", err)
	}
	got, err := repo.FindByIDForShop(ctx, goods.ID, goods.ShopID)
	if err != nil {
		t.Fatalf("find goods: %v", err)
	}
	if got.ID != goods.ID {
		t.Fatalf("unexpected goods id: got %d want %d", got.ID, goods.ID)
	}
}
