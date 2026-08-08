// Package handler 供应商与模型管理 HTTP 处理器
// 处理供应商（凭证持有者）的增删改查/连接测试/预置模板，以及供应商下模型的嵌套管理
// 对应 RESTful API: /api/v1/providers, /api/v1/models（契约见 contracts/providers-api.md）
// api_key 永不明文返回，仅返回掩码形式
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/response"
	"github.com/alchemy-furnace/server/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ModelHandler 供应商与模型管理 HTTP 处理器
type ModelHandler struct {
	providers *service.ProviderService
	models    *service.ModelService
}

// NewModelHandler 创建供应商与模型管理处理器
func NewModelHandler() *ModelHandler {
	return &ModelHandler{
		providers: service.NewProviderService(),
		models:    service.NewModelService(),
	}
}

// handleModelError 统一处理模型业务错误：校验错误 → 400，引用/级联冲突 → 409，其余 → 500
func handleModelError(c *gin.Context, err error, logMsg string) {
	var validationErr *service.ValidationError
	if errors.As(err, &validationErr) {
		response.BadRequest(c, validationErr.Msg)
		return
	}
	var referencedErr *service.ModelReferencedError
	if errors.As(err, &referencedErr) {
		c.JSON(http.StatusConflict, response.Response{
			Code:    http.StatusConflict,
			Message: referencedErr.Error(),
			Data:    gin.H{"referenced_by": referencedErr.Count},
		})
		return
	}
	var hasModelsErr *service.ProviderHasModelsError
	if errors.As(err, &hasModelsErr) {
		c.JSON(http.StatusConflict, response.Response{
			Code:    http.StatusConflict,
			Message: hasModelsErr.Error(),
			Data:    gin.H{"model_count": hasModelsErr.Count},
		})
		return
	}
	zap.L().Error(logMsg, zap.Error(err))
	response.InternalError(c, err.Error())
}

// ---------- 预置模板 ----------

// ListTemplates 预置供应商模板清单
// GET /api/v1/providers/templates
func (h *ModelHandler) ListTemplates(c *gin.Context) {
	response.Success(c, h.providers.Templates())
}

// ---------- 供应商管理 ----------

// ListProviders 供应商列表
// GET /api/v1/providers?enabled=true&page=1&page_size=50
func (h *ModelHandler) ListProviders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	// enabled 过滤：仅显式传 true/false 时生效
	var enabled *bool
	if raw := c.Query("enabled"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "enabled 参数仅支持 true/false")
			return
		}
		enabled = &v
	}

	providers, total, err := h.providers.List(page, pageSize, enabled)
	if err != nil {
		zap.L().Error("[炼丹炉] 查询供应商列表失败", zap.Error(err))
		response.InternalError(c, "查询供应商列表失败")
		return
	}

	response.SuccessWithPage(c, total, page, pageSize, providers)
}

// CreateProvider 创建供应商
// POST /api/v1/providers
// Body: { "name": "deepseek", "display_name": "DeepSeek", "protocol": "openai-compatible",
//
//	"base_url": "https://api.deepseek.com/v1", "api_key": "sk-...", ... }
func (h *ModelHandler) CreateProvider(c *gin.Context) {
	var req model.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	created, err := h.providers.Create(&req)
	if err != nil {
		handleModelError(c, err, "[炼丹炉] 创建供应商失败")
		return
	}

	response.Created(c, created)
}

// UpdateProvider 更新供应商
// PUT /api/v1/providers/:id
// api_key 不传 = 不修改，传空字符串 = 清除密钥，传值 = 重新加密存储
func (h *ModelHandler) UpdateProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "供应商ID格式不正确")
		return
	}

	var req model.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	updated, err := h.providers.Update(uint(id), &req)
	if err != nil {
		handleModelError(c, err, "[炼丹炉] 更新供应商失败")
		return
	}

	response.Success(c, updated)
}

// DeleteProvider 删除供应商
// DELETE /api/v1/providers/:id
// 下有关联模型时返回 409: { "code": 409, "message": "该供应商下仍有 N 个模型，无法删除", "data": { "model_count": N } }
func (h *ModelHandler) DeleteProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "供应商ID格式不正确")
		return
	}

	if err := h.providers.Delete(uint(id)); err != nil {
		handleModelError(c, err, "[炼丹炉] 删除供应商失败")
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

// TestProviderConnection 供应商连接测试
// POST /api/v1/providers/:id/test-connection
// Body（可选）: { "model": "deepseek-chat" }，缺省用该供应商下第一个启用模型
// 以供应商凭证发起一次最小 LLM 调用（max_tokens=1），返回 { success, latency_ms, error }
func (h *ModelHandler) TestProviderConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "供应商ID格式不正确")
		return
	}

	// body 可选：允许空 body，绑定失败（EOF）不视为错误
	var req model.TestConnectionRequest
	_ = c.ShouldBindJSON(&req)

	result, err := h.providers.TestConnection(c.Request.Context(), uint(id), req.Model)
	if err != nil {
		handleModelError(c, err, "[炼丹炉] 供应商连接测试失败")
		return
	}

	response.Success(c, result)
}

// ---------- 供应商下的模型管理 ----------

// ListProviderModels 供应商下的模型列表（含 referenced_by 引用数）
// GET /api/v1/providers/:id/models
func (h *ModelHandler) ListProviderModels(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "供应商ID格式不正确")
		return
	}

	models, err := h.models.ListByProvider(uint(id))
	if err != nil {
		handleModelError(c, err, "[炼丹炉] 查询供应商下模型列表失败")
		return
	}

	response.Success(c, models)
}

// CreateProviderModel 在供应商下创建模型
// POST /api/v1/providers/:id/models
// Body: { "name": "deepseek-chat", "display_name": "DeepSeek-V3", "temperature": 0.7, ... }
func (h *ModelHandler) CreateProviderModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "供应商ID格式不正确")
		return
	}

	var req model.CreateLLMModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	created, err := h.models.Create(uint(id), &req)
	if err != nil {
		handleModelError(c, err, "[炼丹炉] 创建模型失败")
		return
	}

	response.Created(c, created)
}

// UpdateModel 更新模型
// PUT /api/v1/models/:id
// is_default/is_synthesis 置 true 时事务内清除其他记录
func (h *ModelHandler) UpdateModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "模型ID格式不正确")
		return
	}

	var req model.UpdateLLMModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	updated, err := h.models.Update(uint(id), &req)
	if err != nil {
		handleModelError(c, err, "[炼丹炉] 更新模型失败")
		return
	}

	response.Success(c, updated)
}

// DeleteModel 删除模型
// DELETE /api/v1/models/:id
// 被道人引用时返回 409: { "code": 409, "message": "该模型仍被 N 个道人引用，无法删除", "data": { "referenced_by": N } }
func (h *ModelHandler) DeleteModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "模型ID格式不正确")
		return
	}

	if err := h.models.Delete(uint(id)); err != nil {
		handleModelError(c, err, "[炼丹炉] 删除模型失败")
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

// ListOptions 已启用供应商下的已启用模型精简列表（含供应商显示名），供道人表单下拉使用
// GET /api/v1/models/options
func (h *ModelHandler) ListOptions(c *gin.Context) {
	options, err := h.models.Options()
	if err != nil {
		zap.L().Error("[炼丹炉] 查询模型选项列表失败", zap.Error(err))
		response.InternalError(c, "查询模型选项列表失败")
		return
	}

	response.Success(c, options)
}
