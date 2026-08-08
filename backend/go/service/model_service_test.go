// Package service 模型管理业务逻辑单元测试
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
	alchemycrypto "github.com/alchemy-furnace/server/pkg/crypto"

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

	if err := db.AutoMigrate(&model.LLMModel{}, &model.DaoAgent{}); err != nil {
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

// TestModelServiceCreateEncryptsKey 创建时 api_key 加密存储，响应仅含掩码
func TestModelServiceCreateEncryptsKey(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")
	svc := NewModelService()

	resp, err := svc.Create(&model.CreateLLMModelRequest{
		Name:        "deepseek-chat",
		DisplayName: "DeepSeek-V3",
		Provider:    "deepseek",
		BaseURL:     "https://api.deepseek.com/v1",
		APIKey:      "sk-testkey1234wxyz",
	})
	if err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}

	// 数据库中存储的必须是密文
	var stored model.LLMModel
	if err := db.First(&stored, resp.ID).Error; err != nil {
		t.Fatalf("查询模型失败: %v", err)
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
}

// TestModelServiceCreateDuplicateName 名称唯一冲突应返回校验错误
func TestModelServiceCreateDuplicateName(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	svc := NewModelService()

	req := &model.CreateLLMModelRequest{
		Name:        "gpt-4o",
		DisplayName: "GPT-4o",
		Provider:    "openai",
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

// TestModelServiceSecretRequired 未配置 MODEL_KEY_SECRET 且 api_key 非空时写操作应报错
func TestModelServiceSecretRequired(t *testing.T) {
	setupModelTestDB(t, "") // 空密钥
	svc := NewModelService()

	_, err := svc.Create(&model.CreateLLMModelRequest{
		Name:        "gpt-4o",
		DisplayName: "GPT-4o",
		Provider:    "openai",
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

	// api_key 为空（本地无鉴权服务）时允许创建
	if _, err := svc.Create(&model.CreateLLMModelRequest{
		Name:        "llama3",
		DisplayName: "Llama 3",
		Provider:    "ollama",
		BaseURL:     "http://localhost:11434/v1",
	}); err != nil {
		t.Errorf("无 api_key 的本地模型应允许创建: %v", err)
	}
}

// TestModelServiceDeleteProtection 被道人引用的模型拒绝删除并返回引用数量
func TestModelServiceDeleteProtection(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")
	svc := NewModelService()

	resp, err := svc.Create(&model.CreateLLMModelRequest{
		Name:        "gpt-4o",
		DisplayName: "GPT-4o",
		Provider:    "openai",
		BaseURL:     "https://api.openai.com/v1",
	})
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

	// 列表接口的 referenced_by 计数
	list, _, err := svc.List(1, 50, nil)
	if err != nil {
		t.Fatalf("查询模型列表失败: %v", err)
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
	svc := NewModelService()

	a, err := svc.Create(&model.CreateLLMModelRequest{
		Name: "gpt-4o", DisplayName: "GPT-4o", Provider: "openai",
		BaseURL: "https://api.openai.com/v1", IsDefault: true,
	})
	if err != nil {
		t.Fatalf("创建模型 A 失败: %v", err)
	}
	b, err := svc.Create(&model.CreateLLMModelRequest{
		Name: "deepseek-chat", DisplayName: "DeepSeek", Provider: "deepseek",
		BaseURL: "https://api.deepseek.com/v1",
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

	// 默认凭证解析应命中 B
	creds, err := svc.ResolveDefaultCredentials()
	if err != nil {
		t.Fatalf("解析默认凭证失败: %v", err)
	}
	if creds.Model != "deepseek-chat" || creds.BaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("默认凭证解析不符: %+v", creds)
	}
}

// TestModelServiceResolveDisabled 停用模型解析应返回明确中文错误
func TestModelServiceResolveDisabled(t *testing.T) {
	setupModelTestDB(t, "unit-test-secret")
	svc := NewModelService()

	resp, err := svc.Create(&model.CreateLLMModelRequest{
		Name: "gpt-4o", DisplayName: "GPT-4o", Provider: "openai",
		BaseURL: "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}
	fls := false
	if _, err := svc.Update(resp.ID, &model.UpdateLLMModelRequest{IsEnabled: &fls}); err != nil {
		t.Fatalf("停用模型失败: %v", err)
	}

	_, err = svc.ResolveCredentials("gpt-4o")
	if err == nil || !strings.Contains(err.Error(), "已停用") {
		t.Errorf("停用模型解析应返回停用错误: %v", err)
	}
}

// TestModelServiceUpdateAPIKeySemantics api_key 更新语义：nil=不变，""=清除，值=重新加密
func TestModelServiceUpdateAPIKeySemantics(t *testing.T) {
	db := setupModelTestDB(t, "unit-test-secret")
	svc := NewModelService()

	resp, err := svc.Create(&model.CreateLLMModelRequest{
		Name: "gpt-4o", DisplayName: "GPT-4o", Provider: "openai",
		BaseURL: "https://api.openai.com/v1", APIKey: "sk-original12345abcd",
	})
	if err != nil {
		t.Fatalf("创建模型失败: %v", err)
	}

	// nil = 不修改
	newName := "gpt-4o-new"
	if _, err := svc.Update(resp.ID, &model.UpdateLLMModelRequest{Name: &newName}); err != nil {
		t.Fatalf("更新名称失败: %v", err)
	}
	var stored model.LLMModel
	db.First(&stored, resp.ID)
	plain, _ := alchemycrypto.Decrypt(stored.APIKeyEncrypted, "unit-test-secret")
	if plain != "sk-original12345abcd" {
		t.Errorf("api_key 应保持不变: %q", plain)
	}

	// 值 = 重新加密
	newKey := "sk-replaced9999zzzz"
	if _, err := svc.Update(resp.ID, &model.UpdateLLMModelRequest{APIKey: &newKey}); err != nil {
		t.Fatalf("更换 api_key 失败: %v", err)
	}
	db.First(&stored, resp.ID)
	plain, _ = alchemycrypto.Decrypt(stored.APIKeyEncrypted, "unit-test-secret")
	if plain != newKey {
		t.Errorf("api_key 应已更换: %q", plain)
	}

	// "" = 清除密钥
	empty := ""
	updated, err := svc.Update(resp.ID, &model.UpdateLLMModelRequest{APIKey: &empty})
	if err != nil {
		t.Fatalf("清除 api_key 失败: %v", err)
	}
	if updated.HasAPIKey {
		t.Error("清除后 has_api_key 应为 false")
	}
}
