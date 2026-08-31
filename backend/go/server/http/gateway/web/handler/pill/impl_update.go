// 旧金丹更新入口（任务 5 起下线）
// 编辑丹方改 POST /api/v1/recipes/:id/revisions（不可变版本，expected_revision_id 竞争检查）
// （plan 任务 5 行 430：旧 /pills 写入返回 410 pill.legacy_api_removed）。
package pill

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// UpdateRequest 旧更新金丹请求契约（保留以文档化旧客户端载荷；接口已下线）
type UpdateRequest struct {
	Name        *string        `json:"name" binding:"omitempty,max=100"`
	Description *string        `json:"description"`
	SkillSchema map[string]any `json:"skill_schema"`
	Tags        []any          `json:"tags"`
	Author      *string        `json:"author" binding:"omitempty,max=100"`
	Version     *string        `json:"version" binding:"omitempty,max=20"`
}

// Update 旧金丹更新已下线
// PUT /api/v1/pills/:uuid → 410
func (cls *Pill) Update(c *gin.Context) (response.Code, any, error) {
	return 0, nil, errors.ErrorGone("pill.legacy_api_removed",
		"金丹消耗品重构：旧金丹管理接口已下线，编辑请使用 /api/v1/recipes/:id/revisions")
}
