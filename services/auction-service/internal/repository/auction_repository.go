package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrAuctionNotFound        = errors.New("auction not found")
	ErrAuctionDuplicated      = errors.New("auction already exists for goods")
	ErrInvalidAuctionState    = errors.New("invalid auction state")
	ErrBidRecordNotFound      = errors.New("bid record not found")
	ErrOutdatedAuctionVersion = errors.New("outdated auction version")
)

type ListAuctionFilter struct {
	ShopID   int64
	Status   *model.AuctionStatus
	Page     int
	PageSize int
}

type AuctionRepository interface {
	Create(ctx context.Context, auction *model.Auction) error
	FindByID(ctx context.Context, id int64) (*model.Auction, error)
	FindByIDForShop(ctx context.Context, id int64, shopID int64) (*model.Auction, error)
	FindByGoodsID(ctx context.Context, goodsID int64) (*model.Auction, error)
	ListByShop(ctx context.Context, filter ListAuctionFilter) ([]*model.Auction, int64, error)
	UpdatePendingConfig(ctx context.Context, auction *model.Auction) error
	UpdateStatusSnapshot(ctx context.Context, auction *model.Auction) error
	Delete(ctx context.Context, id int64, shopID int64) error
}

type GormAuctionRepository struct {
	db *gorm.DB
}

func NewGormAuctionRepository(db *gorm.DB) *GormAuctionRepository {
	return &GormAuctionRepository{db: db}
}

func (r *GormAuctionRepository) Create(ctx context.Context, auction *model.Auction) error {
	return mapMySQLError(r.db.WithContext(ctx).Create(auction).Error)
}

func (r *GormAuctionRepository) FindByID(ctx context.Context, id int64) (*model.Auction, error) {
	var auction model.Auction
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_delete = 0", id).
		First(&auction).Error
	return finishFindAuction(&auction, err)
}

func (r *GormAuctionRepository) FindByIDForShop(ctx context.Context, id int64, shopID int64) (*model.Auction, error) {
	var auction model.Auction
	err := r.db.WithContext(ctx).
		Where("id = ? AND shop_id = ? AND is_delete = 0", id, shopID).
		First(&auction).Error
	return finishFindAuction(&auction, err)
}

func (r *GormAuctionRepository) FindByGoodsID(ctx context.Context, goodsID int64) (*model.Auction, error) {
	var auction model.Auction
	err := r.db.WithContext(ctx).
		Where("goods_id = ? AND is_delete = 0", goodsID).
		First(&auction).Error
	return finishFindAuction(&auction, err)
}

func (r *GormAuctionRepository) ListByShop(ctx context.Context, filter ListAuctionFilter) ([]*model.Auction, int64, error) {
	page, pageSize := normalizePagination(filter.Page, filter.PageSize, 10)
	query := r.db.WithContext(ctx).
		Model(&model.Auction{}).
		Where("shop_id = ? AND is_delete = 0", filter.ShopID)
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Auction
	err := query.Order("created_at DESC, id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *GormAuctionRepository) UpdatePendingConfig(ctx context.Context, auction *model.Auction) error {
	// 只有待开始竞拍允许修改核心规则，避免运行中竞拍被改价。
	updates := map[string]any{
		"start_price":   auction.StartPrice,
		"bid_increment": auction.BidIncrement,
		"seal_price":    auction.SealPrice,
		"current_price": auction.CurrentPrice,
		"start_time":    auction.StartTime,
		"end_time":      auction.EndTime,
	}
	tx := r.db.WithContext(ctx).
		Model(&model.Auction{}).
		Where("id = ? AND shop_id = ? AND status = ? AND is_delete = 0", auction.ID, auction.ShopID, model.AuctionStatusPending).
		Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrInvalidAuctionState
	}
	return nil
}

func (r *GormAuctionRepository) UpdateStatusSnapshot(ctx context.Context, auction *model.Auction) error {
	// 事件可能乱序或重复到达，只接受版本号更大的快照，防止旧消息覆盖新状态。
	updates := map[string]any{
		"current_price":  auction.CurrentPrice,
		"deal_price":     auction.DealPrice,
		"bid_count":      auction.BidCount,
		"status":         auction.Status,
		"start_time":     auction.StartTime,
		"end_time":       auction.EndTime,
		"winner_user_id": auction.WinnerUserID,
		"version":        auction.Version,
	}
	tx := r.db.WithContext(ctx).
		Model(&model.Auction{}).
		Where("id = ? AND is_delete = 0 AND version < ?", auction.ID, auction.Version).
		Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrOutdatedAuctionVersion
	}
	return nil
}

func (r *GormAuctionRepository) Delete(ctx context.Context, id int64, shopID int64) error {
	// 删除仅做主表逻辑删除，出价记录保留用于审计。
	tx := r.db.WithContext(ctx).
		Model(&model.Auction{}).
		Where("id = ? AND shop_id = ? AND status IN ? AND is_delete = 0", id, shopID, []model.AuctionStatus{
			model.AuctionStatusPending,
			model.AuctionStatusFailed,
			model.AuctionStatusCanceled,
		}).
		Update("is_delete", true)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrInvalidAuctionState
	}
	return nil
}

func finishFindAuction(auction *model.Auction, err error) (*model.Auction, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAuctionNotFound
	}
	if err != nil {
		return nil, err
	}
	return auction, nil
}

func mapMySQLError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "Duplicate entry") && strings.Contains(err.Error(), "uk_goods_id") {
		return ErrAuctionDuplicated
	}
	return err
}

func normalizePagination(page int, pageSize int, defaultPageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
