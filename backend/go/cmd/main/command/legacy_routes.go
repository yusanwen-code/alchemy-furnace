// 旧架构路由注册(迁移期临时共存)
// ⚠️ 本文件随域迁移逐组删除;US2 完成后整文件删除(T036)
package command

import (
	"github.com/alchemy-furnace/server/handler"
	"github.com/gin-gonic/gin"
)

// registerLegacyRoutes 注册尚未迁移域的旧路由
// 已注销: pills/agents(US1 迁入新网关 web.Register)
func registerLegacyRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	chatHandler := handler.NewChatHandler()
	systemHandler := handler.NewSystemHandler()
	trialHandler := handler.NewTrialHandler()
	modelHandler := handler.NewModelHandler()

	// 对话管理
	chatGroup := v1.Group("/chat")
	{
		chatGroup.POST("/sessions", chatHandler.CreateSession)
		chatGroup.GET("/sessions", chatHandler.ListSessions)
		chatGroup.GET("/sessions/:id/messages", chatHandler.GetMessages)
		chatGroup.POST("/sse/:session_id", chatHandler.SSEChat)
	}

	// 试丹
	trialGroup := v1.Group("/trial")
	{
		trialGroup.POST("/synthesis", trialHandler.Synthesize)
		trialGroup.POST("/chat", trialHandler.Chat)
	}

	// 供应商与模型管理(/templates 与 /options 必须先于 :id 注册)
	providers := v1.Group("/providers")
	{
		providers.GET("/templates", modelHandler.ListTemplates)
		providers.GET("", modelHandler.ListProviders)
		providers.POST("", modelHandler.CreateProvider)
		providers.PUT("/:id", modelHandler.UpdateProvider)
		providers.DELETE("/:id", modelHandler.DeleteProvider)
		providers.POST("/:id/test-connection", modelHandler.TestProviderConnection)
		providers.GET("/:id/models", modelHandler.ListProviderModels)
		providers.POST("/:id/models", modelHandler.CreateProviderModel)
	}

	models := v1.Group("/models")
	{
		models.GET("/options", modelHandler.ListOptions)
		models.PUT("/:id", modelHandler.UpdateModel)
		models.DELETE("/:id", modelHandler.DeleteModel)
	}

	// 系统接口
	sys := v1.Group("/system")
	{
		sys.GET("/health", systemHandler.HealthCheck)
		sys.GET("/config", systemHandler.GetConfig)
	}
}
