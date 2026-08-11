// 演示模式响应头中间件(007-demo-mode)
// DEMO_MODE=true 时所有响应注入 X-Alchemy-Mode: demo,便于前端 SSR/客户端识别
package middleware

import (
	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/gin-gonic/gin"
)

// ModeHeader 演示模式响应头: 仅在 IsDemo() 时注入 X-Alchemy-Mode=demo
// 真实模式不发送该头(老客户端不会因新增头部而出现解析错误)
func ModeHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		if configuration.IsDemo() {
			c.Header("X-Alchemy-Mode", "demo")
		}
		c.Next()
	}
}
