// 任务 5 测试：旧入口封堵（plan 任务 5 行 430-431）
// 覆盖：旧 GET /pills/:uuid 仅提供 LegacyMap 跳转 {entity_type,recipe_id}，
// 不假装该 ID 是可用金丹；无映射 → 404。
package pill_inventory

import (
	"net/http"
	"testing"

	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

// TestLegacyPillDetailRedirect 旧金丹详情 → LegacyMap 跳转信息（200）
func TestLegacyPillDetailRedirect(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter()

	recipeID := uuid.New()
	legacyID := uuid.NewString()
	if err := db.Create(&model.PillLegacyMap{
		LegacyKind: "pill", LegacyID: legacyID, TargetUUID: recipeID,
	}).Error; err != nil {
		t.Fatalf("造映射失败: %v", err)
	}

	status, envelope := doJSON(t, r, http.MethodGet, "/api/v1/pills/"+legacyID, "", "")
	if status != http.StatusOK {
		t.Fatalf("跳转查询期望 200, 实际 %d, body: %v", status, envelope)
	}
	if entity, _ := envelope["data"].(map[string]interface{})["entity_type"].(string); entity != "recipe" {
		t.Fatalf("entity_type 应恒为 recipe, 实际 %v", envelope)
	}
	if rid, _ := envelope["data"].(map[string]interface{})["recipe_id"].(string); rid != recipeID.String() {
		t.Fatalf("recipe_id 不一致: %v", envelope)
	}
}

// TestLegacyPillDetailUnmapped 无映射旧 ID → 404 pill.legacy_not_found
func TestLegacyPillDetailUnmapped(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	status, envelope := doJSON(t, r, http.MethodGet, "/api/v1/pills/"+uuid.NewString(), "", "")
	if status != http.StatusNotFound {
		t.Fatalf("无映射期望 404, 实际 %d, body: %v", status, envelope)
	}
	if envelope["error_code"] != "pill.legacy_not_found" {
		t.Fatalf("期望 error_code=pill.legacy_not_found, 实际 %v", envelope)
	}
}

// TestLegacyPillDetailInvalidUUID 旧路径非 UUID → 400
func TestLegacyPillDetailInvalidUUID(t *testing.T) {
	setupTestDB(t)
	r := setupRouter()

	status, _ := doJSON(t, r, http.MethodGet, "/api/v1/pills/not-a-uuid", "", "")
	if status != http.StatusBadRequest {
		t.Fatalf("非 UUID 期望 400, 实际 %d", status)
	}
}
