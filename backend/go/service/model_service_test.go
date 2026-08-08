// Package service 模型管理业务逻辑单元测试（003 两级结构：供应商 + 模型）
// 使用 sqlite 内存库（glebarez/sqlite，纯 Go 驱动），无需外部基础设施
package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupModelTestDB 初始化 sqlite 内存库并注入全局 dao.DB 与加密密钥配置
func setupModelTestDB(t *testing.T, secret string) *gorm.DB {
	t.Helper()

	// 每个测试独立的命名内存库（cache=shared 保证多连接可见同一份数据）
	dsn := fmt.Sprintf("file:modeltest%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("打开 sqlite 内存库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&model.LLMProvider{}, &model.LLMModel{}, &model.DaoAgent{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}

	dao.DB = db
	t.Cleanup(func() { dao.DB = nil })

	t.Setenv("AF_MODEL_KEY_SECRET", secret)
	if _, err := config.Load(); err != nil {
		t.Fatalf("加载测试配置失败: %v", err)
	}
	if got := config.Get().ModelKeySecret; got != secret {
		t.Fatalf("加密密钥配置未生效: got %q", got)
	}
	return db
}

// createTestProvider 测试辅助：创建一个已启用供应商（可选加密 api_key）
func createTestProvider(t *testing.T, name, baseURL, apiKey string) *model.ProviderResponse {
	t.Helper()
	svc := NewProviderService()
	resp, err := svc.Create(&model.CreateProviderRequest{
		Name:        name,
		DisplayName: name,
		BaseURL:     baseURL,
		APIKey:      apiKey,
	})
	if err != nil {
		t.Fatalf("创建测试供应商失败: %v", err)
	}
	return resp
}

// TestMaskAPIKey 掩码规则：长度 > 7 显示前 3 位 + **** + 末 4 位，否则 ****
func TestMaskAPIKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"sk-abcdefghwxyz", "sk-****wxyz"},
		{"sk-testkey1234wxyz", "sk-****wxyz"},
		{"short", "****"},
		{"1234567", "****"},
		{"", ""},
	}
	for _, c := range cases {
		if got := MaskAPIKey(c.in); got != c.want {
			t.Errorf("MaskAPIKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestModelServiceNestedCreate 嵌套创建：模型归属供应商，响应携带供应商引用信息
func TestModelServiceNestedCreate(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	p := createTestProvider(t, "deepseek", "https://api.deepseek.com/v1", "sk-testkey1234wxyz")
	svc := NewModelService()

	resp, err := svc.Create(p.ID, &model.CreateLLMModelRequest{
		Name:        "deepseek-chat",
		DisplayName: "DeepSeek-V3",
	})
	if err != nil {
		t.Fatalf("嵌套创建模型失败: %v", err)
	}
	if resp.ProviderID != p.ID {
		t.Errorf("provider_id 不符: got %d, want %d", resp.ProviderID, p.ID)
	}
	if resp.ProviderName != "deepseek" {
		t.Errorf("provider_name 不符: %q", resp.ProviderName)
	}

	// 供应商不存在时拒绝创建
	_, err = svc.Create(9999, &model.CreateLLMModelRequest{Name: "x", DisplayName: "x"})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("不存在的供应商应返回 ValidationError: %v", err)
	}
}

// TestModelServiceCreateDisabledProvider 供应商已停用时拒绝创建模型
func TestModelServiceCreateDisabledProvider(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	p := createTestProvider(t, "openai", "https://api.openai.com/v1", "")
	fls := false
	if _, err := NewProviderService().Update(p.ID, &model.UpdateProviderRequest{IsEnabled: &fls}); err != nil {
		t.Fatalf("停用供应商失败: %v", err)
	}

	_, err := NewModelService().Create(p.ID, &model.CreateLLMModelRequest{Name: "gpt-4o", DisplayName: "GPT-4o"})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || !strings.Contains(err.Error(), "已停用") {
		t.Errorf("停用供应商下创建模型应返回停用错误: %v", err)
	}
}

// TestModelServiceCreateDuplicateName 同供应商下名称冲突报错；不同供应商允许同名
func TestModelServiceCreateDuplicateName(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	pa := createTestProvider(t, "openai", "https://api.openai.com/v1", "")
	pb := createTestProvider(t, "azure-openai", "https://example.openai.azure.com/v1", "")
	svc := NewModelService()

	req := &model.CreateLLMModelRequest{Name: "gpt-4o", DisplayName: "GPT-4o"}
	if _, err := svc.Create(pa.ID, req); err != nil {
		t.Fatalf("首次创建失败: %v", err)
	}

	// 同供应商下重名 → 冲突
	_, err := svc.Create(pa.ID, req)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("同供应商重名创建应返回 ValidationError: %v", err)
	}
	if !strings.Contains(err.Error(), "已存在") {
		t.Errorf("错误信息应提示名称已存在: %v", err)
	}

	// 不同供应商允许同名
	if _, err := svc.Create(pb.ID, req); err != nil {
		t.Errorf("不同供应商下同名模型应允许创建: %v", err)
	}
}

// TestModelServiceDeleteProtection 被道人引用的模型拒绝删除并返回引用数量
func TestModelServiceDeleteProtection(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")
	p := createTestProvider(t, "openai", "https://api.openai.com/v1", "")
	svc := NewModelService()

	resp, err := svc.Create(p.ID, &model.CreateLLMModelRequest{Name: "gpt-4o", DisplayName: "GPT-4o"})
	if err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}

	// 两个道人引用该模型
	for _, name := range []string{"道人甲", "道人乙"} {
		if err := db.Create(&model.DaoAgent{Name: name, ModelName: "gpt-4o", Status: "active"}).Error; err != nil {
			t.Fatalf("创建道人失败: %v", err)
		}
	}

	err = svc.Delete(resp.ID)
	var refErr *ModelReferencedError
	if !errors.As(err, &refErr) {
		t.Fatalf("被引用的模型删除应返回 ModelReferencedError: %v", err)
	}
	if refErr.Count != 2 {
		t.Errorf("引用数量不符: got %d, want 2", refErr.Count)
	}
	if !strings.Contains(err.Error(), "2 个道人") {
		t.Errorf("错误信息应包含引用数量: %v", err)
	}

	// 供应商下模型列表的 referenced_by 计数
	list, err := svc.ListByProvider(p.ID)
	if err != nil {
		t.Fatalf("查询供应商下模型列表失败: %v", err)
	}
	if len(list) != 1 || list[0].ReferencedBy != 2 {
		t.Errorf("列表 referenced_by 不符: %+v", list)
	}

	// 移除引用后可正常删除
	db.Where("1 = 1").Delete(&model.DaoAgent{})
	if err := svc.Delete(resp.ID); err != nil {
		t.Errorf("无引用后删除应成功: %v", err)
	}
}

// TestModelServiceSetDefaultClearsOthers 设置默认模型时清除其他记录的默认标记
func TestModelServiceSetDefaultClearsOthers(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")
	p := createTestProvider(t, "openai", "https://api.openai.com/v1", "sk-default1234abcd")
	svc := NewModelService()

	a, err := svc.Create(p.ID, &model.CreateLLMModelRequest{
		Name: "gpt-4o", DisplayName: "GPT-4o", IsDefault: true,
	})
	if err != nil {
		t.Fatalf("创建模型 A 失败: %v", err)
	}
	b, err := svc.Create(p.ID, &model.CreateLLMModelRequest{
		Name: "gpt-4o-mini", DisplayName: "GPT-4o Mini",
	})
	if err != nil {
		t.Fatalf("创建模型 B 失败: %v", err)
	}

	// 将 B 设为默认
	tru := true
	if _, err := svc.Update(b.ID, &model.UpdateLLMModelRequest{IsDefault: &tru}); err != nil {
		t.Fatalf("设置 B 为默认失败: %v", err)
	}

	var ma, mb model.LLMModel
	db.First(&ma, a.ID)
	db.First(&mb, b.ID)
	if ma.IsDefault {
		t.Error("A 的默认标记应被清除")
	}
	if !mb.IsDefault {
		t.Error("B 应为默认模型")
	}

	// 默认凭证解析应命中 B，并继承供应商凭证
	creds, err := svc.ResolveDefaultCredentials()
	if err != nil {
		t.Fatalf("解析默认凭证失败: %v", err)
	}
	if creds.Model != "gpt-4o-mini" || creds.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("默认凭证解析不符: %+v", creds)
	}
	if creds.APIKey != "sk-default1234abcd" {
		t.Errorf("默认凭证应解密供应商 api_key: %q", creds.APIKey)
	}
}

// TestModelServiceSetSynthesisClearsOthers 设置合成专用模型时清除其他记录的合成标记
func TestModelServiceSetSynthesisClearsOthers(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")
	p := createTestProvider(t, "openai", "https://api.openai.com/v1", "")
	svc := NewModelService()

	a, err := svc.Create(p.ID, &model.CreateLLMModelRequest{
		Name: "gpt-4o-mini", DisplayName: "GPT-4o Mini", IsSynthesis: true,
	})
	if err != nil {
		t.Fatalf("创建模型 A 失败: %v", err)
	}
	b, err := svc.Create(p.ID, &model.CreateLLMModelRequest{
		Name: "gpt-4o", DisplayName: "GPT-4o",
	})
	if err != nil {
		t.Fatalf("创建模型 B 失败: %v", err)
	}

	tru := true
	if _, err := svc.Update(b.ID, &model.UpdateLLMModelRequest{IsSynthesis: &tru}); err != nil {
		t.Fatalf("设置 B 为合成专用失败: %v", err)
	}

	var ma, mb model.LLMModel
	db.First(&ma, a.ID)
	db.First(&mb, b.ID)
	if ma.IsSynthesis {
		t.Error("A 的合成标记应被清除")
	}
	if !mb.IsSynthesis {
		t.Error("B 应为合成专用模型")
	}

	creds, err := svc.ResolveSynthesisCredentials()
	if err != nil {
		t.Fatalf("解析合成凭证失败: %v", err)
	}
	if creds.Model != "gpt-4o" {
		t.Errorf("合成凭证解析不符: %+v", creds)
	}
}

// TestModelServiceResolveChain 凭证解析链：模型停用 / 供应商停用 / 正常继承供应商凭证
func TestModelServiceResolveChain(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	p := createTestProvider(t, "deepseek", "https://api.deepseek.com/v1", "sk-resolve9999zzzz")
	svc := NewModelService()

	resp, err := svc.Create(p.ID, &model.CreateLLMModelRequest{Name: "deepseek-chat", DisplayName: "DeepSeek-V3"})
	if err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}

	// 正常路径：继承供应商 base_url + 解密 api_key
	creds, err := svc.ResolveCredentials("deepseek-chat")
	if err != nil {
		t.Fatalf("解析凭证失败: %v", err)
	}
	if creds.Model != "deepseek-chat" || creds.BaseURL != "https://api.deepseek.com/v1" || creds.APIKey != "sk-resolve9999zzzz" {
		t.Errorf("凭证解析不符: %+v", creds)
	}

	// 模型停用 → 明确中文错误
	fls := false
	if _, err := svc.Update(resp.ID, &model.UpdateLLMModelRequest{IsEnabled: &fls}); err != nil {
		t.Fatalf("停用模型失败: %v", err)
	}
	_, err = svc.ResolveCredentials("deepseek-chat")
	if err == nil || !strings.Contains(err.Error(), "该道人使用的模型已停用") {
		t.Errorf("模型停用应返回停用错误: %v", err)
	}

	// 恢复模型、停用供应商 → 供应商停用错误
	tru := true
	if _, err := svc.Update(resp.ID, &model.UpdateLLMModelRequest{IsEnabled: &tru}); err != nil {
		t.Fatalf("启用模型失败: %v", err)
	}
	if _, err := NewProviderService().Update(p.ID, &model.UpdateProviderRequest{IsEnabled: &fls}); err != nil {
		t.Fatalf("停用供应商失败: %v", err)
	}
	_, err = svc.ResolveCredentials("deepseek-chat")
	if err == nil || !strings.Contains(err.Error(), "该模型所属供应商已停用") {
		t.Errorf("供应商停用应返回停用错误: %v", err)
	}

	// 完全未登记的模型 → 回退空凭证（仅含模型名），不报错
	creds, err = svc.ResolveCredentials("never-registered-model")
	if err != nil {
		t.Fatalf("未登记模型应回退环境变量凭证: %v", err)
	}
	if creds.Model != "never-registered-model" || creds.BaseURL != "" || creds.APIKey != "" {
		t.Errorf("未登记模型应返回空凭证: %+v", creds)
	}
}

// TestModelServiceResolveDuplicateNames 同名模型跨供应商时取 sort_order,id 最小者
func TestModelServiceResolveDuplicateNames(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	pa := createTestProvider(t, "openai", "https://api.openai.com/v1", "")
	pb := createTestProvider(t, "azure-openai", "https://example.openai.azure.com/v1", "")
	svc := NewModelService()

	// 后创建的供应商（id 更大）先登记同名模型
	if _, err := svc.Create(pb.ID, &model.CreateLLMModelRequest{Name: "gpt-4o", DisplayName: "Azure GPT-4o"}); err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}
	if _, err := svc.Create(pa.ID, &model.CreateLLMModelRequest{Name: "gpt-4o", DisplayName: "GPT-4o"}); err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}

	creds, err := svc.ResolveCredentials("gpt-4o")
	if err != nil {
		t.Fatalf("解析凭证失败: %v", err)
	}
	// 同 sort_order 下取 id 最小者（先创建的在 pb 下）
	if creds.BaseURL != "https://example.openai.azure.com/v1" {
		t.Errorf("同名模型应取 sort_order,id 最小者的供应商: %+v", creds)
	}
}

// TestModelServiceOptions 选项列表：仅含启用供应商下的启用模型，带供应商显示名
func TestModelServiceOptions(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	enabledProvider := createTestProvider(t, "deepseek", "https://api.deepseek.com/v1", "")
	disabledProvider := createTestProvider(t, "openai", "https://api.openai.com/v1", "")
	svc := NewModelService()

	if _, err := svc.Create(enabledProvider.ID, &model.CreateLLMModelRequest{
		Name: "deepseek-chat", DisplayName: "DeepSeek-V3", IsDefault: true,
	}); err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}

	// 停用另一个供应商（停用前先建模型）
	if _, err := svc.Create(disabledProvider.ID, &model.CreateLLMModelRequest{
		Name: "gpt-4o", DisplayName: "GPT-4o",
	}); err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}
	fls := false
	if _, err := NewProviderService().Update(disabledProvider.ID, &model.UpdateProviderRequest{IsEnabled: &fls}); err != nil {
		t.Fatalf("停用供应商失败: %v", err)
	}

	options, err := svc.Options()
	if err != nil {
		t.Fatalf("查询模型选项失败: %v", err)
	}
	if len(options) != 1 {
		t.Fatalf("停用供应商下的模型不应出现在选项中: %+v", options)
	}
	opt := options[0]
	if opt.Name != "deepseek-chat" || opt.ProviderName != "deepseek" || opt.ProviderDisplayName != "deepseek" || !opt.IsDefault {
		t.Errorf("选项字段不符: %+v", opt)
	}
}
