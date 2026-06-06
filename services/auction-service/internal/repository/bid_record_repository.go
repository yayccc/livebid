package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"gorm.io/gorm"
)

var ErrBidRecordDuplicated = errors.New("出价记录已存在，跳过重复写入")

type ListBidRecordFilter struct {
	AuctionID int64
	Page      int
	PageSize  int
}

type BidRecordRepository interface {
	Create(ctx context.Context, record *model.BidRecord) error
	CreateIfNotExists(ctx context.Context, record *model.BidRecord) error
	ListByAuction(ctx context.Context, filter ListBidRecordFilter) ([]*model.BidRecord, int64, error)
	CountByShopBetween(ctx context.Context, shopID int64, start time.Time, end time.Time) (int64, error)
}

type GormBidRecordRepository struct {
	db *gorm.DB
}

func NewGormBidRecordRepository(db *gorm.DB) *GormBidRecordRepository {
	return &GormBidRecordRepository{db: db}
}

func (r *GormBidRecordRepository) Create(ctx context.Context, record *model.BidRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *GormBidRecordRepository) CreateIfNotExists(ctx context.Context, record *model.BidRecord) error {
	// RocketMQ 至少一次投递会导致重复消费，主键重复代表该出价记录已落库。
	err := r.db.WithContext(ctx).Create(record).Error
	if err != nil && strings.Contains(err.Error(), "Duplicate entry") {
		return ErrBidRecordDuplicated
	}
	return err
}

func (r *GormBidRecordRepository) ListByAuction(ctx context.Context, filter ListBidRecordFilter) ([]*model.BidRecord, int64, error) {
	page, pageSize := normalizePagination(filter.Page, filter.PageSize, 20)
	query := r.db.WithContext(ctx).
		Model(&model.BidRecord{}).
		Where("auction_id = ? AND is_delete = 0", filter.AuctionID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.BidRecord
	err := query.Order("bid_time DESC, id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *GormBidRecordRepository) CountByShopBetween(ctx context.Context, shopID int64, start time.Time, end time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.BidRecord{}).
		Where("shop_id = ? AND is_delete = 0 AND bid_time >= ? AND bid_time < ?", shopID, start, end).
		Count(&total).Error
	return total, err
}
