// Package engine 提供调用 Python 语言引擎的公共错误类型与错误映射。
// 对话服务、试炼服务共用，避免循环依赖。
package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// EngineError 语言引擎返回的非 200 错误，保留状态码供错误映射使用。
type EngineError struct {
	Op         string // 操作描述
	StatusCode int    // HTTP 状态码
	Body       string // 引擎响应体
}

func (e *EngineError) Error() string {
	return fmt.Sprintf("%s返回错误: status=%d, body=%s", e.Op, e.StatusCode, e.Body)
}

// MapEngineError 将调用语言引擎的错误映射为可读中文描述（websocket-protocol.md 错误语义）。
//   - 401/403: 模型凭证无效
//   - 网络超时/408/504: 引擎响应超时
//   - 5xx: 引擎服务异常
//   - 连接失败: 无法连接语言引擎
func MapEngineError(err error) string {
	var engineErr *EngineError
	if errors.As(err, &engineErr) {
		switch {
		case engineErr.StatusCode == http.StatusUnauthorized || engineErr.StatusCode == http.StatusForbidden:
			return "模型凭证无效，请检查模型管理中的 API Key"
		case engineErr.StatusCode == http.StatusRequestTimeout || engineErr.StatusCode == http.StatusGatewayTimeout:
			return "语言引擎响应超时，请稍后重试"
		case engineErr.StatusCode >= 500:
			return "语言引擎服务异常，请稍后重试"
		default:
			return fmt.Sprintf("语言引擎请求失败（状态码 %d）", engineErr.StatusCode)
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "语言引擎响应超时，请稍后重试"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "语言引擎响应超时，请稍后重试"
	}
	return "无法连接语言引擎，请检查服务是否启动"
}
