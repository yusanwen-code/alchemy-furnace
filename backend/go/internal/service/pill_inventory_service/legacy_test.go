// 任务 5 测试：旧实体映射解析（plan 任务 5：旧 GET pill 跳转 + 旧 pill ID 导出共用）
// 覆盖：旧定义→丹方、旧绑定→能力、无映射 404、未知 kind 400
package pill_inventory_service

import (
	"context"
	"testing"

	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

// TestResolveLegacyPillToRecipe 旧定义映射 → 丹方 UUID
func TestResolveLegacyPillToRecipe(t *testing.T) {
	svc, db := newTestSvc(t)
	recipeUUID := uuid.New()
	if err := db.Create(&model.PillLegacyMap{
		LegacyKind: "pill", LegacyID: "old-def-1", TargetUUID: recipeUUID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	target, err := svc.ResolveLegacy(context.Background(), "pill", "old-def-1")
	if err != nil {
		t.Fatalf("ResolveLegacy: %v", err)
	}
	if target != recipeUUID {
		t.Fatalf("target=%s, want %s", target, recipeUUID)
	}
}

// TestResolveLegacyBindToEffect 旧绑定映射 → 能力 UUID
func TestResolveLegacyBindToEffect(t *testing.T) {
	svc, db := newTestSvc(t)
	effectUUID := uuid.New()
	if err := db.Create(&model.PillLegacyMap{
		LegacyKind: "bind", LegacyID: "old-bind-9", TargetUUID: effectUUID,
	}).Error; err != nil {
		t.Fatal(err)
	}

	target, err := svc.ResolveLegacy(context.Background(), "bind", "old-bind-9")
	if err != nil {
		t.Fatalf("ResolveLegacy: %v", err)
	}
	if target != effectUUID {
		t.Fatalf("target=%s, want %s", target, effectUUID)
	}
}

// TestResolveLegacyUnknownNotFound 无映射旧 ID → 404 pill.legacy_not_found
func TestResolveLegacyUnknownNotFound(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.ResolveLegacy(context.Background(), "pill", "never-mapped")
	if err == nil {
		t.Fatal("无映射旧 ID 应 404")
	}
	if !err.IsType(errors.ErrorTypeRecordNotFound) {
		t.Fatalf("应返回 RecordNotFound, 实际 %v", err)
	}
	if err.GetCode() != "pill.legacy_not_found" {
		t.Fatalf("code=%s, want pill.legacy_not_found", err.GetCode())
	}
}

// TestResolveLegacyInvalidKindRejected 未知旧实体类型 → 400
func TestResolveLegacyInvalidKindRejected(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.ResolveLegacy(context.Background(), "recipe", "whatever")
	if err == nil {
		t.Fatal("未知 kind 应 400")
	}
	if !err.IsType(errors.ErrorTypeInvalidRequest) {
		t.Fatalf("应返回 InvalidRequest, 实际 %v", err)
	}
	if err.GetCode() != "pill.invalid_legacy_kind" {
		t.Fatalf("code=%s, want pill.invalid_legacy_kind", err.GetCode())
	}
}
