package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/yayccc/livebid/services/goods-service/internal/model"
	"gorm.io/gorm"
)

var ErrGoodsNotFound = errors.New("goods not found")

type ListGoodsFilter struct {
	Page      int
	PageSize  int
	Keyword   string
	ShopID    *int64
	IsDeleted *bool
	Status    *model.GoodsStatus
}

type GoodsRepository interface {
	Create(ctx context.Context, goods *model.Goods) error
	FindByID(ctx context.Context, id int64) (*model.Goods, error)
	FindByIDForShop(ctx context.Context, id int64, shopID int64) (*model.Goods, error)
	Update(ctx context.Context, goods *model.Goods) error
	Delete(ctx context.Context, id int64, shopID int64) error
	List(ctx context.Context, filter ListGoodsFilter) ([]*model.Goods, int64, error)
	BatchFindByIDs(ctx context.Context, ids []int64) ([]*model.Goods, error)
	UpdateStatus(ctx context.Context, id int64, shopID int64, status model.GoodsStatus) error
}

type GormGoodsRepository struct {
	db *gorm.DB
}

func NewGormGoodsRepository(db *gorm.DB) *GormGoodsRepository {
	return &GormGoodsRepository{db: db}
}

func (r *GormGoodsRepository) Create(ctx context.Context, goods *model.Goods) error {
	return r.db.WithContext(ctx).Create(goods).Error
}

func (r *GormGoodsRepository) FindByID(ctx context.Context, id int64) (*model.Goods, error) {
	var goods model.Goods
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = 0", id).
		First(&goods).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGoodsNotFound
	}
	if err != nil {
		return nil, err
	}
	return &goods, nil
}

func (r *GormGoodsRepository) FindByIDForShop(ctx context.Context, id int64, shopID int64) (*model.Goods, error) {
	var goods model.Goods
	err := r.db.WithContext(ctx).
		Where("id = ? AND shop_id = ? AND is_deleted = 0", id, shopID).
		First(&goods).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGoodsNotFound
	}
	if err != nil {
		return nil, err
	}
	return &goods, nil
}

func (r *GormGoodsRepository) Update(ctx context.Context, goods *model.Goods) error {
	tx := r.db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("id = ? AND shop_id = ? AND is_deleted = 0", goods.ID, goods.ShopID).
		Updates(map[string]any{
			"title":       goods.Title,
			"cover_url":   goods.CoverURL,
			"description": goods.Description,
		})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrGoodsNotFound
	}
	return nil
}

func (r *GormGoodsRepository) Delete(ctx context.Context, id int64, shopID int64) error {
	tx := r.db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("id = ? AND shop_id = ? AND is_deleted = 0", id, shopID).
		Update("is_deleted", true)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrGoodsNotFound
	}
	return nil
}

func (r *GormGoodsRepository) List(ctx context.Context, filter ListGoodsFilter) ([]*model.Goods, int64, error) {
	page, pageSize := normalizePagination(filter.Page, filter.PageSize)
	query := r.db.WithContext(ctx).Model(&model.Goods{})
	query = applyGoodsFilter(query, filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*model.Goods
	err := query.
		Order("created_at DESC, id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *GormGoodsRepository) BatchFindByIDs(ctx context.Context, ids []int64) ([]*model.Goods, error) {
	var list []*model.Goods
	err := r.db.WithContext(ctx).
		Where("id IN ? AND is_deleted = 0", ids).
		Order("id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *GormGoodsRepository) UpdateStatus(ctx context.Context, id int64, shopID int64, status model.GoodsStatus) error {
	tx := r.db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("id = ? AND shop_id = ? AND is_deleted = 0", id, shopID).
		Update("status", status)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrGoodsNotFound
	}
	return nil
}

func applyGoodsFilter(query *gorm.DB, filter ListGoodsFilter) *gorm.DB {
	isDeleted := false
	if filter.IsDeleted != nil {
		isDeleted = *filter.IsDeleted
	}
	query = query.Where("is_deleted = ?", isDeleted)

	if filter.ShopID != nil {
		query = query.Where("shop_id = ?", *filter.ShopID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%")
	}
	return query
}

func normalizePagination(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
