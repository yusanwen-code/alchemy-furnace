package system

import (
	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// GetConfig 获取系统配置
// GET /api/v1/system/config
// 返回前端需要的配置信息(可用模型清单与默认/合成模型)
func (cls *System) GetConfig(c *gin.Context) (response.Code, any, error) {
	cfg := configuration.Configuration

	return response.Ok, &ConfigResponse{
		Version:        cls.version,
		Models:         cfg.LLM.Models,
		DefaultModel:   cfg.LLM.DefaultModel,
		SynthesisModel: cfg.LLM.SynthesisModel,
	}, nil
}
