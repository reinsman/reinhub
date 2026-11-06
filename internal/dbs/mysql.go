package dbs

import (
	"fmt"

	"github.com/reinsman/reinhub/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var autoMigrateModels []interface{}

// RegisterModels 注册需要在应用启动时迁移的领域模型。
func RegisterModels(models ...interface{}) {
	autoMigrateModels = append(autoMigrateModels, models...)
}

// Init 连接数据库并迁移所有已注册的领域模型。
func Init(cfg config.Database, debug bool) (*gorm.DB, error) {
	db, err := Conn(cfg.Db, cfg.User, cfg.Pwd, cfg.Endpoint, debug)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := AutoMigrate(db); err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, fmt.Errorf("auto migrate database: %w", err)
	}
	return db, nil
}

func Conn(db, user, pass, ep string, debug bool) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, ep, db)
	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}
	if debug {
		conn = conn.Debug()
	}
	return conn, nil
}

// AutoMigrate 迁移所有在包初始化阶段注册的模型。
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not connected")
	}
	if len(autoMigrateModels) == 0 {
		return nil
	}
	return db.AutoMigrate(autoMigrateModels...)
}
