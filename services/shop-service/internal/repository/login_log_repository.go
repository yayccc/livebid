package repository

import (
	"context"

	"github.com/yayccc/livebid/services/shop-service/internal/model"
	"gorm.io/gorm"
)

type ShopLoginLogRepository interface {
	Create(ctx context.Context, log *model.ShopLoginLog) error
}

type GormShopLoginLogRepository struct {
	db *gorm.DB
}

func NewGormShopLoginLogRepository(db *gorm.DB) *GormShopLoginLogRepository {
	return &GormShopLoginLogRepository{db: db}
}

func (r *GormShopLoginLogRepository) Create(ctx context.Context, log *model.ShopLoginLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}
