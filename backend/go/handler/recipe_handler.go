// Package handler 丹方管理 HTTP 处理器
// 处理丹方文件的上传、查询、删除和重新提取
// 对应 RESTful API: /api/v1/recipes
package handler

import (
	"strconv"

	"github.com/alchemy-furnace/server/pkg/response"
	"github.com/alchemy-furnace/server/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RecipeHandler 丹方 HTTP 处理器
type RecipeHandler struct {
	service *service.RecipeService
}

// NewRecipeHandler 创建丹方处理器
func NewRecipeHandler() *RecipeHandler {
	return &RecipeHandler{
		service: service.NewRecipeService(),
	}
}

// ListRecipesByPill 获取金丹下的丹方列表
// GET /api/v1/recipes/pill/:pill_id?page=1&page_size=10
func (h *RecipeHandler) ListRecipesByPill(c *gin.Context) {
	pillID, err := strconv.ParseUint(c.Param("pill_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "金丹ID格式不正确")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	recipes, total, err := h.service.ListRecipesByPill(uint(pillID), page, pageSize)
	if err != nil {
		zap.L().Error("[炼丹炉] 查询丹方列表失败", zap.Error(err))
		response.InternalError(c, "查询丹方列表失败")
		return
	}

	response.SuccessWithPage(c, total, page, pageSize, recipes)
}

// UploadRecipes 上传丹方文件（支持多文件）
// POST /api/v1/recipes/upload
// Content-Type: multipart/form-data
// Form: pill_id=1&files[]=@file1.doc&files[]=@file2.pdf
func (h *RecipeHandler) UploadRecipes(c *gin.Context) {
	// 解析 pill_id
	pillIDStr := c.PostForm("pill_id")
	if pillIDStr == "" {
		response.BadRequest(c, "缺少金丹ID(pill_id)")
		return
	}
	pillID, err := strconv.ParseUint(pillIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "金丹ID格式不正确")
		return
	}

	// 解析多文件上传（字段名: files[]）
	form, err := c.MultipartForm()
	if err != nil {
		response.BadRequest(c, "解析上传文件失败: "+err.Error())
		return
	}

	files := form.File["files[]"]
	if len(files) == 0 {
		response.BadRequest(c, "未上传任何文件")
		return
	}

	// 限制单次上传文件数量
	if len(files) > 20 {
		response.BadRequest(c, "单次最多上传20个文件")
		return
	}

	// 调用业务层处理上传
	recipes, err := h.service.UploadRecipes(uint(pillID), files)
	if err != nil {
		zap.L().Error("[炼丹炉] 上传丹方失败", zap.Error(err))
		response.Error(c, response.CodeFileUploadError, "上传失败: "+err.Error())
		return
	}

	zap.L().Info("[炼丹炉] 丹方上传成功", zap.Int("count", len(recipes)))
	response.Success(c, gin.H{
		"uploaded": len(recipes),
		"recipes":  recipes,
	})
}

// DeleteRecipe 删除丹方
// DELETE /api/v1/recipes/:id
func (h *RecipeHandler) DeleteRecipe(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "丹方ID格式不正确")
		return
	}

	if err := h.service.DeleteRecipe(uint(id)); err != nil {
		zap.L().Error("[炼丹炉] 删除丹方失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "删除丹方失败")
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

// ReExtract 重新提取丹方
// POST /api/v1/recipes/:id/re-extract
func (h *RecipeHandler) ReExtract(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "丹方ID格式不正确")
		return
	}

	if err := h.service.ReExtract(uint(id)); err != nil {
		zap.L().Error("[炼丹炉] 重新提取失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "重新提取失败")
		return
	}

	response.Success(c, gin.H{"re_extracting": true})
}
