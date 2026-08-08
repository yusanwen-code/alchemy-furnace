// Package service 模型管理业务逻辑层
// 管理 LLM 模型配置（llm_models 表）：归属供应商的模型 CRUD、两级凭证解析链
// 凭证（base_url/api_key）由供应商持有，模型仅声明模型名与生成参数
package service

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ModelService 模型管理业务逻辑
type ModelService struct{}

// NewModelService 创建模型管理业务实例
func NewModelService() *ModelService {
	return &ModelService{}
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

// toResponse 转换为响应 DTO（凭证在供应商上，模型响应仅携带供应商引用信息）
func (s *ModelService) toResponse(m *model.LLMModel, referencedBy int64) *model.LLMModelResponse {
	return &model.LLMModelResponse{
		ID:                  m.ID,
		ProviderID:          m.ProviderID,
		Name:                m.Name,
		DisplayName:         m.DisplayName,
		ProviderName:        m.Provider.Name,
		ProviderDisplayName: m.Provider.DisplayName,
		Temperature:         m.Temperature,
		MaxTokens:           m.MaxTokens,
		IsEnabled:           m.IsEnabled,
		IsDefault:           m.IsDefault,
		IsSynthesis:         m.IsSynthesis,
		SortOrder:           m.SortOrder,
		ReferencedBy:        referencedBy,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
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

// ListByProvider 查询指定供应商下的模型列表，附带引用计数
func (s *ModelService) ListByProvider(providerID uint) ([]model.LLMModelResponse, error) {
	var p model.LLMProvider
	if err := dao.GetDB().First(&p, providerID).Error; err != nil {
		return nil, fmt.Errorf("供应商(id=%d)不存在: %w", providerID, err)
	}

	var models []model.LLMModel
	if err := dao.GetDB().Where("provider_id = ?", providerID).
		Order("sort_order ASC, id ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询供应商下模型列表失败: %w", err)
	}

	counts, err := s.referencedCounts()
	if err != nil {
		return nil, err
	}

	list := make([]model.LLMModelResponse, 0, len(models))
	for i := range models {
		models[i].Provider = p
		list = append(list, *s.toResponse(&models[i], counts[models[i].Name]))
	}
	return list, nil
}

// GetByID 根据 ID 获取模型详情
func (s *ModelService) GetByID(id uint) (*model.LLMModelResponse, error) {
	var m model.LLMModel
	if err := dao.GetDB().Preload("Provider").First(&m, id).Error; err != nil {
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
	if len(name) > 100 {
		return &ValidationError{Msg: "模型名长度不能超过 100 个字符"}
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

// nameExistsInProvider 检查同供应商下模型名是否已被其他记录占用
func nameExistsInProvider(db *gorm.DB, providerID uint, name string, excludeID uint) (bool, error) {
	var count int64
	q := db.Model(&model.LLMModel{}).Where("provider_id = ? AND name = ?", providerID, name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// loadEnabledProvider 加载供应商并校验存在且已启用（创建模型前置校验）
func loadEnabledProvider(providerID uint) (*model.LLMProvider, error) {
	var p model.LLMProvider
	if err := dao.GetDB().First(&p, providerID).Error; err != nil {
		return nil, &ValidationError{Msg: fmt.Sprintf("供应商(id=%d)不存在", providerID)}
	}
	if !p.IsEnabled {
		return nil, &ValidationError{Msg: fmt.Sprintf("供应商「%s」已停用，请先启用后再添加模型", p.DisplayName)}
	}
	return &p, nil
}

// Create 在指定供应商下创建模型配置
func (s *ModelService) Create(providerID uint, req *model.CreateLLMModelRequest) (*model.LLMModelResponse, error) {
	p, err := loadEnabledProvider(providerID)
	if err != nil {
		return nil, err
	}

	if err := validateName(req.Name); err != nil {
		return nil, err
	}
	req.Name = strings.TrimSpace(req.Name)
	if strings.TrimSpace(req.DisplayName) == "" {
		return nil, &ValidationError{Msg: "显示名不能为空"}
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

	exists, err := nameExistsInProvider(dao.GetDB(), providerID, req.Name, 0)
	if err != nil {
		return nil, fmt.Errorf("校验模型名唯一性失败: %w", err)
	}
	if exists {
		return nil, &ValidationError{Msg: fmt.Sprintf("该供应商下模型名「%s」已存在", req.Name)}
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	m := model.LLMModel{
		ProviderID:  providerID,
		Name:        req.Name,
		DisplayName: strings.TrimSpace(req.DisplayName),
		Temperature: temperature,
		MaxTokens:   maxTokens,
		IsEnabled:   isEnabled,
		IsDefault:   req.IsDefault,
		IsSynthesis: req.IsSynthesis,
		SortOrder:   req.SortOrder,
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
		zap.String("provider", p.Name))
	m.Provider = *p
	return s.toResponse(&m, 0), nil
}

// Update 更新模型配置（指针字段区分未传与置空）
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
		exists, err := nameExistsInProvider(dao.GetDB(), m.ProviderID, name, id)
		if err != nil {
			return nil, fmt.Errorf("校验模型名唯一性失败: %w", err)
		}
		if exists {
			return nil, &ValidationError{Msg: fmt.Sprintf("该供应商下模型名「%s」已存在", name)}
		}
		m.Name = name
	}
	if req.DisplayName != nil {
		if strings.TrimSpace(*req.DisplayName) == "" {
			return nil, &ValidationError{Msg: "显示名不能为空"}
		}
		m.DisplayName = strings.TrimSpace(*req.DisplayName)
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

// Options 已启用供应商下的已启用模型精简列表，供道人表单下拉使用
// 按供应商 sort_order + 模型 sort_order 排序
func (s *ModelService) Options() ([]model.LLMModelOption, error) {
	type optionRow struct {
		Name                string
		DisplayName         string
		IsDefault           bool
		ProviderName        string
		ProviderDisplayName string
	}
	var rows []optionRow
	if err := dao.GetDB().Table("llm_models").
		Select("llm_models.name, llm_models.display_name, llm_models.is_default, llm_providers.name AS provider_name, llm_providers.display_name AS provider_display_name").
		Joins("JOIN llm_providers ON llm_providers.id = llm_models.provider_id").
		Where("llm_models.is_enabled = ? AND llm_providers.is_enabled = ?", true, true).
		Order("llm_providers.sort_order ASC, llm_providers.id ASC, llm_models.sort_order ASC, llm_models.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询启用模型列表失败: %w", err)
	}

	options := make([]model.LLMModelOption, 0, len(rows))
	for _, r := range rows {
		options = append(options, model.LLMModelOption{
			Name:                r.Name,
			DisplayName:         r.DisplayName,
			ProviderName:        r.ProviderName,
			ProviderDisplayName: r.ProviderDisplayName,
			IsDefault:           r.IsDefault,
		})
	}
	return options, nil
}

// ValidateEnabledModel 校验模型名引用了已启用供应商下的已启用模型配置（道人创建/更新时使用）
func (s *ModelService) ValidateEnabledModel(name string) error {
	var count int64
	if err := dao.GetDB().Table("llm_models").
		Joins("JOIN llm_providers ON llm_providers.id = llm_models.provider_id").
		Where("llm_models.name = ? AND llm_models.is_enabled = ? AND llm_providers.is_enabled = ?", name, true, true).
		Count(&count).Error; err != nil {
		return fmt.Errorf("校验模型配置失败: %w", err)
	}
	if count == 0 {
		return &ValidationError{Msg: fmt.Sprintf("模型「%s」不存在或已停用，请先在模型管理中配置", name)}
	}
	return nil
}

// ---------- 两级凭证解析链（model → provider → 解密） ----------

// resolveModelCredentials 由已启用模型解析完整调用凭证：加载所属供应商 → 校验启用 → 解密 api_key
func (s *ModelService) resolveModelCredentials(m *model.LLMModel) (*ModelCredentials, error) {
	var p model.LLMProvider
	if err := dao.GetDB().First(&p, m.ProviderID).Error; err != nil {
		return nil, fmt.Errorf("查询模型所属供应商配置失败: %w", err)
	}
	if !p.IsEnabled {
		return nil, errors.New("该模型所属供应商已停用")
	}

	apiKey, err := decryptAPIKey(p.APIKeyEncrypted)
	if err != nil {
		return nil, err
	}
	return &ModelCredentials{Model: m.Name, BaseURL: p.BaseURL, APIKey: apiKey}, nil
}

// ResolveCredentials 解析对话模型的调用凭证（两级解析链：model → provider → 解密）
//   - 同名模型存在于多个供应商时，取 sort_order,id 最小者并记录 warning
//   - 找到已启用模型：返回供应商凭证（base_url + 解密 api_key）
//   - 找到但全部已停用：返回明确错误「该道人使用的模型已停用，请更换模型」
//   - 未找到：返回仅含模型名的空凭证（Python 回退环境变量配置，向后兼容）并记录警告
//   - name 为空：解析默认模型
func (s *ModelService) ResolveCredentials(name string) (*ModelCredentials, error) {
	if name == "" {
		return s.ResolveDefaultCredentials()
	}

	var models []model.LLMModel
	if err := dao.GetDB().Where("name = ?", name).
		Order("sort_order ASC, id ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查询模型配置失败: %w", err)
	}
	if len(models) == 0 {
		// 向后兼容：模型表无此记录时透传模型名，Python 回退环境变量凭证
		zap.L().Warn("[炼丹炉] 模型未在模型管理中登记，回退环境变量凭证", zap.String("model", name))
		return &ModelCredentials{Model: name}, nil
	}

	// 同名模型可能跨供应商存在多个：取第一个已启用者
	var selected *model.LLMModel
	enabledCount := 0
	for i := range models {
		if models[i].IsEnabled {
			enabledCount++
			if selected == nil {
				selected = &models[i]
			}
		}
	}
	if selected == nil {
		return nil, errors.New("该道人使用的模型已停用，请更换模型")
	}
	if enabledCount > 1 {
		zap.L().Warn("[炼丹炉] 同名模型存在于多个供应商，按 sort_order,id 取第一个",
			zap.String("model", name),
			zap.Uint("selected_id", selected.ID),
			zap.Uint("provider_id", selected.ProviderID))
	}

	return s.resolveModelCredentials(selected)
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

	return s.resolveModelCredentials(&m)
}

// ResolveSynthesisCredentials 解析语言模式合成专用模型凭证：is_synthesis 优先，回退 is_default
func (s *ModelService) ResolveSynthesisCredentials() (*ModelCredentials, error) {
	var m model.LLMModel
	err := dao.GetDB().Where("is_synthesis = ? AND is_enabled = ?", true, true).First(&m).Error
	switch {
	case err == nil:
		return s.resolveModelCredentials(&m)
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 无合成专用模型，回退默认模型
		return s.ResolveDefaultCredentials()
	default:
		return nil, fmt.Errorf("查询合成模型配置失败: %w", err)
	}
}
