// Package service 供应商管理业务逻辑层
// 管理 LLM 供应商配置（llm_providers 表）：CRUD、凭证加解密、连接测试、预置模板
// 供应商是协议 + Base URL + API Key 的唯一持有者；api_key 以 AES-GCM 加密存储
// （密钥来自 MODEL_KEY_SECRET），接口输出仅掩码形式，明文永不落日志
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/config"
	alchemycrypto "github.com/alchemy-furnace/server/internal/util/crypto"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProviderService 供应商管理业务逻辑
type ProviderService struct {
	engineBaseURL string
	httpClient    *http.Client
}

// NewProviderService 创建供应商管理业务实例
func NewProviderService() *ProviderService {
	return &ProviderService{
		engineBaseURL: config.Get().PythonEngine.BaseURL,
		httpClient:    &http.Client{Timeout: 60 * time.Second},
	}
}

// ProviderHasModelsError 供应商下仍有模型，拒绝删除（处理器映射为 HTTP 409）
type ProviderHasModelsError struct {
	Count int64
}

func (e *ProviderHasModelsError) Error() string {
	return fmt.Sprintf("该供应商下仍有 %d 个模型，无法删除", e.Count)
}

// ---------- 凭证加解密（供应商/模型服务共用） ----------

// keySecret 读取当前配置的加密密钥（动态读取便于测试与运行时配置）
func keySecret() string {
	return config.Get().ModelKeySecret
}

// encryptAPIKey 加密明文 api_key；明文为空直接返回空（免密钥本地服务）
// 明文非空但未配置 MODEL_KEY_SECRET 时返回明确错误
func encryptAPIKey(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if keySecret() == "" {
		return "", &ValidationError{Msg: "未配置 MODEL_KEY_SECRET 环境变量，无法存储 API Key"}
	}
	enc, err := alchemycrypto.Encrypt(plain, keySecret())
	if err != nil {
		return "", fmt.Errorf("加密 API Key 失败: %w", err)
	}
	return enc, nil
}

// decryptAPIKey 解密供应商的 api_key；未配置（空密文）返回空字符串
func decryptAPIKey(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	plain, err := alchemycrypto.Decrypt(encrypted, keySecret())
	if err != nil {
		return "", fmt.Errorf("供应商凭证解密失败，请检查 MODEL_KEY_SECRET 配置: %w", err)
	}
	return plain, nil
}

// ---------- 查询 ----------

// toResponse 转换为响应 DTO（解密仅用于生成掩码，明文不出现在任何输出中）
func (s *ProviderService) toResponse(p *model.LLMProvider, modelCount int64) *model.ProviderResponse {
	masked := ""
	if p.APIKeyEncrypted != "" {
		plain, err := decryptAPIKey(p.APIKeyEncrypted)
		if err != nil {
			zap.L().Warn("[炼丹炉] 供应商密钥解密失败，掩码降级显示", zap.Uint("provider_id", p.ID), zap.Error(err))
			masked = "****"
		} else {
			masked = MaskAPIKey(plain)
		}
	}
	return &model.ProviderResponse{
		ID:           p.ID,
		Name:         p.Name,
		DisplayName:  p.DisplayName,
		Protocol:     p.Protocol,
		BaseURL:      p.BaseURL,
		APIKeyMasked: masked,
		HasAPIKey:    p.APIKeyEncrypted != "",
		IsEnabled:    p.IsEnabled,
		SortOrder:    p.SortOrder,
		Remark:       p.Remark,
		ModelCount:   modelCount,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

// modelCounts 统计每个供应商下的模型数量
func (s *ProviderService) modelCounts() (map[uint]int64, error) {
	type row struct {
		ProviderID uint
		Cnt        int64
	}
	var rows []row
	if err := dao.GetDB().Model(&model.LLMModel{}).
		Select("provider_id, COUNT(*) AS cnt").
		Group("provider_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("统计供应商模型数量失败: %w", err)
	}
	counts := make(map[uint]int64, len(rows))
	for _, r := range rows {
		counts[r.ProviderID] = r.Cnt
	}
	return counts, nil
}

// modelCount 统计指定供应商下的模型数量
func (s *ProviderService) modelCount(providerID uint) (int64, error) {
	var count int64
	if err := dao.GetDB().Model(&model.LLMModel{}).
		Where("provider_id = ?", providerID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计供应商模型数量失败: %w", err)
	}
	return count, nil
}

// List 供应商列表，支持 enabled 过滤与分页，附带模型数量统计
func (s *ProviderService) List(page, pageSize int, enabled *bool) ([]model.ProviderResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	db := dao.GetDB().Model(&model.LLMProvider{})
	if enabled != nil {
		db = db.Where("is_enabled = ?", *enabled)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询供应商总数失败: %w", err)
	}

	var providers []model.LLMProvider
	offset := (page - 1) * pageSize
	if err := db.Order("sort_order ASC, id ASC").Offset(offset).Limit(pageSize).Find(&providers).Error; err != nil {
		return nil, 0, fmt.Errorf("查询供应商列表失败: %w", err)
	}

	counts, err := s.modelCounts()
	if err != nil {
		return nil, 0, err
	}

	list := make([]model.ProviderResponse, 0, len(providers))
	for i := range providers {
		list = append(list, *s.toResponse(&providers[i], counts[providers[i].ID]))
	}
	return list, total, nil
}

// GetByID 根据 ID 获取供应商详情（掩码形式）
func (s *ProviderService) GetByID(id uint) (*model.ProviderResponse, error) {
	var p model.LLMProvider
	if err := dao.GetDB().First(&p, id).Error; err != nil {
		return nil, fmt.Errorf("供应商(id=%d)不存在: %w", id, err)
	}
	count, err := s.modelCount(p.ID)
	if err != nil {
		return nil, err
	}
	return s.toResponse(&p, count), nil
}

// ---------- 校验 ----------

// validateProviderName 校验供应商标识长度
func validateProviderName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return &ValidationError{Msg: "供应商标识不能为空"}
	}
	if len(name) > 50 {
		return &ValidationError{Msg: "供应商标识长度不能超过 50 个字符"}
	}
	return nil
}

// validateProtocol 校验协议类型（当前仅支持 openai-compatible，预留扩展）
func validateProtocol(protocol string) error {
	if protocol != "openai-compatible" {
		return &ValidationError{Msg: fmt.Sprintf("暂不支持的协议类型「%s」，当前仅支持 openai-compatible", protocol)}
	}
	return nil
}

// providerNameExists 检查供应商标识是否已被其他记录占用
func providerNameExists(db *gorm.DB, name string, excludeID uint) (bool, error) {
	var count int64
	q := db.Model(&model.LLMProvider{}).Where("name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ---------- 写入 ----------

// Create 创建供应商配置
func (s *ProviderService) Create(req *model.CreateProviderRequest) (*model.ProviderResponse, error) {
	if err := validateProviderName(req.Name); err != nil {
		return nil, err
	}
	req.Name = strings.TrimSpace(req.Name)
	if strings.TrimSpace(req.DisplayName) == "" {
		return nil, &ValidationError{Msg: "显示名不能为空"}
	}
	if err := validateBaseURL(req.BaseURL); err != nil {
		return nil, err
	}
	protocol := strings.TrimSpace(req.Protocol)
	if protocol == "" {
		protocol = "openai-compatible"
	}
	if err := validateProtocol(protocol); err != nil {
		return nil, err
	}

	exists, err := providerNameExists(dao.GetDB(), req.Name, 0)
	if err != nil {
		return nil, fmt.Errorf("校验供应商标识唯一性失败: %w", err)
	}
	if exists {
		return nil, &ValidationError{Msg: fmt.Sprintf("供应商标识「%s」已存在", req.Name)}
	}

	encrypted, err := encryptAPIKey(req.APIKey)
	if err != nil {
		return nil, err
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	p := model.LLMProvider{
		Name:            req.Name,
		DisplayName:     strings.TrimSpace(req.DisplayName),
		Protocol:        protocol,
		BaseURL:         req.BaseURL,
		APIKeyEncrypted: encrypted,
		IsEnabled:       isEnabled,
		SortOrder:       req.SortOrder,
		Remark:          req.Remark,
	}
	if err := dao.GetDB().Create(&p).Error; err != nil {
		return nil, fmt.Errorf("创建供应商配置失败: %w", err)
	}

	zap.L().Info("[炼丹炉] 新供应商入炉",
		zap.Uint("id", p.ID),
		zap.String("name", p.Name),
		zap.String("protocol", p.Protocol))
	return s.toResponse(&p, 0), nil
}

// Update 更新供应商配置（指针字段区分未传与置空）
// api_key: nil=不修改，空字符串=清除密钥，非空=重新加密存储；更新后其下所有模型立即生效
func (s *ProviderService) Update(id uint, req *model.UpdateProviderRequest) (*model.ProviderResponse, error) {
	var p model.LLMProvider
	if err := dao.GetDB().First(&p, id).Error; err != nil {
		return nil, fmt.Errorf("供应商(id=%d)不存在: %w", id, err)
	}

	if req.Name != nil {
		if err := validateProviderName(*req.Name); err != nil {
			return nil, err
		}
		name := strings.TrimSpace(*req.Name)
		exists, err := providerNameExists(dao.GetDB(), name, id)
		if err != nil {
			return nil, fmt.Errorf("校验供应商标识唯一性失败: %w", err)
		}
		if exists {
			return nil, &ValidationError{Msg: fmt.Sprintf("供应商标识「%s」已存在", name)}
		}
		p.Name = name
	}
	if req.DisplayName != nil {
		if strings.TrimSpace(*req.DisplayName) == "" {
			return nil, &ValidationError{Msg: "显示名不能为空"}
		}
		p.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.Protocol != nil {
		if err := validateProtocol(*req.Protocol); err != nil {
			return nil, err
		}
		p.Protocol = *req.Protocol
	}
	if req.BaseURL != nil {
		if err := validateBaseURL(*req.BaseURL); err != nil {
			return nil, err
		}
		p.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil {
		encrypted, err := encryptAPIKey(*req.APIKey)
		if err != nil {
			return nil, err
		}
		p.APIKeyEncrypted = encrypted // 空字符串即清除密钥
	}
	if req.IsEnabled != nil {
		p.IsEnabled = *req.IsEnabled
	}
	if req.SortOrder != nil {
		p.SortOrder = *req.SortOrder
	}
	if req.Remark != nil {
		p.Remark = *req.Remark
	}

	if err := dao.GetDB().Save(&p).Error; err != nil {
		return nil, fmt.Errorf("更新供应商配置失败: %w", err)
	}

	zap.L().Info("[炼丹炉] 供应商配置已更新", zap.Uint("id", p.ID), zap.String("name", p.Name))
	return s.GetByID(p.ID)
}

// Delete 删除供应商配置；下有关联模型时返回 ProviderHasModelsError（含模型数量），不级联
func (s *ProviderService) Delete(id uint) error {
	var p model.LLMProvider
	if err := dao.GetDB().First(&p, id).Error; err != nil {
		return fmt.Errorf("供应商(id=%d)不存在: %w", id, err)
	}

	count, err := s.modelCount(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return &ProviderHasModelsError{Count: count}
	}

	if err := dao.GetDB().Delete(&p).Error; err != nil {
		return fmt.Errorf("删除供应商配置失败: %w", err)
	}

	zap.L().Info("[炼丹炉] 供应商配置已移除", zap.Uint("id", id), zap.String("name", p.Name))
	return nil
}

// TestConnection 以供应商凭证发起一次最小 LLM 调用（max_tokens=1），测量延迟
// modelName 可选：缺省用该供应商下第一个启用模型；无启用模型且未传 → 校验错误
func (s *ProviderService) TestConnection(ctx context.Context, id uint, modelName string) (*model.TestConnectionResult, error) {
	var p model.LLMProvider
	if err := dao.GetDB().First(&p, id).Error; err != nil {
		return nil, fmt.Errorf("供应商(id=%d)不存在: %w", id, err)
	}

	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		var m model.LLMModel
		err := dao.GetDB().Where("provider_id = ? AND is_enabled = ?", p.ID, true).
			Order("sort_order ASC, id ASC").First(&m).Error
		if err != nil {
			return nil, &ValidationError{Msg: "请先为该供应商添加模型"}
		}
		modelName = m.Name
	}

	apiKey, err := decryptAPIKey(p.APIKeyEncrypted)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]interface{}{
		"model":      modelName,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens": 1,
	}
	if p.BaseURL != "" {
		reqBody["base_url"] = p.BaseURL
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
			Error:     MapEngineError(&EngineError{Op: "供应商连接测试", StatusCode: resp.StatusCode, Body: string(body)}),
		}, nil
	}

	return &model.TestConnectionResult{Success: true, LatencyMs: latency, Error: ""}, nil
}

// Templates 返回预置供应商模板清单（静态常量，前后端单一数据源）
func (s *ProviderService) Templates() []model.ProviderTemplate {
	return model.ProviderTemplates
}
