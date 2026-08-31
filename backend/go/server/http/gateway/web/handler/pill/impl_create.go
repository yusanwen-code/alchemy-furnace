// 旧金丹创建入口（任务 5 起下线）
// 金丹消耗品重构：旧 /pills 写入全部关闭，创建丹方改 POST /api/v1/recipes
// （plan 任务 5 行 430：旧 /pills 写入返回 410 pill.legacy_api_removed）。
package pill

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// CreateRequest 旧创建金丹请求契约（保留以文档化旧客户端载荷；接口已下线）
type CreateRequest struct {
	Name        string         `json:"name" binding:"required,max=100"`
	Description string         `json:"description"`
	SkillSchema model.JSONMap  `json:"skill_schema" binding:"required"`
	Tags        model.JSONList `json:"tags"`
	Author      string         `json:"author" binding:"max=100"`
	Version     string         `json:"version" binding:"max=20"`
}

// Create 旧金丹创建已下线
// POST /api/v1/pills → 410
func (cls *Pill) Create(c *gin.Context) (response.Code, any, error) {
	return 0, nil, errors.ErrorGone("pill.legacy_api_removed",
		"金丹消耗品重构：旧金丹管理接口已下线，请使用 /api/v1/recipes")
}
