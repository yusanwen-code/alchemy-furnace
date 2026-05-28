// Package middleware 提供「炼丹炉」的 HTTP 中间件
// CORS 中间件：允许前端跨域访问，支持本地开发和生产环境部署
package middleware

import (
	"github.com/alchemy-furnace/server/pkg/config"
	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件
// 允许前端从不同域名访问 API，支持预检请求(Preflight)
// 在开发环境允许所有来源，生产环境可通过配置限制
func CORS() gin.HandlerFunc {
	cfg := config.Get()
	allowOrigins := cfg.Server.AllowOrigins
	if allowOrigins == "" {
		allowOrigins = "*"
	}

	return func(c *gin.Context) {
		// 设置允许的请求来源
		origin := c.Request.Header.Get("Origin")
		if allowOrigins == "*" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			// 检查来源是否在允许列表中
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		// 设置允许的 HTTP 方法
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")

		// 设置允许的请求头
		c.Writer.Header().Set("Access-Control-Allow-Headers",
			"Origin, Content-Type, Accept, Authorization, X-Requested-With")

		// 允许携带凭证（如 Cookie）
		if allowOrigins != "*" {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// 设置预检请求的缓存时间（秒）
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// 暴露给前端的响应头
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		// 处理 OPTIONS 预检请求，直接返回 204
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
