package model

import (
	"time"

	"gorm.io/datatypes"
)

type UserAddress struct {
	ID            int64          `gorm:"primaryKey;column:id"`
	UserID        int64          `gorm:"column:user_id;not null;index:idx_user_id;index:idx_user_default,priority:1"`
	ReceiverName  string         `gorm:"column:receiver_name;type:varchar(64);not null"`
	ReceiverPhone string         `gorm:"column:receiver_phone;type:varchar(20);not null"`
	Province      string         `gorm:"column:province;type:varchar(64);not null"`
	City          string         `gorm:"column:city;type:varchar(64);not null"`
	District      string         `gorm:"column:district;type:varchar(64);not null"`
	DetailAddress string         `gorm:"column:detail_address;type:varchar(255);not null"`
	PostalCode    string         `gorm:"column:postal_code;type:varchar(20)"`
	IsDefault     bool           `gorm:"column:is_default;type:tinyint;not null;default:0;index:idx_user_default,priority:2"`
	Extra         datatypes.JSON `gorm:"column:extra;type:json"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	IsDeleted     bool           `gorm:"column:is_deleted;type:tinyint;default:0;index"`
}

func (UserAddress) TableName() string {
	return "user_address"
}
