package repository

import (
	"database/sql"
	"time"

	"github.com/yayccc/livebid/services/shop-service/internal/config"
	"github.com/yayccc/livebid/services/shop-service/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func OpenDB(cfg config.MySQLConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	configurePool(sqlDB, cfg)

	if cfg.AutoMigrate {
		if err := db.AutoMigrate(&model.Shop{}, &model.ShopLoginLog{}); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func configurePool(db *sql.DB, cfg config.MySQLConfig) {
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSeconds) * time.Second)
}
