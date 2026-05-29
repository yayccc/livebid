package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/yayccc/livebid/services/auction-service/internal/model"
	"gorm.io/gorm"
)

type EventLogRepository interface {
	BeginConsume(ctx context.Context, log *model.AuctionEventConsumeLog) (bool, error)
	MarkSucceeded(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string, err error) error
}

type GormEventLogRepository struct {
	db *gorm.DB
}

func NewGormEventLogRepository(db *gorm.DB) *GormEventLogRepository {
	return &GormEventLogRepository{db: db}
}

func (r *GormEventLogRepository) BeginConsume(ctx context.Context, log *model.AuctionEventConsumeLog) (bool, error) {
	// 插入成功代表当前实例抢到处理权；主键冲突则按历史状态决定是否跳过或重试。
	err := r.db.WithContext(ctx).Create(log).Error
	if err == nil {
		return true, nil
	}
	if !strings.Contains(err.Error(), "Duplicate entry") {
		return false, err
	}

	var existing model.AuctionEventConsumeLog
	if err := r.db.WithContext(ctx).Where("event_id = ?", log.EventID).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrAuctionNotFound
		}
		return false, err
	}
	if existing.Status == model.AuctionEventConsumeSucceeded || existing.Status == model.AuctionEventConsumeProcessing {
		// 已成功或已有实例处理中，都直接 ack，避免重复业务写入。
		return false, nil
	}
	// 失败消息允许再次进入处理中，由 RocketMQ 控制重投次数。
	tx := r.db.WithContext(ctx).
		Model(&model.AuctionEventConsumeLog{}).
		Where("event_id = ? AND status = ?", log.EventID, model.AuctionEventConsumeFailed).
		Updates(map[string]any{
			"status":      model.AuctionEventConsumeProcessing,
			"retry_times": gorm.Expr("retry_times + 1"),
			"last_error":  "",
		})
	return tx.RowsAffected > 0, tx.Error
}

func (r *GormEventLogRepository) MarkSucceeded(ctx context.Context, eventID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.AuctionEventConsumeLog{}).
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status":      model.AuctionEventConsumeSucceeded,
			"last_error":  "",
			"consumed_at": &now,
		}).Error
}

func (r *GormEventLogRepository) MarkFailed(ctx context.Context, eventID string, consumeErr error) error {
	message := consumeErr.Error()
	if len(message) > 500 {
		message = message[:500]
	}
	return r.db.WithContext(ctx).
		Model(&model.AuctionEventConsumeLog{}).
		Where("event_id = ?", eventID).
		Updates(map[string]any{
			"status":     model.AuctionEventConsumeFailed,
			"last_error": message,
		}).Error
}
