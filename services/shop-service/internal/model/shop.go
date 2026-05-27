package model

import (
	"time"

	"gorm.io/datatypes"
)

type ShopStatus int8

const (
	ShopStatusEnabled  ShopStatus = 1
	ShopStatusDisabled ShopStatus = 2
	ShopStatusBanned   ShopStatus = 3
	ShopStatusAuditing ShopStatus = 4
)

type AuditStatus int8

const (
	AuditStatusPending  AuditStatus = 0
	AuditStatusPassed   AuditStatus = 1
	AuditStatusRejected AuditStatus = 2
)

type Shop struct {
	ID           int64          `gorm:"primaryKey;column:id"`
	Username     string         `gorm:"column:username;type:varchar(64);not null;uniqueIndex:uk_username"`
	PasswordHash string         `gorm:"column:password_hash;type:varchar(255);not null"`
	ShopName     string         `gorm:"column:shop_name;type:varchar(128);not null;uniqueIndex:uk_shop_name"`
	Logo         string         `gorm:"column:logo;type:varchar(255)"`
	Description  string         `gorm:"column:description;type:text"`
	Phone        string         `gorm:"column:phone;type:varchar(20)"`
	Email        string         `gorm:"column:email;type:varchar(128)"`
	Status       ShopStatus     `gorm:"column:status;type:tinyint;not null;default:1;index:idx_status"`
	AuditStatus  AuditStatus    `gorm:"column:audit_status;type:tinyint;default:0"`
	AuditReason  string         `gorm:"column:audit_reason;type:varchar(255)"`
	Extra        datatypes.JSON `gorm:"column:extra;type:json"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	IsDeleted    bool           `gorm:"column:is_deleted;type:tinyint;default:0;index"`
}

type ShopLoginLog struct {
	ID          int64          `gorm:"primaryKey;column:id"`
	ShopID      int64          `gorm:"column:shop_id;not null;index:idx_shop_id;index:idx_shop_time,priority:1"`
	LoginIP     string         `gorm:"column:login_ip;type:varchar(64)"`
	UserAgent   string         `gorm:"column:user_agent;type:varchar(255)"`
	LoginTime   time.Time      `gorm:"column:login_time;not null;index:idx_login_time;index:idx_shop_time,priority:2"`
	LoginResult int8           `gorm:"column:login_result;type:tinyint;not null"`
	Extra       datatypes.JSON `gorm:"column:extra;type:json"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	IsDeleted   bool           `gorm:"column:is_deleted;type:tinyint;default:0;index"`
}

func (Shop) TableName() string {
	return "shop"
}

func (ShopLoginLog) TableName() string {
	return "shop_login_log"
}
