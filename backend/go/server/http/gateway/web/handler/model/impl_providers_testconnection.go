package model

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// TestConnectionRequest 供应商连接测试请求(model 可选,缺省用该供应商下第一个启用模型)
type TestConnectionRequest struct {
	Model string `json:"model"`
}

// TestConnection 供应商连接测试
// 以供应商凭证发起一次最小 LLM 调用(max_tokens=1),返回 {success, latency_ms, error}
// POST /api/v1/providers/:uuid/test-connection
func (cls *Model) TestConnection(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	// body 可选:允许空 body,绑定失败(EOF)不视为错误
	var body TestConnectionRequest
	_ = c.ShouldBindJSON(&body)

	result, serr := cls.provider.TestConnection(contextutil.NewContextWithGin(c), uid, body.Model)
	if serr != nil {
		return 0, nil, serr
	}
	return response.Ok, result, nil
}
