// 旧服用列表读入口（任务 8 旧入口审计后下线）
// 金丹消耗品重构：迁移后旧 agent_pills 表保留（受控回滚），该路径会把已服用金丹
// 当作活跃绑定展示（移除能力后旧行仍在，与 effects 状态矛盾），且无任何调用方。
// 按「所有旧入口切换或关闭」恒 410 pill.legacy_api_removed；
// 能力展示改走 GET /api/v1/agents/:uuid/effects（已吸收能力快照）。
package agent

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// ListPills 旧服用列表已下线
// GET /api/v1/agents/:uuid/pills → 410
func (cls *Agent) ListPills(c *gin.Context) (response.Code, any, error) {
	return 0, nil, errors.ErrorGone("pill.legacy_api_removed",
		"金丹消耗品重构：旧服用列表接口已下线，已吸收能力请使用 /api/v1/agents/:id/effects")
}
