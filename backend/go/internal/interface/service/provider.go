package service

import (
	"context"

	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

// Provider 供应商配置业务逻辑接口(对外标识一律为 UUID;错误为 errors.Error 类型化错误)
type Provider interface {
	// ListProviders 分页查询供应商列表(enabled=nil 不筛选),返回富视图(含模型数量与掩码 api_key)
	ListProviders(ctx context.Context, page, size int, enabled *bool) (int64, []*model.ProviderView, errors.Error)

	// GetProviderByUUID 按 UUID 获取供应商详情(掩码形式)
	GetProviderByUUID(ctx context.Context, uid uuid.UUID) (*model.ProviderView, errors.Error)

	// CreateProvider 创建供应商;apiKey 明文加密后存储;未配置 MODEL_KEY_SECRET 时返回 ErrorInvalidRequest
	CreateProvider(ctx context.Context, name, displayName, protocol, baseURL, apiKey string, isEnabled bool, sortOrder int, remark string) (*model.ProviderView, errors.Error)

	// UpdateProvider 按 UUID 部分更新供应商(nil 字段不更新);apiKey: nil=不改,空串=清除,非空=重新加密
	UpdateProvider(ctx context.Context, uid uuid.UUID, name, displayName, protocol, baseURL, apiKey *string, isEnabled *bool, sortOrder *int, remark *string) (*model.ProviderView, errors.Error)

	// DeleteProvider 按 UUID 删除供应商;下有模型时返回 ErrorConflictWithData(409, model_count)
	DeleteProvider(ctx context.Context, uid uuid.UUID) errors.Error

	// TestConnection 以供应商凭证发起最小 LLM 调用(max_tokens=1)测量延迟;modelName 空时回退首个启用模型
	TestConnection(ctx context.Context, uid uuid.UUID, modelName string) (*model.TestConnectionResult, errors.Error)

	// Templates 返回预置供应商模板清单(静态常量)
	Templates() []model.ProviderTemplate
}
