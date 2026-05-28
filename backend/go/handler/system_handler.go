// Package handler 系统管理 HTTP 处理器
// 提供健康检查和配置查询接口
// 对应 API: /api/v1/system/health, /api/v1/system/config
package handler

import (
	"net/http"
	"time"

	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/config"
	"github.com/alchemy-furnace/server/pkg/response"
	"github.com/gin-gonic/gin"
)

// SystemHandler 系统处理器
type SystemHandler struct {
	version string
}

// NewSystemHandler 创建系统处理器
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{
		version: "1.0.0",
	}
}

// HealthCheck 健康检查
// GET /api/v1/system/health
// 返回各个组件的健康状态，用于监控和负载均衡
func (h *SystemHandler) HealthCheck(c *gin.Context) {
	health := model.HealthCheckResponse{
		Status:    "ok",
		Version:   h.version,
		Timestamp: time.Now().Unix(),
		DB:        "ok",
		Qdrant:    "unknown", // 可通过实际请求检测
		PythonRAG: "unknown", // 可通过实际请求检测
	}

	// TODO: 可以在这里添加对 Qdrant 和 Python RAG 的实际连通性检查

	c.JSON(http.StatusOK, health)
}

// GetConfig 获取系统配置
// GET /api/v1/system/config
// 返回前端需要的配置信息，如可用模型列表
func (h *SystemHandler) GetConfig(c *gin.Context) {
	cfg := config.Get()

	data := gin.H{
		"version":       h.version,
		"models":        cfg.LLM.Models,
		"default_model": cfg.LLM.DefaultModel,
		"upload": gin.H{
			"max_size_mb":   cfg.Upload.MaxSize,
			"allow_types":   cfg.Upload.AllowTypes,
			"max_files":     20,
		},
	}

	response.Success(c, data)
}
