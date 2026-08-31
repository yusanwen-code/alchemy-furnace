package agent

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Get 道人详情(含已吸收能力快照与语言模式缓存)
// 任务 8 旧入口审计: 不再返回遗留 agent_pills 绑定(旧表仅保留供回滚),
// 能力数据走 GET /api/v1/agents/:uuid/effects
// GET /api/v1/agents/:uuid
func (cls *Agent) Get(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	agent, serr := cls.agent.GetAgentDetailByUUID(contextutil.NewContextWithGin(c), uid)
	if serr != nil {
		return response.NotFound, nil, serr
	}
	return response.Ok, toDetailResponse(agent), nil
}
