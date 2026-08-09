// Package service 供应商管理业务逻辑单元测试
// 使用 sqlite 内存库（glebarez/sqlite，纯 Go 驱动），无需外部基础设施
package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/model"
	alchemycrypto "github.com/alchemy-furnace/server/internal/util/crypto"
)

// TestProviderCreateEncryptsKey 创建时 api_key 加密存储，响应仅含掩码
func TestProviderCreateEncryptsKey(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")
	svc := NewProviderService()

	resp, err := svc.Create(&model.CreateProviderRequest{
		Name:        "deepseek",
		DisplayName: "DeepSeek",
		BaseURL:     "https://api.deepseek.com/v1",
		APIKey:      "sk-testkey1234wxyz",
	})
	if err != nil {
		t.Fatalf("创建供应商失败: %v", err)
	}

	// 数据库中存储的必须是密文
	var stored model.LLMProvider
	if err := db.First(&stored, resp.ID).Error; err != nil {
		t.Fatalf("查询供应商失败: %v", err)
	}
	if stored.APIKeyEncrypted == "" || stored.APIKeyEncrypted == "sk-testkey1234wxyz" {
		t.Fatalf("api_key 未加密存储: %q", stored.APIKeyEncrypted)
	}
	if strings.Contains(stored.APIKeyEncrypted, "testkey") {
		t.Fatalf("密文不应包含明文片段: %q", stored.APIKeyEncrypted)
	}

	// 密文可解密还原
	plain, err := alchemycrypto.Decrypt(stored.APIKeyEncrypted, "unit-test-secret")
	if err != nil || plain != "sk-testkey1234wxyz" {
		t.Fatalf("解密还原失败: plain=%q err=%v", plain, err)
	}

	// 响应仅含掩码，不含明文
	if !resp.HasAPIKey {
		t.Error("has_api_key 应为 true")
	}
	if resp.APIKeyMasked != "sk-****wxyz" {
		t.Errorf("api_key_masked 不符: %q", resp.APIKeyMasked)
	}
	out := fmt.Sprintf("%+v", resp)
	if strings.Contains(out, "testkey1234") {
		t.Errorf("响应不应包含明文 api_key: %s", out)
	}

	// protocol 缺省应为 openai-compatible
	if resp.Protocol != "openai-compatible" {
		t.Errorf("protocol 缺省值不符: %q", resp.Protocol)
	}
}

// TestProviderCreateDuplicateName 供应商标识唯一冲突应返回校验错误
func TestProviderCreateDuplicateName(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	svc := NewProviderService()

	req := &model.CreateProviderRequest{
		Name:        "openai",
		DisplayName: "OpenAI",
		BaseURL:     "https://api.openai.com/v1",
	}
	if _, err := svc.Create(req); err != nil {
		t.Fatalf("首次创建失败: %v", err)
	}

	_, err := svc.Create(req)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("重名创建应返回 ValidationError: %v", err)
	}
	if !strings.Contains(err.Error(), "已存在") {
		t.Errorf("错误信息应提示名称已存在: %v", err)
	}
}

// TestProviderCreateValidation base_url 非法 / protocol 不支持应返回校验错误
func TestProviderCreateValidation(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	svc := NewProviderService()

	_, err := svc.Create(&model.CreateProviderRequest{
		Name: "bad-url", DisplayName: "Bad", BaseURL: "ftp://example.com",
	})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("非法 base_url 应返回 ValidationError: %v", err)
	}

	_, err = svc.Create(&model.CreateProviderRequest{
		Name: "bad-proto", DisplayName: "Bad", BaseURL: "https://api.example.com/v1", Protocol: "anthropic",
	})
	if !errors.As(err, &validationErr) || !strings.Contains(err.Error(), "openai-compatible") {
		t.Errorf("不支持的协议应返回 ValidationError: %v", err)
	}
}

// TestProviderSecretRequired 未配置 MODEL_KEY_SECRET 且 api_key 非空时写操作应报错
func TestProviderSecretRequired(t *testing.T) {
	setupModelTestDB(t, "") // 空密钥
	svc := NewProviderService()

	_, err := svc.Create(&model.CreateProviderRequest{
		Name:        "openai",
		DisplayName: "OpenAI",
		BaseURL:     "https://api.openai.com/v1",
		APIKey:      "sk-anything",
	})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("空密钥 + 非空 api_key 应返回 ValidationError: %v", err)
	}
	if !strings.Contains(err.Error(), "MODEL_KEY_SECRET") {
		t.Errorf("错误信息应提及 MODEL_KEY_SECRET: %v", err)
	}
}

// TestProviderEmptyKeyAllowed 免密钥本地服务（Ollama 风格）允许空 api_key
func TestProviderEmptyKeyAllowed(t *testing.T) {
	setupModelTestDB(t, "") // 即使未配置加密密钥也应允许
	svc := NewProviderService()

	resp, err := svc.Create(&model.CreateProviderRequest{
		Name:        "ollama",
		DisplayName: "Ollama（本地）",
		BaseURL:     "http://localhost:11434/v1",
	})
	if err != nil {
		t.Fatalf("无 api_key 的本地供应商应允许创建: %v", err)
	}
	if resp.HasAPIKey {
		t.Error("has_api_key 应为 false")
	}
	if resp.APIKeyMasked != "" {
		t.Errorf("api_key_masked 应为空: %q", resp.APIKeyMasked)
	}
}

// TestProviderDeleteProtection 下有关联模型的供应商拒绝删除并返回模型数量
func TestProviderDeleteProtection(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	providerSvc := NewProviderService()
	modelSvc := NewModelService()

	p := createTestProvider(t, "openai", "https://api.openai.com/v1", "")
	for _, name := range []string{"gpt-4o", "gpt-4o-mini"} {
		if _, err := modelSvc.Create(p.ID, &model.CreateLLMModelRequest{Name: name, DisplayName: name}); err != nil {
			t.Fatalf("创建模型失败: %v", err)
		}
	}

	// 列表接口的 model_count 统计
	list, total, err := providerSvc.List(1, 50, nil)
	if err != nil {
		t.Fatalf("查询供应商列表失败: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ModelCount != 2 {
		t.Errorf("列表 model_count 不符: total=%d list=%+v", total, list)
	}

	err = providerSvc.Delete(p.ID)
	var hasModelsErr *ProviderHasModelsError
	if !errors.As(err, &hasModelsErr) {
		t.Fatalf("下有模型的供应商删除应返回 ProviderHasModelsError: %v", err)
	}
	if hasModelsErr.Count != 2 {
		t.Errorf("模型数量不符: got %d, want 2", hasModelsErr.Count)
	}
	if !strings.Contains(err.Error(), "2 个模型") {
		t.Errorf("错误信息应包含模型数量: %v", err)
	}

	// 删除其下模型后可正常删除
	models, _ := modelSvc.ListByProvider(p.ID)
	for _, m := range models {
		if err := modelSvc.Delete(m.ID); err != nil {
			t.Fatalf("删除模型失败: %v", err)
		}
	}
	if err := providerSvc.Delete(p.ID); err != nil {
		t.Errorf("无模型后删除应成功: %v", err)
	}
}

// TestProviderUpdateAPIKeySemantics api_key 更新三态语义：nil=不变，""=清除，值=重新加密
func TestProviderUpdateAPIKeySemantics(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")
	svc := NewProviderService()

	resp, err := svc.Create(&model.CreateProviderRequest{
		Name:        "openai",
		DisplayName: "OpenAI",
		BaseURL:     "https://api.openai.com/v1",
		APIKey:      "sk-original12345abcd",
	})
	if err != nil {
		t.Fatalf("创建供应商失败: %v", err)
	}

	// nil = 不修改
	newName := "openai-new"
	if _, err := svc.Update(resp.ID, &model.UpdateProviderRequest{Name: &newName}); err != nil {
		t.Fatalf("更新名称失败: %v", err)
	}
	var stored model.LLMProvider
	db.First(&stored, resp.ID)
	plain, _ := alchemycrypto.Decrypt(stored.APIKeyEncrypted, "unit-test-secret")
	if plain != "sk-original12345abcd" {
		t.Errorf("api_key 应保持不变: %q", plain)
	}

	// 值 = 重新加密
	newKey := "sk-replaced9999zzzz"
	if _, err := svc.Update(resp.ID, &model.UpdateProviderRequest{APIKey: &newKey}); err != nil {
		t.Fatalf("更换 api_key 失败: %v", err)
	}
	db.First(&stored, resp.ID)
	plain, _ = alchemycrypto.Decrypt(stored.APIKeyEncrypted, "unit-test-secret")
	if plain != newKey {
		t.Errorf("api_key 应已更换: %q", plain)
	}

	// "" = 清除密钥
	empty := ""
	updated, err := svc.Update(resp.ID, &model.UpdateProviderRequest{APIKey: &empty})
	if err != nil {
		t.Fatalf("清除 api_key 失败: %v", err)
	}
	if updated.HasAPIKey {
		t.Error("清除后 has_api_key 应为 false")
	}
	db.First(&stored, resp.ID)
	if stored.APIKeyEncrypted != "" {
		t.Errorf("清除后密文应为空: %q", stored.APIKeyEncrypted)
	}
}

// TestProviderTemplates 预置模板清单：8 项，ollama 免密钥且分组为 local
func TestProviderTemplates(t *testing.T) {
	svc := NewProviderService()
	templates := svc.Templates()
	if len(templates) != 8 {
		t.Fatalf("模板数量不符: got %d, want 8", len(templates))
	}
	seen := make(map[string]model.ProviderTemplate, len(templates))
	for _, tpl := range templates {
		seen[tpl.ID] = tpl
		if tpl.Protocol != "openai-compatible" {
			t.Errorf("模板 %s 协议不符: %q", tpl.ID, tpl.Protocol)
		}
	}
	ollama, ok := seen["ollama"]
	if !ok {
		t.Fatal("模板清单缺少 ollama")
	}
	if ollama.RequiresAPIKey || ollama.Group != "local" {
		t.Errorf("ollama 模板不符: %+v", ollama)
	}
	if seen["openai"].Group != "international" {
		t.Errorf("openai 模板分组不符: %+v", seen["openai"])
	}
	if seen["deepseek"].Group != "domestic" {
		t.Errorf("deepseek 模板分组不符: %+v", seen["deepseek"])
	}
}

// TestRebuildLegacyLLMModelsTable T018：旧版 llm_models（无 provider_id 列）应被检测并清空重建
// sqlite 下迁移逻辑完全可移植（HasTable/HasColumn/DropTable + 原生 CREATE INDEX IF NOT EXISTS）
func TestRebuildLegacyLLMModelsTable(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")

	// 构造 002 旧结构的 llm_models 表（凭证挂在模型上，无 provider_id）
	if err := db.Migrator().DropTable("llm_models"); err != nil {
		t.Fatalf("清理测试表失败: %v", err)
	}
	legacyDDL := `CREATE TABLE llm_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		display_name TEXT NOT NULL,
		provider TEXT NOT NULL,
		base_url TEXT NOT NULL,
		api_key_encrypted TEXT,
		is_enabled INTEGER DEFAULT 1
	)`
	if err := db.Exec(legacyDDL).Error; err != nil {
		t.Fatalf("构造旧版 llm_models 表失败: %v", err)
	}
	if err := db.Exec(`INSERT INTO llm_models (name, display_name, provider, base_url) VALUES ('gpt-4o', 'GPT-4o', 'openai', 'https://api.openai.com/v1')`).Error; err != nil {
		t.Fatalf("写入旧版数据失败: %v", err)
	}

	// 执行迁移：应检测旧结构并 DROP 重建
	if err := dao.AutoMigrate(db); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}

	// 新结构应包含 provider_id 列
	if !db.Migrator().HasColumn(&model.LLMModel{}, "provider_id") {
		t.Error("重建后的 llm_models 应包含 provider_id 列")
	}
	// 旧数据已清空
	var count int64
	if err := db.Model(&model.LLMModel{}).Count(&count).Error; err != nil {
		t.Fatalf("查询模型数量失败: %v", err)
	}
	if count != 0 {
		t.Errorf("重建后的 llm_models 应为空: got %d", count)
	}
	// llm_providers 表应已创建
	if !db.Migrator().HasTable("llm_providers") {
		t.Error("迁移后应存在 llm_providers 表")
	}

	// 索引应存在（联合唯一 + 部分唯一索引）
	var indexes []string
	if err := db.Raw(`SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'llm_models'`).Scan(&indexes).Error; err != nil {
		t.Fatalf("查询索引失败: %v", err)
	}
	indexSet := make(map[string]bool, len(indexes))
	for _, idx := range indexes {
		indexSet[idx] = true
	}
	for _, want := range []string{"idx_llm_models_provider_name", "idx_llm_models_default", "idx_llm_models_synthesis"} {
		if !indexSet[want] {
			t.Errorf("缺少索引 %s，现有索引: %v", want, indexes)
		}
	}

	// 幂等：再次执行迁移不应报错也不应再 DROP
	m := model.LLMModel{ProviderID: 1, Name: "keep", DisplayName: "Keep", IsEnabled: true}
	if err := dao.GetDB().Exec(`INSERT INTO llm_providers (name, display_name, protocol, base_url, is_enabled) VALUES ('p1', 'P1', 'openai-compatible', 'https://example.com/v1', 1)`).Error; err != nil {
		t.Fatalf("写入供应商失败: %v", err)
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("写入模型失败: %v", err)
	}
	if err := dao.AutoMigrate(db); err != nil {
		t.Fatalf("二次迁移失败: %v", err)
	}
	if err := db.Model(&model.LLMModel{}).Count(&count).Error; err != nil || count != 1 {
		t.Errorf("二次迁移不应清空新结构数据: count=%d err=%v", count, err)
	}
}
