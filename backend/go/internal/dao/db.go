// Package dao 数据库访问层(新结构): 连接管理 + golang-migrate 版本化迁移 + 种子
// schema 来源为 migration/*.sql;GORM 不再承担建表职责
package dao

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alchemy-furnace/server/internal/configuration"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库连接实例
var DB *gorm.DB

// InitDatabase 初始化数据库连接(不执行迁移/种子,二者由 cobra 子命令显式触发)
func InitDatabase(cfg *configuration.DatabaseConfig) error {
	dsn := cfg.DSN()
	log.Printf("[炼丹炉] 正在连接数据库: host=%s port=%d dbname=%s", cfg.Host, cfg.Port, cfg.DBName)

	logLevel := logger.Info
	if configuration.Configuration.Server.Mode == "release" {
		logLevel = logger.Error
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(log.Writer(), "[GORM] ", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logLevel,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库连接失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	DB = db
	log.Println("[炼丹炉] 数据库连接成功,仙炉已就位")
	return nil
}

// GetDB 获取数据库实例,未初始化则 panic
func GetDB() *gorm.DB {
	if DB == nil {
		log.Fatal("[炼丹炉] 致命错误: 数据库未初始化,请先调用 InitDatabase")
	}
	return DB
}

// Transaction 在事务中执行函数,fn 返回错误则回滚
func Transaction(fn func(tx *gorm.DB) error) error {
	return DB.Transaction(fn)
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	log.Println("[炼丹炉] 关闭数据库连接")
	return sqlDB.Close()
}
