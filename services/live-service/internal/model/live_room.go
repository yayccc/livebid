package model

import (
	"time"

	"gorm.io/datatypes"
)

type LiveRoomStatus int8

const (
	LiveRoomStatusNotLive LiveRoomStatus = 1
	LiveRoomStatusLiving  LiveRoomStatus = 2
)

type MediaStreamStatus int8

const (
	MediaStreamStatusOffline MediaStreamStatus = 0
	MediaStreamStatusOnline  MediaStreamStatus = 1
)

type LiveRoom struct {
	ID                int64             `gorm:"primaryKey;column:id"`
	ShopID            int64             `gorm:"column:shop_id;not null;index:idx_shop_id;index:idx_shop_status,priority:1"`
	Title             string            `gorm:"column:title;type:varchar(128);not null"`
	Cover             string            `gorm:"column:cover;type:varchar(255)"`
	Description       string            `gorm:"column:description;type:text"`
	Status            LiveRoomStatus    `gorm:"column:status;type:tinyint;not null;default:1;index:idx_status;index:idx_shop_status,priority:2"`
	StreamName        string            `gorm:"column:stream_name;type:varchar(128);not null;uniqueIndex:uk_stream_name"`
	StreamKeyHash     string            `gorm:"column:stream_key_hash;type:varchar(255);not null"`
	MediaStreamStatus MediaStreamStatus `gorm:"column:media_stream_status;type:tinyint;not null;default:0;index:idx_media_stream_status"`
	ActualStartTime   *time.Time        `gorm:"column:actual_start_time"`
	ActualEndTime     *time.Time        `gorm:"column:actual_end_time"`
	Extra             datatypes.JSON    `gorm:"column:extra;type:json"`
	CreatedAt         time.Time         `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time         `gorm:"column:updated_at;autoUpdateTime"`
	IsDeleted         bool              `gorm:"column:is_deleted;type:tinyint;not null;default:0;index"`
}

func (LiveRoom) TableName() string {
	return "live_room"
}
