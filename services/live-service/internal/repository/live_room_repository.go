package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/yayccc/livebid/services/live-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrLiveRoomNotFound      = errors.New("live room not found")
	ErrStreamNameDuplicated  = errors.New("stream name already exists")
	ErrLiveRoomStateConflict = errors.New("live room state conflict")
	ErrLiveRoomOwnerMismatch = errors.New("live room owner mismatch")
)

type ListLiveRoomsFilter struct {
	ShopID int64
	Status model.LiveRoomStatus
	Limit  int
	Offset int
}

type LiveRoomRepository interface {
	Create(ctx context.Context, room *model.LiveRoom) error
	FindByID(ctx context.Context, id int64) (*model.LiveRoom, error)
	FindByStreamName(ctx context.Context, streamName string) (*model.LiveRoom, error)
	List(ctx context.Context, filter ListLiveRoomsFilter) ([]*model.LiveRoom, int64, error)
	Update(ctx context.Context, room *model.LiveRoom) error
}

type GormLiveRoomRepository struct {
	db *gorm.DB
}

func NewGormLiveRoomRepository(db *gorm.DB) *GormLiveRoomRepository {
	return &GormLiveRoomRepository{db: db}
}

func (r *GormLiveRoomRepository) Create(ctx context.Context, room *model.LiveRoom) error {
	err := r.db.WithContext(ctx).Create(room).Error
	return mapMySQLError(err)
}

func (r *GormLiveRoomRepository) FindByID(ctx context.Context, id int64) (*model.LiveRoom, error) {
	var room model.LiveRoom
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = 0", id).
		First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLiveRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *GormLiveRoomRepository) FindByStreamName(ctx context.Context, streamName string) (*model.LiveRoom, error) {
	var room model.LiveRoom
	err := r.db.WithContext(ctx).
		Where("stream_name = ? AND is_deleted = 0", streamName).
		First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLiveRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *GormLiveRoomRepository) List(ctx context.Context, filter ListLiveRoomsFilter) ([]*model.LiveRoom, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.LiveRoom{}).Where("is_deleted = 0")
	if filter.ShopID > 0 {
		query = query.Where("shop_id = ?", filter.ShopID)
	}
	if filter.Status > 0 {
		query = query.Where("status = ?", filter.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rooms []*model.LiveRoom
	if err := query.
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&rooms).Error; err != nil {
		return nil, 0, err
	}
	return rooms, total, nil
}

func (r *GormLiveRoomRepository) Update(ctx context.Context, room *model.LiveRoom) error {
	tx := r.db.WithContext(ctx).
		Model(&model.LiveRoom{}).
		Where("id = ? AND is_deleted = 0", room.ID).
		Updates(map[string]any{
			"title":               room.Title,
			"cover":               room.Cover,
			"description":         room.Description,
			"status":              room.Status,
			"stream_name":         room.StreamName,
			"stream_key_hash":     room.StreamKeyHash,
			"media_stream_status": room.MediaStreamStatus,
			"actual_start_time":   room.ActualStartTime,
			"actual_end_time":     room.ActualEndTime,
			"extra":               room.Extra,
		})
	if tx.Error != nil {
		return mapMySQLError(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrLiveRoomNotFound
	}
	return nil
}

func mapMySQLError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.Contains(message, "Duplicate entry") && strings.Contains(message, "uk_stream_name") {
		return ErrStreamNameDuplicated
	}
	return err
}
