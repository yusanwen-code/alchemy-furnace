// Package middleware 日志中间件
// 使用 zap 记录每个 HTTP 请求的详细信息，包括请求方法、路径、耗时、状态码等
// 帮助追踪"炼丹"过程中的问题
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 中间件使用的 zap logger 实例
var Logger *zap.Logger

// InitLogger 初始化 zap logger
// 根据环境选择开发模式（彩色输出）或生产模式（JSON格式）
func InitLogger(mode string) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if mode == "release" {
		// 生产模式：JSON 格式，性能优化
		logger, err = zap.NewProduction()
	} else {
		// 开发模式：彩色控制台输出
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		return nil, err
	}

	Logger = logger
	return logger, nil
}

// GinLogger Gin HTTP 请求日志中间件
// 记录每个请求的：方法、路径、客户端IP、耗时、状态码、响应大小
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()

		// 处理请求
		c.Next()

		// 计算耗时
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		responseSize := c.Writer.Size()

		// 构建日志字段
		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("duration", duration),
			zap.Int("size", responseSize),
		}

		// 如果有错误，记录错误信息
		if len(c.Errors) > 0 {
			fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
		}

		// 根据状态码选择日志级别
		logMsg := "[炼丹炉] HTTP请求"
		switch {
		case statusCode >= 500:
			if Logger != nil {
				Logger.Error(logMsg+" - 服务器错误", fields...)
			}
		case statusCode >= 400:
			if Logger != nil {
				Logger.Warn(logMsg+" - 客户端错误", fields...)
			}
		default:
			if Logger != nil {
				Logger.Info(logMsg, fields...)
			}
		}

		// 同时输出到控制台（便于 Docker 日志查看）
		if statusCode >= 400 {
			// 使用标准日志输出，保持在非开发模式下也能看到请求
		}
	}
}

// SyncLogger 同步 logger 缓冲区，在程序退出前调用
func SyncLogger() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
