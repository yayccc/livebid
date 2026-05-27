package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/yayccc/livebid/services/shop-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrShopNotFound       = errors.New("shop not found")
	ErrUsernameDuplicated = errors.New("username already exists")
	ErrShopNameDuplicated = errors.New("shop name already exists")
)

type ShopRepository interface {
	Create(ctx context.Context, shop *model.Shop) error
	FindByID(ctx context.Context, id int64) (*model.Shop, error)
	FindByUsername(ctx context.Context, username string) (*model.Shop, error)
	Update(ctx context.Context, shop *model.Shop) error
}

type GormShopRepository struct {
	db *gorm.DB
}

func NewGormShopRepository(db *gorm.DB) *GormShopRepository {
	return &GormShopRepository{db: db}
}

func (r *GormShopRepository) Create(ctx context.Context, shop *model.Shop) error {
	err := r.db.WithContext(ctx).Create(shop).Error
	return mapMySQLError(err)
}

func (r *GormShopRepository) FindByID(ctx context.Context, id int64) (*model.Shop, error) {
	var shop model.Shop
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = 0", id).
		First(&shop).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrShopNotFound
	}
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *GormShopRepository) FindByUsername(ctx context.Context, username string) (*model.Shop, error) {
	var shop model.Shop
	err := r.db.WithContext(ctx).
		Where("username = ? AND is_deleted = 0", username).
		First(&shop).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrShopNotFound
	}
	if err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *GormShopRepository) Update(ctx context.Context, shop *model.Shop) error {
	tx := r.db.WithContext(ctx).
		Model(&model.Shop{}).
		Where("id = ? AND is_deleted = 0", shop.ID).
		Updates(shop)
	if tx.Error != nil {
		return mapMySQLError(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrShopNotFound
	}
	return nil
}

func mapMySQLError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.Contains(message, "Duplicate entry") && strings.Contains(message, "uk_username") {
		return ErrUsernameDuplicated
	}
	if strings.Contains(message, "Duplicate entry") && strings.Contains(message, "uk_shop_name") {
		return ErrShopNameDuplicated
	}
	return err
}
