package testutil

import (
	"fmt"

	"github.com/reinsman/reinhub/internal/config"
	"github.com/reinsman/reinhub/internal/dbs"
	"gorm.io/gorm"
)

// InitTest 初始化测试环境（连接数据库）
// models: 需要自动建表的业务模型
func InitTest(models ...interface{}) *gorm.DB {
	cfg, err := config.Load("../bin/conf.toml")
	if err != nil {
		panic(fmt.Errorf("config.Load return error: %s", err.Error()))
	}
	db, err := dbs.Conn(cfg.Db.Db,
		cfg.Db.User,
		cfg.Db.Pwd,
		cfg.Db.Endpoint,
		false)
	if err != nil {
		panic(fmt.Errorf("dbs.Conn return error: %s", err.Error()))
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			panic(fmt.Errorf("dbs.AutoMigrate return error: %s", err.Error()))
		}
	}
	return db
}
