// golang-migrate 版本化迁移装配(iofs 源 + postgres 驱动,SQL 经 embed.FS 内嵌)
package dao

import (
	"errors"
	"fmt"

	"github.com/alchemy-furnace/server/migration"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// newMigrator 基于当前数据库连接构造迁移器
func newMigrator() (*migrate.Migrate, error) {
	if DB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层连接失败: %w", err)
	}

	source, err := iofs.New(migration.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("加载内嵌迁移文件失败: %w", err)
	}
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("初始化 postgres 迁移驱动失败: %w", err)
	}
	return migrate.NewWithInstance("iofs", source, "postgres", driver)
}

// MigrateUp 执行全部未应用的迁移(幂等: 版本表 schema_migrations 自动跳过已执行项)
func MigrateUp() error {
	m, err := newMigrator()
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("执行迁移失败: %w", err)
	}
	return nil
}

// MigrateDown 回滚全部迁移(DROP 所有表;用于开发库重建)
func MigrateDown() error {
	m, err := newMigrator()
	if err != nil {
		return err
	}
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("回滚迁移失败: %w", err)
	}
	return nil
}

// MigrateVersion 返回当前迁移版本(诊断用)
func MigrateVersion() (uint, bool, error) {
	m, err := newMigrator()
	if err != nil {
		return 0, false, err
	}
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, dirty, err
}
