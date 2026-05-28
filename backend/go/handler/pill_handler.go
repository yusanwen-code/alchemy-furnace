// Package handler 金丹管理 HTTP 处理器
// 处理金丹的增删改查接口，对应 RESTful API: /api/v1/pills
// 金丹是知识库的载体，删除金丹会级联删除其丹方和向量数据
package handler

import (
	"strconv"

	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/response"
	"github.com/alchemy-furnace/server/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PillHandler 金丹 HTTP 处理器
type PillHandler struct {
	service *service.PillService
}

// NewPillHandler 创建金丹处理器
func NewPillHandler() *PillHandler {
	return &PillHandler{
		service: service.NewPillService(),
	}
}

// ListPills 金丹列表
// GET /api/v1/pills?page=1&page_size=10&keyword=xxx
func (h *PillHandler) ListPills(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	// 调用业务层
	pills, total, err := h.service.ListPills(page, pageSize, keyword)
	if err != nil {
		zap.L().Error("[炼丹炉] 查询金丹列表失败", zap.Error(err))
		response.InternalError(c, "查询金丹列表失败")
		return
	}

	response.SuccessWithPage(c, total, page, pageSize, pills)
}

// GetPill 金丹详情
// GET /api/v1/pills/:id
func (h *PillHandler) GetPill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "金丹ID格式不正确")
		return
	}

	pill, err := h.service.GetPill(uint(id))
	if err != nil {
		zap.L().Error("[炼丹炉] 查询金丹详情失败", zap.Uint64("id", id), zap.Error(err))
		response.NotFound(c, "金丹不存在")
		return
	}

	response.Success(c, pill)
}

// CreatePill 创建金丹
// POST /api/v1/pills
// Body: { "name": "金丹名称", "description": "描述" }
func (h *PillHandler) CreatePill(c *gin.Context) {
	var req model.CreatePillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	pill, err := h.service.CreatePill(&req)
	if err != nil {
		zap.L().Error("[炼丹炉] 创建金丹失败", zap.Error(err))
		response.InternalError(c, "创建金丹失败")
		return
	}

	response.Created(c, pill)
}

// UpdatePill 更新金丹
// PUT /api/v1/pills/:id
// Body: { "name": "新名称", "description": "新描述" }
func (h *PillHandler) UpdatePill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "金丹ID格式不正确")
		return
	}

	var req model.UpdatePillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	pill, err := h.service.UpdatePill(uint(id), &req)
	if err != nil {
		zap.L().Error("[炼丹炉] 更新金丹失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "更新金丹失败")
		return
	}

	response.Success(c, pill)
}

// DeletePill 删除金丹（级联删除向量和丹方）
// DELETE /api/v1/pills/:id
func (h *PillHandler) DeletePill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "金丹ID格式不正确")
		return
	}

	if err := h.service.DeletePill(uint(id)); err != nil {
		zap.L().Error("[炼丹炉] 删除金丹失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "删除金丹失败")
		return
	}

	response.Success(c, gin.H{"deleted": true})
}
