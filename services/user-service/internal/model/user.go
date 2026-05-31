package model

import (
	"time"

	"gorm.io/datatypes"
)

type UserStatus int8

const (
	UserStatusEnabled  UserStatus = 1
	UserStatusDisabled UserStatus = 2
	UserStatusBanned   UserStatus = 3
)

type Gender int8

const (
	GenderUnknown Gender = 0
	GenderMale    Gender = 1
	GenderFemale  Gender = 2
)

type User struct {
	ID           int64          `gorm:"primaryKey;column:id"`
	Username     string         `gorm:"column:username;type:varchar(64);not null;uniqueIndex:uk_username"`
	PasswordHash string         `gorm:"column:password_hash;type:varchar(255);not null"`
	Nickname     string         `gorm:"column:nickname;type:varchar(128);not null"`
	Avatar       string         `gorm:"column:avatar;type:varchar(255)"`
	Gender       Gender         `gorm:"column:gender;type:tinyint;default:0"`
	Birthday     *time.Time     `gorm:"column:birthday;type:date"`
	Phone        string         `gorm:"column:phone;type:varchar(20);index:idx_phone"`
	Email        string         `gorm:"column:email;type:varchar(128);index:idx_email"`
	Status       UserStatus     `gorm:"column:status;type:tinyint;not null;default:1;index:idx_status"`
	Extra        datatypes.JSON `gorm:"column:extra;type:json"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	IsDeleted    bool           `gorm:"column:is_deleted;type:tinyint;default:0;index"`
}

func (User) TableName() string {
	return "user"
}
