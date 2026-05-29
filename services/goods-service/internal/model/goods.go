package model

import "time"

type GoodsStatus int8

const (
	GoodsStatusOffSale GoodsStatus = 0
	GoodsStatusOnSale  GoodsStatus = 1
)

type Goods struct {
	ID          int64       `gorm:"primaryKey;column:id"`
	ShopID      int64       `gorm:"column:shop_id;not null;index:idx_shop_id"`
	Title       string      `gorm:"column:title;type:varchar(255);not null"`
	CoverURL    string      `gorm:"column:cover_url;type:varchar(500)"`
	Description string      `gorm:"column:description;type:text"`
	Status      GoodsStatus `gorm:"column:status;type:tinyint;not null;default:0;index:idx_status"`
	CreatedAt   time.Time   `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time   `gorm:"column:updated_at;autoUpdateTime"`
	IsDeleted   bool        `gorm:"column:is_deleted;type:tinyint;default:0;index"`
}

func (Goods) TableName() string {
	return "goods"
}
