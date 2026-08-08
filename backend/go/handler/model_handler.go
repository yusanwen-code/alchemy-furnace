// Package handler 模型管理 HTTP 处理器
// 处理 LLM 模型配置的增删改查与连接测试
// 对应 RESTful API: /api/v1/models（契约见 contracts/models-api.md）
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

// ModelHandler 模型管理 HTTP 处理器
type ModelHandler struct {
	service *service.ModelService
}

// NewModelHandler 创建模型管理处理器
func NewModelHandler() *ModelHandler {
	return &ModelHandler{
		service: service.NewModelService(),
	}
}

// handleModelError 统一处理模型业务错误：校验错误 → 400，引用冲突 → 409，其余 → 500
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
	zap.L().Error(logMsg, zap.Error(err))
	response.InternalError(c, err.Error())
}

// ListModels 模型列表
// GET /api/v1/models?enabled=true&page=1&page_size=50
func (h *ModelHandler) ListModels(c *gin.Context) {
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

	models, total, err := h.service.List(page, pageSize, enabled)
	if err != nil {
		zap.L().Error("[炼丹炉] 查询模型列表失败", zap.Error(err))
		response.InternalError(c, "查询模型列表失败")
		return
	}

	response.SuccessWithPage(c, total, page, pageSize, models)
}

// CreateModel 创建模型
// POST /api/v1/models
// Body: { "name": "deepseek-chat", "display_name": "DeepSeek-V3", "provider": "deepseek",
//
//	"base_url": "https://api.deepseek.com/v1", "api_key": "sk-...", ... }
func (h *ModelHandler) CreateModel(c *gin.Context) {
	var req model.CreateLLMModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	created, err := h.service.Create(&req)
	if err != nil {
		handleModelError(c, err, "[炼丹炉] 创建模型失败")
		return
	}

	response.Created(c, created)
}

// UpdateModel 更新模型
// PUT /api/v1/models/:id
// api_key 不传 = 不修改，传空字符串 = 清除密钥，传值 = 重新加密存储
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

	updated, err := h.service.Update(uint(id), &req)
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

	if err := h.service.Delete(uint(id)); err != nil {
		handleModelError(c, err, "[炼丹炉] 删除模型失败")
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

// TestConnection 模型连接测试
// POST /api/v1/models/:id/test-connection
// 以该模型凭证发起一次最小 LLM 调用（max_tokens=1），返回 { success, latency_ms, error }
func (h *ModelHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "模型ID格式不正确")
		return
	}

	result, err := h.service.TestConnection(c.Request.Context(), uint(id))
	if err != nil {
		handleModelError(c, err, "[炼丹炉] 模型连接测试失败")
		return
	}

	response.Success(c, result)
}

// ListOptions 已启用模型的精简列表，供道人表单下拉使用
// GET /api/v1/models/options
func (h *ModelHandler) ListOptions(c *gin.Context) {
	options, err := h.service.Options()
	if err != nil {
		zap.L().Error("[炼丹炉] 查询模型选项列表失败", zap.Error(err))
		response.InternalError(c, "查询模型选项列表失败")
		return
	}

	response.Success(c, options)
}
