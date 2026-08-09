// Package web 新网关路由注册(对齐 Luna-CY 模板 server/http/gateway/web)
// 迁移期: 逐域在此注册并从 cmd/main/command/legacy_routes.go 注销旧路由
package web

import (
	"github.com/alchemy-furnace/server/server/http/gateway/web/handler"
	"github.com/alchemy-furnace/server/server/http/router"
	"github.com/gin-gonic/gin"
)

// Register 注册新网关全部路由(已迁移域: pill + agent)
func Register(r *gin.Engine) error {
	v1 := r.Group("/api/v1")

	pillHandler := handler.NewPill()
	agentHandler := handler.NewAgent()

	// 金丹管理(UUID 对外标识)
	pills := v1.Group("/pills")
	{
		pills.GET("", router.WrapperPage(pillHandler.List))
		pills.POST("", router.Wrapper(pillHandler.Create))
		pills.GET("/:uuid", router.Wrapper(pillHandler.Get))
		pills.PUT("/:uuid", router.Wrapper(pillHandler.Update))
		pills.DELETE("/:uuid", router.Wrapper(pillHandler.Delete))
	}

	// 道人管理(UUID 对外标识;服用记录路径参数 :pill_uuid)
	agents := v1.Group("/agents")
	{
		agents.GET("", router.WrapperPage(agentHandler.List))
		agents.POST("", router.Wrapper(agentHandler.Create))
		agents.GET("/:uuid", router.Wrapper(agentHandler.Get))
		agents.PUT("/:uuid", router.Wrapper(agentHandler.Update))
		agents.DELETE("/:uuid", router.Wrapper(agentHandler.Delete))
		agents.POST("/:uuid/pills", router.Wrapper(agentHandler.BindPill))
		agents.PUT("/:uuid/pills/:pill_uuid", router.Wrapper(agentHandler.UpdateAgentPill))
		agents.DELETE("/:uuid/pills/:pill_uuid", router.Wrapper(agentHandler.UnbindPill))
		agents.GET("/:uuid/pills", router.Wrapper(agentHandler.ListPills))
	}

	return nil
}
