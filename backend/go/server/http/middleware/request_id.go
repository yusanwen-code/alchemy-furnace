// 请求链路 ID 中间件: X-Request-ID 入站透传,缺失则生成 UUID
// 必须在所有中间件之前注册,保证响应包络与日志均能取到
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID 注入请求链路 ID 到 gin.Context("X-Request-ID")与响应头
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("X-Request-ID", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}
