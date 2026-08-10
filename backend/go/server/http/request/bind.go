// Package request 统一请求绑定与校验翻译
package request

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/gin-gonic/gin"
)

// ShouldBindJSON 绑定 JSON 请求体;失败返回 ErrorTypeInvalidRequest(400)
// 消息格式沿用旧版「请求参数错误: <校验细节>」,前端展示逻辑零改动
func ShouldBindJSON(c *gin.Context, dst any) errors.Error {
	if err := c.ShouldBindJSON(dst); err != nil {
		return errors.New(errors.ErrorTypeInvalidRequest, "SHREQ_Q.SBJ_ON.0001", "请求参数错误: %s", err.Error())
	}
	return nil
}
