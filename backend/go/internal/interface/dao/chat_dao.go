// Package dao 数据访问接口定义(对齐 Luna-CY 模板 internal/interface/dao)
package dao

import (
	"context"

	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

// Chat 对话域数据访问接口(会话/消息)
type Chat interface {
	// TakeSessionByUUID 按对外 UUID 查询会话(预加载道人),不存在返回 ErrorTypeRecordNotFound
	TakeSessionByUUID(ctx context.Context, uid uuid.UUID) (*model.ChatSession, errors.Error)

	// FindSessions 分页查询会话列表(agentID>0 时按道人过滤),按更新时间倒序
	FindSessions(ctx context.Context, agentID uint, page int, size int) (int64, []*model.ChatSession, errors.Error)

	// SaveSession 新建会话
	SaveSession(ctx context.Context, session *model.ChatSession) errors.Error

	// UpdateSession 按字段 map 部分更新会话(如标题)
	UpdateSession(ctx context.Context, session *model.ChatSession, updates map[string]any) errors.Error

	// DeleteSession 删除会话(消息由 FK CASCADE 清理)
	DeleteSession(ctx context.Context, session *model.ChatSession) errors.Error

	// FindMessages 分页查询会话消息(按时间正序)
	FindMessages(ctx context.Context, sessionID uint, page int, size int) (int64, []*model.ChatMessage, errors.Error)

	// SaveMessage 写入消息并刷新所属会话 updated_at
	SaveMessage(ctx context.Context, message *model.ChatMessage) errors.Error
}
