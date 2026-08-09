// 旧架构路由注册(迁移期临时共存)
// ⚠️ 本文件随域迁移逐组删除;US2 完成后整文件删除(T036)
package command

import (
	"github.com/gin-gonic/gin"
)

// registerLegacyRoutes 注册尚未迁移域的旧路由
// 已注销: pills/agents(US1)/system/trial/providers/models(US2)/chat(US2) 迁入新网关 web.Register
// 全部域迁移完成,旧路由注册为空;待 T036 删除本文件与旧 handler/service/dao/pkg 目录
func registerLegacyRoutes(r *gin.Engine) {
	// no-op: 所有域已迁入新网关 web.Register
}

