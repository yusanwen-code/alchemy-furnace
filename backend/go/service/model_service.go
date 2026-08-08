// Package service 模型管理业务逻辑层
// 管理 LLM 模型配置（llm_models 表）：CRUD、凭证解析、连接测试
// api_key 以 AES-GCM 加密存储（密钥来自 MODEL_KEY_SECRET），接口输出仅掩码形式
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/config"
	alchemycrypto "github.com/alchemy-furnace/server/pkg/crypto"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ModelService 模型管理业务逻辑
type ModelService struct {
	engineBaseURL string
	httpClient    *http.Client
}

// NewModelService 创建模型管理业务实例
func NewModelService() *ModelService {
	return &ModelService{
		engineBaseURL: config.Get().PythonEngine.BaseURL,
		httpClient:    &http.Client{Timeout: 60 * time.Second},
	}
}

// ModelCredentials 解析后的模型调用凭证，按请求透传给 Python 语言引擎
// BaseURL/APIKey 为空时 Python 回退到自身环境变量配置（向后兼容）
type ModelCredentials struct {
	Model   string `json:"model"`
	BaseURL string `json:"base_url,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
}

// ValidationError 请求参数校验错误（处理器映射为 HTTP 400）
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

// ModelReferencedError 模型仍被道人引用，拒绝删除（处理器映射为 HTTP 409）
type ModelReferencedError struct {
	Count int64
}

func (e *ModelReferencedError) Error() string {
	return fmt.Sprintf("该模型仍被 %d 个道人引用，无法删除", e.Count)
}

// MaskAPIKey 生成 api_key 掩码：长度 > 7 时显示前 3 位 + **** + 末 4 位（如 sk-****wxyz），否则 ****
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	if len(apiKey) > 7 {
		return apiKey[:3] + "****" + apiKey[len(apiKey)-4:]
	}
	return "****"
}

// keySecret 读取当前配置的加密密钥（动态读取便于测试与运行时配置）
func (s *ModelService) keySecret() string {
	return config.Get().ModelKeySecret
}

// decryptAPIKey 解密模型的 api_key；未配置（空密文）返回空字符串
func (s *ModelService) decryptAPIKey(m *model.LLMModel) (string, error) {
	if m.APIKeyEncrypted == "" {
		return "", nil
	}
	plain, err := alchemycrypto.Decrypt(m.APIKeyEncrypted, s.keySecret())
	if err != nil {
		return "", fmt.Errorf("模型凭证解密失败，请检查 MODEL_KEY_SECRET 配置: %w", err)
	}
	return plain, nil
}

// toResponse 转换为响应 DTO（解密仅用于生成掩码，明文不出现在任何输出中）
func (s *ModelService) toResponse(m *model.LLMModel, referencedBy int64) *model.LLMModelResponse {
	masked := ""
	if m.APIKeyEncrypted != "" {
		plain, err := s.decryptAPIKey(m)
		if err != nil {
			zap.L().Warn("[炼丹炉] 模型密钥解密失败，掩码降级显示", zap.Uint("model_id", m.ID), zap.Error(err))
			masked = "****"
		} else {
			masked = MaskAPIKey(plain)
		}
	}
	return &model.LLMModelResponse{
		ID:           m.ID,
		Name:         m.Name,
		DisplayName:  m.DisplayName,
		Provider:     m.Provider,
		BaseURL:      m.BaseURL,
		APIKeyMasked: masked,
		HasAPIKey:    m.APIKeyEncrypted != "",
		Temperature:  m.Temperature,
		MaxTokens:    m.MaxTokens,
		IsEnabled:    m.IsEnabled,
		IsDefault:    m.IsDefault,
		IsSynthesis:  m.IsSynthesis,
		SortOrder:    m.SortOrder,
		ReferencedBy: referencedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// referencedCounts 统计每个模型名被 dao_agents.model_name 引用的数量
func (s *ModelService) referencedCounts() (map[string]int64, error) {
	type row struct {
		ModelName string
		Cnt       int64
	}
	var rows []row
	if err := dao.GetDB().Model(&model.DaoAgent{}).
		Select("model_name, COUNT(*) AS cnt").
		Group("model_name").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("统计模型引用数量失败: %w", err)
	}
	counts := make(map[string]int64, len(rows))
	for _, r := range rows {
		counts[r.ModelName] = r.Cnt
	}
	return counts, nil
}

// referencedCount 统计指定模型名被引用的数量
func (s *ModelService) referencedCount(name string) (int64, error) {
	var count int64
	if err := dao.GetDB().Model(&model.DaoAgent{}).
		Where("model_name = ?", name).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计模型引用数量失败: %w", err)
	}
	return count, nil
}

// List 模型列表，支持 enabled 过滤与分页，附带引用计数
func (s *ModelService) List(page, pageSize int, enabled *bool) ([]model.LLMModelResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	db := dao.GetDB().Model(&model.LLMModel{})
	if enabled != nil {
		db = db.Where("is_enabled = ?", *enabled)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模型总数失败: %w", err)
	}

	var models []model.LLMModel
	offset := (page - 1) * pageSize
	if err := db.Order("sort_order ASC, id ASC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模型列表失败: %w", err)
	}

	counts, err := s.referencedCounts()
	if err != nil {
		return nil, 0, err
	}

	list := make([]model.LLMModelResponse, 0, len(models))
	for i := range models {
		list = append(list, *s.toResponse(&models[i], counts[models[i].Name]))
	}
	return list, total, nil
}

// GetByID 根据 ID 获取模型详情（掩码形式）
func (s *ModelService) GetByID(id uint) (*model.LLMModelResponse, error) {
	var m model.LLMModel
	if err := dao.GetDB().First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("模型(id=%d)不存在: %w", id, err)
	}
	count, err := s.referencedCount(m.Name)
	if err != nil {
		return nil, err
	}
	return s.toResponse(&m, count), nil
}

// validateBaseURL 校验 base_url 为合法 http(s) URL
func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return &ValidationError{Msg: "base_url 不是合法的 URL"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return &ValidationError{Msg: "base_url 仅支持 http/https 协议"}
	}
	return nil
}

// validateName 校验模型名长度
func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return &ValidationError{Msg: "模型名不能为空"}
	}
	if len(name) > 50 {
		return &ValidationError{Msg: "模型名长度不能超过 50 个字符"}
	}
	return nil
}

// validateParams 校验温度与最大 token 范围
func validateParams(temperature float64, maxTokens int) error {
	if temperature < 0 || temperature > 2 {
		return &ValidationError{Msg: "temperature 必须在 0-2 之间"}
	}
	if maxTokens < 1 || maxTokens > 128000 {
		return &ValidationError{Msg: "max_tokens 必须在 1-128000 之间"}
	}
	return nil
}

// encryptAPIKey 加密明文 api_key；明文为空直接返回空（无鉴权本地服务）
// 明文非空但未配置 MODEL_KEY_SECRET 时返回明确错误
func (s *ModelService) encryptAPIKey(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if s.keySecret() == "" {
		return "", &ValidationError{Msg: "未配置 MODEL_KEY_SECRET 环境变量，无法存储 API Key"}
	}
	enc, err := alchemycrypto.Encrypt(plain, s.keySecret())
	if err != nil {
		return "", fmt.Errorf("加密 API Key 失败: %w", err)
	}
	return enc, nil
}

// nameExists 检查模型名是否已被其他记录占用
func nameExists(db *gorm.DB, name string, excludeID uint) (bool, error) {
	var count int64
	q := db.Model(&model.LLMModel{}).Where("name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Create 创建模型配置
func (s *ModelService) Create(req *model.CreateLLMModelRequest) (*model.LLMModelResponse, error) {
	if err := validateName(req.Name); err != nil {
		return nil, err
	}
	req.Name = strings.TrimSpace(req.Name)
	if strings.TrimSpace(req.DisplayName) == "" {
		return nil, &ValidationError{Msg: "显示名不能为空"}
	}
	if strings.TrimSpace(req.Provider) == "" {
		return nil, &ValidationError{Msg: "服务商不能为空"}
	}
	if err := validateBaseURL(req.BaseURL); err != nil {
		return nil, err
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	if err := validateParams(temperature, maxTokens); err != nil {
		return nil, err
	}

	exists, err := nameExists(dao.GetDB(), req.Name, 0)
	if err != nil {
		return nil, fmt.Errorf("校验模型名唯一性失败: %w", err)
	}
	if exists {
		return nil, &ValidationError{Msg: fmt.Sprintf("模型名「%s」已存在", req.Name)}
	}

	encrypted, err := s.encryptAPIKey(req.APIKey)
	if err != nil {
		return nil, err
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	m := model.LLMModel{
		Name:            req.Name,
		DisplayName:     strings.TrimSpace(req.DisplayName),
		Provider:        strings.TrimSpace(req.Provider),
		BaseURL:         req.BaseURL,
		APIKeyEncrypted: encrypted,
		Temperature:     temperature,
		MaxTokens:       maxTokens,
		IsEnabled:       isEnabled,
		IsDefault:       req.IsDefault,
		IsSynthesis:     req.IsSynthesis,
		SortOrder:       req.SortOrder,
	}

	// 设置默认/合成专用模型时，事务内清除其他记录的同名字段
	err = dao.GetDB().Transaction(func(tx *gorm.DB) error {
		if m.IsDefault {
			if err := tx.Model(&model.LLMModel{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return fmt.Errorf("清除其他默认模型失败: %w", err)
			}
		}
		if m.IsSynthesis {
			if err := tx.Model(&model.LLMModel{}).Where("is_synthesis = ?", true).Update("is_synthesis", false).Error; err != nil {
				return fmt.Errorf("清除其他合成专用模型失败: %w", err)
			}
		}
		if err := tx.Create(&m).Error; err != nil {
			return fmt.Errorf("创建模型配置失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	zap.L().Info("[炼丹炉] 新模型入炉",
		zap.Uint("id", m.ID),
		zap.String("name", m.Name),
		zap.String("provider", m.Provider))
	return s.toResponse(&m, 0), nil
}

// Update 更新模型配置（指针字段区分未传与置空）
// api_key: nil=不修改，空字符串=清除密钥，非空=重新加密存储
func (s *ModelService) Update(id uint, req *model.UpdateLLMModelRequest) (*model.LLMModelResponse, error) {
	var m model.LLMModel
	if err := dao.GetDB().First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("模型(id=%d)不存在: %w", id, err)
	}

	if req.Name != nil {
		if err := validateName(*req.Name); err != nil {
			return nil, err
		}
		name := strings.TrimSpace(*req.Name)
		exists, err := nameExists(dao.GetDB(), name, id)
		if err != nil {
			return nil, fmt.Errorf("校验模型名唯一性失败: %w", err)
		}
		if exists {
			return nil, &ValidationError{Msg: fmt.Sprintf("模型名「%s」已存在", name)}
		}
		m.Name = name
	}
	if req.DisplayName != nil {
		if strings.TrimSpace(*req.DisplayName) == "" {
			return nil, &ValidationError{Msg: "显示名不能为空"}
		}
		m.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.Provider != nil {
		if strings.TrimSpace(*req.Provider) == "" {
			return nil, &ValidationError{Msg: "服务商不能为空"}
		}
		m.Provider = strings.TrimSpace(*req.Provider)
	}
	if req.BaseURL != nil {
		if err := validateBaseURL(*req.BaseURL); err != nil {
			return nil, err
		}
		m.BaseURL = *req.BaseURL
	}
	if req.Temperature != nil {
		m.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		m.MaxTokens = *req.MaxTokens
	}
	if err := validateParams(m.Temperature, m.MaxTokens); err != nil {
		return nil, err
	}
	if req.APIKey != nil {
		encrypted, err := s.encryptAPIKey(*req.APIKey)
		if err != nil {
			return nil, err
		}
		m.APIKeyEncrypted = encrypted // 空字符串即清除密钥
	}
	if req.IsEnabled != nil {
		m.IsEnabled = *req.IsEnabled
	}
	if req.SortOrder != nil {
		m.SortOrder = *req.SortOrder
	}
	if req.IsDefault != nil {
		m.IsDefault = *req.IsDefault
	}
	if req.IsSynthesis != nil {
		m.IsSynthesis = *req.IsSynthesis
	}

	err := dao.GetDB().Transaction(func(tx *gorm.DB) error {
		if m.IsDefault {
			if err := tx.Model(&model.LLMModel{}).Where("is_default = ? AND id <> ?", true, m.ID).Update("is_default", false).Error; err != nil {
				return fmt.Errorf("清除其他默认模型失败: %w", err)
			}
		}
		if m.IsSynthesis {
			if err := tx.Model(&model.LLMModel{}).Where("is_synthesis = ? AND id <> ?", true, m.ID).Update("is_synthesis", false).Error; err != nil {
				return fmt.Errorf("清除其他合成专用模型失败: %w", err)
			}
		}
		if err := tx.Save(&m).Error; err != nil {
			return fmt.Errorf("更新模型配置失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	zap.L().Info("[炼丹炉] 模型配置已更新", zap.Uint("id", m.ID), zap.String("name", m.Name))
	return s.GetByID(m.ID)
}

// Delete 删除模型配置；被道人引用时返回 ModelReferencedError（含引用数量）
func (s *ModelService) Delete(id uint) error {
	var m model.LLMModel
	if err := dao.GetDB().First(&m, id).Error; err != nil {
		return fmt.Errorf("模型(id=%d)不存在: %w", id, err)
	}

	count, err := s.referencedCount(m.Name)
	if err != nil {
		return err
	}
	if count > 0 {
		return &ModelReferencedError{Count: count}
	}

	if err := dao.GetDB().Delete(&m).Error; err != nil {
		return fmt.Errorf("删除模型配置失败: %w", err)
	}

	zap.L().Info("[炼丹炉] 模型配置已移除", zap.Uint("id", id), zap.String("name", m.Name))
	return nil
}

// TestConnection 以该模型凭证发起一次最小 LLM 调用（max_tokens=1），测量延迟
func (s *ModelService) TestConnection(ctx context.Context, id uint) (*model.TestConnectionResult, error) {
	var m model.LLMModel
	if err := dao.GetDB().First(&m, id).Error; err != nil {
		return nil, fmt.Errorf("模型(id=%d)不存在: %w", id, err)
	}

	apiKey, err := s.decryptAPIKey(&m)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]interface{}{
		"model":      m.Name,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens": 1,
	}
	if m.BaseURL != "" {
		reqBody["base_url"] = m.BaseURL
	}
	if apiKey != "" {
		reqBody["api_key"] = apiKey
	}
	jsonBody, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/api/v1/chat/completions", s.engineBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("构建连接测试请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &model.TestConnectionResult{Success: false, LatencyMs: latency, Error: MapEngineError(err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &model.TestConnectionResult{
			Success:   false,
			LatencyMs: latency,
			Error:     MapEngineError(&EngineError{Op: "模型连接测试", StatusCode: resp.StatusCode, Body: string(body)}),
		}, nil
	}

	return &model.TestConnectionResult{Success: true, LatencyMs: latency, Error: ""}, nil
}

// Options 已启用模型的精简列表，供道人表单下拉使用
func (s *ModelService) Options() ([]model.LLMModelOption, error) {
	var models []model.LLMModel
	if err := dao.GetDB().Where("is_enabled = ?", true).
		Order("sort_order ASC, id ASC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询启用模型列表失败: %w", err)
	}

	options := make([]model.LLMModelOption, 0, len(models))
	for _, m := range models {
		options = append(options, model.LLMModelOption{
			Name:        m.Name,
			DisplayName: m.DisplayName,
			Provider:    m.Provider,
			IsDefault:   m.IsDefault,
		})
	}
	return options, nil
}

// ValidateEnabledModel 校验模型名引用了已启用的模型配置（道人创建/更新时使用）
func (s *ModelService) ValidateEnabledModel(name string) error {
	var count int64
	if err := dao.GetDB().Model(&model.LLMModel{}).
		Where("name = ? AND is_enabled = ?", name, true).
		Count(&count).Error; err != nil {
		return fmt.Errorf("校验模型配置失败: %w", err)
	}
	if count == 0 {
		return &ValidationError{Msg: fmt.Sprintf("模型「%s」不存在或已停用，请先在模型管理中配置", name)}
	}
	return nil
}

// ResolveCredentials 解析对话模型的调用凭证
//   - 找到已启用模型：解密 api_key 返回完整凭证
//   - 找到但已停用：返回明确错误「该道人使用的模型已停用，请更换模型」
//   - 未找到：返回仅含模型名的空凭证（Python 回退环境变量配置，向后兼容）并记录警告
//   - name 为空：解析默认模型
func (s *ModelService) ResolveCredentials(name string) (*ModelCredentials, error) {
	if name == "" {
		return s.ResolveDefaultCredentials()
	}

	var m model.LLMModel
	err := dao.GetDB().Where("name = ?", name).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 向后兼容：模型表无此记录时透传模型名，Python 回退环境变量凭证
		zap.L().Warn("[炼丹炉] 模型未在模型管理中登记，回退环境变量凭证", zap.String("model", name))
		return &ModelCredentials{Model: name}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询模型配置失败: %w", err)
	}

	if !m.IsEnabled {
		return nil, errors.New("该道人使用的模型已停用，请更换模型")
	}

	apiKey, err := s.decryptAPIKey(&m)
	if err != nil {
		return nil, err
	}
	return &ModelCredentials{Model: m.Name, BaseURL: m.BaseURL, APIKey: apiKey}, nil
}

// ResolveDefaultCredentials 解析默认模型凭证；无默认模型时回退环境变量配置
func (s *ModelService) ResolveDefaultCredentials() (*ModelCredentials, error) {
	cfg := config.Get()

	var m model.LLMModel
	err := dao.GetDB().Where("is_default = ? AND is_enabled = ?", true, true).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Warn("[炼丹炉] 未配置默认模型，回退环境变量凭证", zap.String("model", cfg.LLM.DefaultModel))
		return &ModelCredentials{Model: cfg.LLM.DefaultModel}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询默认模型配置失败: %w", err)
	}

	apiKey, err := s.decryptAPIKey(&m)
	if err != nil {
		return nil, err
	}
	return &ModelCredentials{Model: m.Name, BaseURL: m.BaseURL, APIKey: apiKey}, nil
}

// ResolveSynthesisCredentials 解析语言模式合成专用模型凭证：is_synthesis 优先，回退 is_default
func (s *ModelService) ResolveSynthesisCredentials() (*ModelCredentials, error) {
	var m model.LLMModel
	err := dao.GetDB().Where("is_synthesis = ? AND is_enabled = ?", true, true).First(&m).Error
	switch {
	case err == nil:
		apiKey, derr := s.decryptAPIKey(&m)
		if derr != nil {
			return nil, derr
		}
		return &ModelCredentials{Model: m.Name, BaseURL: m.BaseURL, APIKey: apiKey}, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 无合成专用模型，回退默认模型
		return s.ResolveDefaultCredentials()
	default:
		return nil, fmt.Errorf("查询合成模型配置失败: %w", err)
	}
}
