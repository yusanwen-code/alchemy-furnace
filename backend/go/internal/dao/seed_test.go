package dao

import (
	"testing"

	"github.com/alchemy-furnace/server/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSeedBuiltinPillsIsIdempotentAndPreservesExistingData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ElixirPill{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := SeedBuiltinPills(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := db.Model(&model.ElixirPill{}).
		Where("name = ?", "文言文金丹").
		Update("description", "用户保留的说明").Error; err != nil {
		t.Fatalf("customize builtin: %v", err)
	}
	if err := SeedBuiltinPills(db); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	var count int64
	if err := db.Model(&model.ElixirPill{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 5 {
		t.Fatalf("pill count = %d, want 5", count)
	}

	var pill model.ElixirPill
	if err := db.Where("name = ?", "文言文金丹").First(&pill).Error; err != nil {
		t.Fatalf("load customized pill: %v", err)
	}
	if pill.Description != "用户保留的说明" {
		t.Fatalf("description was overwritten: %q", pill.Description)
	}
}

// TestSeedBuiltinRecipesAndGrantFreshInstall 全新安装：
// 5 个内置丹方 + 每丹方赠送 1 枚可用金丹（granted）；重复执行不重复产出（重启不自动补货）
func TestSeedBuiltinRecipesAndGrantFreshInstall(t *testing.T) {
	db := openInventoryTestDB(t, pillInventoryModels()...)
	if err := MigratePillInventory(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := SeedBuiltinRecipes(db); err != nil {
		t.Fatalf("seed recipes: %v", err)
	}
	if err := GrantStarterPills(db); err != nil {
		t.Fatalf("grant: %v", err)
	}

	assertTableCount(t, db, "pill_recipes", 5)
	assertTableCount(t, db, "pill_recipe_revisions", 5)
	assertTableCount(t, db, "pill_items", 5)
	var available int64
	if err := db.Table("pill_items").Where("state = ?", "available").Count(&available).Error; err != nil {
		t.Fatal(err)
	}
	if available != 5 {
		t.Fatalf("available=%d, want 5", available)
	}
	var granted int64
	if err := db.Model(&model.PillStarterGrant{}).Where("disposition = ?", "granted").Count(&granted).Error; err != nil {
		t.Fatal(err)
	}
	if granted != 5 {
		t.Fatalf("granted=%d, want 5", granted)
	}
	// 每丹方恰好 1 枚赠送实例
	var withItem int64
	if err := db.Model(&model.PillStarterGrant{}).Where("item_id IS NOT NULL").Count(&withItem).Error; err != nil {
		t.Fatal(err)
	}
	if withItem != 5 {
		t.Fatalf("带实例的赠送记录=%d, want 5", withItem)
	}

	// 第二次执行：不新增任何数据（防重启自动补货）
	if err := SeedBuiltinRecipes(db); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if err := GrantStarterPills(db); err != nil {
		t.Fatalf("second grant: %v", err)
	}
	for _, table := range []string{"pill_recipes", "pill_recipe_revisions", "pill_items", "pill_starter_grants"} {
		assertTableCount(t, db, table, 5)
	}
}

// TestSeedGrantStarterPillsLegacyAccounted 迁移用户：
// 内置丹方来自旧数据迁移，只写 legacy_accounted 标记，不赠送任何库存
func TestSeedGrantStarterPillsLegacyAccounted(t *testing.T) {
	db := openInventoryTestDB(t, append(legacyInventoryModels(), pillInventoryModels()...)...)
	// 旧库：1 个内置金丹（无绑定）→ 迁移后 1 丹方 + 1 可用实例
	builtin := model.ElixirPill{
		Name: "文言文金丹", Description: "d",
		SkillSchema: model.JSONMap{"identity_card": "X"},
		IsBuiltin:   true, Version: "1.0.0",
	}
	if err := db.Create(&builtin).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigratePillInventory(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// 确保丹方存在：迁移产物的同名丹方命中跳过，其余 4 个内置定义补齐
	if err := SeedBuiltinRecipes(db); err != nil {
		t.Fatalf("seed recipes: %v", err)
	}
	assertTableCount(t, db, "pill_recipes", 5)
	assertTableCount(t, db, "pill_recipe_revisions", 5)
	// 迁移产生的 1 枚可用实例仍在，赠送不得新增库存
	assertTableCount(t, db, "pill_items", 1)

	if err := GrantStarterPills(db); err != nil {
		t.Fatalf("grant: %v", err)
	}
	var accounted int64
	if err := db.Model(&model.PillStarterGrant{}).Where("disposition = ?", "legacy_accounted").Count(&accounted).Error; err != nil {
		t.Fatal(err)
	}
	if accounted != 5 {
		t.Fatalf("legacy_accounted=%d, want 5", accounted)
	}
	var granted int64
	if err := db.Model(&model.PillStarterGrant{}).Where("disposition = ?", "granted").Count(&granted).Error; err != nil {
		t.Fatal(err)
	}
	if granted != 0 {
		t.Fatalf("迁移用户不应有 granted 记录, got %d", granted)
	}
	assertTableCount(t, db, "pill_items", 1)
}

// assertTableCount 表行数断言
func assertTableCount(t *testing.T, db *gorm.DB, table string, want int64) {
	t.Helper()
	var got int64
	if err := db.Table(table).Count(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s: got %d want %d", table, got, want)
	}
}
