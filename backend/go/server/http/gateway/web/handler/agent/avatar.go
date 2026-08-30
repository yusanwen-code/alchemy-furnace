package agent

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/util/avatar"
)

// validateAvatar 校验头像字段(创建与更新共用)
// 契约与错误文案统一由 internal/util/avatar 提供;此处只做错误码包装。
// 错误消息只描述规则,绝不携带头像值(完整 data URI 不进响应/日志)。
func validateAvatar(value string) errors.Error {
	if err := avatar.Validate(value); err != nil {
		return errors.New(errors.ErrorTypeInvalidRequest, "handler.agent.avatar_validate", err.Error())
	}
	return nil
}
