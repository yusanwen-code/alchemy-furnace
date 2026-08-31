// 任务 8 测试：迁移摘要只读查询（升级用户展示）
// 读迁移完成标记 ReportJSON（非实时计数）；无标记 → Migrated=false；
// fresh 安装标记 → 不展示但可读；legacy 标记 → 完整计数 + 备份路径 + 完成时间。
package pill_inventory_service

import (
	"context"
	"testing"

	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/model"
)

func TestMigrationSummaryLegacyReport(t *testing.T) {
	svc, db := newTestSvc(t)
	if err := db.Create(&model.PillMigrationState{
		Key: dao.PillInventoryMigrationKey,
		ReportJSON: model.JSONMap{
			"is_fresh_install": false,
			"legacy_pills":     5,
			"legacy_binds":     3,
			"recipes":          5,
			"available_items":  1,
			"history_items":    4,
			"effects":          3,
			"backup_path":      "/tmp/backups/pill-inventory-v1-20260831.db",
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	s, err := svc.MigrationSummary(context.Background())
	if err != nil {
		t.Fatalf("MigrationSummary: %v", err)
	}
	if !s.Migrated {
		t.Fatal("存在完成标记时应 Migrated=true")
	}
	if s.IsFreshInstall {
		t.Fatal("legacy 报告不得标记为 fresh")
	}
	if s.LegacyPills != 5 || s.LegacyBinds != 3 {
		t.Fatalf("旧定义/绑定=%d/%d, want 5/3", s.LegacyPills, s.LegacyBinds)
	}
	if s.Recipes != 5 || s.AvailableItems != 1 || s.HistoryItems != 4 || s.Effects != 3 {
		t.Fatalf("计数异常: recipes=%d avail=%d history=%d effects=%d",
			s.Recipes, s.AvailableItems, s.HistoryItems, s.Effects)
	}
	if s.BackupPath != "/tmp/backups/pill-inventory-v1-20260831.db" {
		t.Fatalf("backup_path=%q 与报告不符", s.BackupPath)
	}
	if s.CompletedAt == "" {
		t.Fatal("完成标记应带完成时间")
	}
}

func TestMigrationSummaryFreshInstall(t *testing.T) {
	svc, db := newTestSvc(t)
	if err := db.Create(&model.PillMigrationState{
		Key:        dao.PillInventoryMigrationKey,
		ReportJSON: model.JSONMap{"is_fresh_install": true, "legacy_pills": 0},
	}).Error; err != nil {
		t.Fatal(err)
	}

	s, err := svc.MigrationSummary(context.Background())
	if err != nil {
		t.Fatalf("MigrationSummary: %v", err)
	}
	if !s.Migrated || !s.IsFreshInstall {
		t.Fatalf("fresh 标记应 Migrated=true IsFreshInstall=true, got %v/%v", s.Migrated, s.IsFreshInstall)
	}
	if s.Recipes != 0 || s.AvailableItems != 0 {
		t.Fatalf("fresh 报告不应有迁移计数: recipes=%d avail=%d", s.Recipes, s.AvailableItems)
	}
}

func TestMigrationSummaryNoState(t *testing.T) {
	svc, _ := newTestSvc(t)
	s, err := svc.MigrationSummary(context.Background())
	if err != nil {
		t.Fatalf("MigrationSummary: %v", err)
	}
	if s.Migrated {
		t.Fatal("无完成标记时应 Migrated=false（前端据此不展示）")
	}
	if s.BackupPath != "" || s.CompletedAt != "" {
		t.Fatalf("无标记时不应有备份路径/完成时间: %q/%q", s.BackupPath, s.CompletedAt)
	}
}
