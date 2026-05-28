// Package middleware 错误恢复中间件
// 捕获 Gin 处理过程中的 panic，防止单个请求导致整个服务崩溃
// 统一处理 404 和 Method Not Allowed 等错误
package middleware

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/alchemy-furnace/server/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorRecovery 错误恢复中间件（替代 Gin 默认的 Recovery）
// 捕获 handler 中的 panic，记录错误信息，返回友好的错误响应
func ErrorRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录详细的错误信息和堆栈
				stack := debug.Stack()
				log.Printf("[炼丹炉] 捕获到 panic: %v\n%s", err, string(stack))

				if Logger != nil {
					Logger.Error("[炼丹炉] 捕获到 panic",
						zap.Any("error", err),
						zap.String("stack", string(stack)),
						zap.String("path", c.Request.URL.Path),
						zap.String("method", c.Request.Method),
					)
				}

				// 返回统一的 500 错误响应
				response.InternalError(c, fmt.Sprintf("服务器内部错误: %v", err))
				c.Abort()
			}
		}()

		c.Next()
	}
}

// CustomErrorHandler 自定义错误处理中间件
// 统一处理 404 Not Found 和 405 Method Not Allowed 等路由错误
func CustomErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有路由错误
		if c.Writer.Status() == 404 && c.Writer.Written() == false {
			response.NotFound(c, fmt.Sprintf("路径 %s %s 不存在，请检查 API 文档", c.Request.Method, c.Request.URL.Path))
			c.Abort()
			return
		}

		if c.Writer.Status() == 405 && c.Writer.Written() == false {
			response.ErrorWithStatus(c, 405, 405,
				fmt.Sprintf("方法 %s 不允许用于路径 %s", c.Request.Method, c.Request.URL.Path))
			c.Abort()
			return
		}
	}
}

// NoRouteHandler 处理所有未匹配的路由
func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.NotFound(c, "此路径不在「炼丹炉」的仙界地图中，请检查 API 文档")
	}
}

// NoMethodHandler 处理所有未匹配的 HTTP 方法
func NoMethodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.ErrorWithStatus(c, 405, 405, "此 HTTP 方法在此路径上不被允许")
	}
}
