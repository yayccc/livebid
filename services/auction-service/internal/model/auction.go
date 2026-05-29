package model

import "time"

type AuctionStatus int8

const (
	// AuctionStatusPending 表示已创建但尚未加载到 Redis 运行态。
	AuctionStatusPending AuctionStatus = 0
	// AuctionStatusRunning 表示竞拍正在 Redis 中接受出价。
	AuctionStatusRunning  AuctionStatus = 1
	AuctionStatusDeal     AuctionStatus = 2
	AuctionStatusFailed   AuctionStatus = 3
	AuctionStatusCanceled AuctionStatus = 4
)

type Auction struct {
	ID           int64         `gorm:"primaryKey;column:id"`
	GoodsID      int64         `gorm:"column:goods_id;not null;uniqueIndex:uk_goods_id;index:idx_goods_id"`
	ShopID       int64         `gorm:"column:shop_id;not null;index:idx_shop_id;index:idx_shop_status,priority:1"`
	StartPrice   int64         `gorm:"column:start_price;not null"`
	BidIncrement int64         `gorm:"column:bid_increment;not null"`
	SealPrice    *int64        `gorm:"column:seal_price"`
	CurrentPrice int64         `gorm:"column:current_price;not null"`
	DealPrice    *int64        `gorm:"column:deal_price"`
	BidCount     int64         `gorm:"column:bid_count;not null;default:0"`
	Status       AuctionStatus `gorm:"column:status;type:tinyint;not null;default:0;index:idx_status;index:idx_shop_status,priority:2"`
	StartTime    *time.Time    `gorm:"column:start_time;index:idx_start_time"`
	EndTime      *time.Time    `gorm:"column:end_time;index:idx_end_time"`
	WinnerUserID *int64        `gorm:"column:winner_user_id"`
	CreatedAt    time.Time     `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time     `gorm:"column:updated_at;autoUpdateTime"`
	IsDeleted    bool          `gorm:"column:is_delete;type:tinyint;default:0;index:idx_is_delete"`
	Version      int64         `gorm:"column:version;not null;default:0"`
}

type BidRecord struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	AuctionID int64     `gorm:"column:auction_id;not null;index:idx_auction_id;index:idx_auction_price,priority:1;index:idx_auction_time,priority:1"`
	GoodsID   int64     `gorm:"column:goods_id;not null;index:idx_goods_id"`
	ShopID    int64     `gorm:"column:shop_id;not null;index:idx_shop_id"`
	UserID    int64     `gorm:"column:user_id;not null;index:idx_user_id"`
	BidPrice  int64     `gorm:"column:bid_price;not null;index:idx_auction_price,priority:2"`
	BidTime   time.Time `gorm:"column:bid_time;not null;index:idx_auction_time,priority:2"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
	IsDeleted bool      `gorm:"column:is_delete;type:tinyint;default:0;index:idx_is_delete"`
}

func (Auction) TableName() string {
	return "auction"
}

func (BidRecord) TableName() string {
	return "bid_record"
}
