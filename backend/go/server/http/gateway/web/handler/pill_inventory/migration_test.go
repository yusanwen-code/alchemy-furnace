// 任务 8 测试：迁移摘要端点（升级用户展示）
// 覆盖：legacy 报告完整字段、fresh 报告不展示、无标记 migrated=false。
package pill_inventory

import (
	"net/http"
	"testing"

	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/model"
)

// TestMigrationSummaryEndpointLegacy legacy 迁移报告 → 200 + 完整计数
func TestMigrationSummaryEndpointLegacy(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter()

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
			"backup_path":      "/data/backups/pill-inventory-v1.db",
		},
	}).Error; err != nil {
		t.Fatalf("造迁移标记失败: %v", err)
	}

	status, envelope := doJSON(t, r, http.MethodGet, "/api/v1/migration-summary", "", "")
	if status != http.StatusOK {
		t.Fatalf("摘要查询期望 200, 实际 %d, body: %v", status, envelope)
	}
	data, _ := envelope["data"].(map[string]interface{})
	if data["migrated"] != true || data["is_fresh_install"] != false {
		t.Fatalf("标记断言失败: %v", data)
	}
	if data["recipes"] != float64(5) || data["available_items"] != float64(1) ||
		data["effects"] != float64(3) || data["history_items"] != float64(4) {
		t.Fatalf("计数断言失败: %v", data)
	}
	if data["backup_path"] != "/data/backups/pill-inventory-v1.db" {
		t.Fatalf("备份路径缺失: %v", data)
	}
	if completed, _ := data["completed_at"].(string); completed == "" {
		t.Fatalf("缺少完成时间: %v", data)
	}
}

// TestMigrationSummaryEndpointFresh fresh 安装标记 → migrated=true 但 is_fresh_install=true
func TestMigrationSummaryEndpointFresh(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter()

	if err := db.Create(&model.PillMigrationState{
		Key:        dao.PillInventoryMigrationKey,
		ReportJSON: model.JSONMap{"is_fresh_install": true, "legacy_pills": 0},
	}).Error; err != nil {
		t.Fatalf("造迁移标记失败: %v", err)
	}

	status, envelope := doJSON(t, r, http.MethodGet, "/api/v1/migration-summary", "", "")
	if status != http.StatusOK {
		t.Fatalf("摘要查询期望 200, 实际 %d, body: %v", status, envelope)
	}
	data, _ := envelope["data"].(map[string]interface{})
	if data["migrated"] != true || data["is_fresh_install"] != true {
		t.Fatalf("fresh 标记断言失败: %v", data)
	}
}

// TestMigrationSummaryEndpointNoState 无完成标记 → migrated=false（前端不展示）
func TestMigrationSummaryEndpointNoState(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	status, envelope := doJSON(t, r, http.MethodGet, "/api/v1/migration-summary", "", "")
	if status != http.StatusOK {
		t.Fatalf("摘要查询期望 200, 实际 %d, body: %v", status, envelope)
	}
	data, _ := envelope["data"].(map[string]interface{})
	if data["migrated"] != false {
		t.Fatalf("无标记应 migrated=false: %v", data)
	}
}
