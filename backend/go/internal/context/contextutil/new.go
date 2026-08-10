// Package contextutil 请求上下文工具: gin.Context → context.Context,携带 request_id
package contextutil

import (
	"context"

	"github.com/gin-gonic/gin"
)

type requestIDKey struct{}

// NewContextWithGin 从 gin.Context 派生携带 request_id 的 context(SSE 场景须用 c.Request.Context() 为基底以保留取消语义)
func NewContextWithGin(c *gin.Context) context.Context {
	return context.WithValue(c.Request.Context(), requestIDKey{}, c.GetString("X-Request-ID"))
}

// WithRequestID 在任意 context 上设置 request_id(子命令/后台任务用)
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID 取出 request_id,无则返回空串
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}
