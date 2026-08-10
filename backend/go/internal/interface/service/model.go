package service

import (
	"context"

	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/service/credential"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

// Model 模型配置业务逻辑接口(对外标识一律为 UUID;错误为 errors.Error 类型化错误)
type Model interface {
	// ListModelsByProvider 分页查询指定供应商下的模型列表(含引用计数富视图)
	ListModelsByProvider(ctx context.Context, providerUID uuid.UUID, page, size int) ([]*model.ModelView, errors.Error)

	// GetModelByUUID 按 UUID 获取模型详情(含引用计数)
	GetModelByUUID(ctx context.Context, uid uuid.UUID) (*model.ModelView, errors.Error)

	// CreateModel 在指定供应商下创建模型;供应商停用返回 ErrorInvalidRequest;is_default/is_synthesis 事务内清除其他
	CreateModel(ctx context.Context, providerUID uuid.UUID, name, displayName string, temperature float64, maxTokens int, isEnabled, isDefault, isSynthesis bool, sortOrder int) (*model.ModelView, errors.Error)

	// UpdateModel 按 UUID 部分更新模型(nil 字段不更新);is_default/is_synthesis 事务内清除其他
	UpdateModel(ctx context.Context, uid uuid.UUID, name, displayName *string, temperature *float64, maxTokens *int, isEnabled, isDefault, isSynthesis *bool, sortOrder *int) (*model.ModelView, errors.Error)

	// DeleteModel 按 UUID 删除模型;被道人引用时返回 ErrorConflictWithData(409, referenced_by)
	DeleteModel(ctx context.Context, uid uuid.UUID) errors.Error

	// Options 已启用供应商下的已启用模型精简列表(道人表单下拉)
	Options(ctx context.Context) ([]model.LLMModelOption, errors.Error)

	// ResolveCredentials 解析对话模型调用凭证(两级: model -> provider -> 解密);name 空时取默认模型
	ResolveCredentials(ctx context.Context, name string) (*credential.ModelCredentials, errors.Error)

	// ResolveDefaultCredentials 解析默认模型凭证;无默认模型时回退环境变量配置
	ResolveDefaultCredentials(ctx context.Context) (*credential.ModelCredentials, errors.Error)

	// ResolveSynthesisCredentials 解析合成专用模型凭证;无则回退默认模型
	ResolveSynthesisCredentials(ctx context.Context) (*credential.ModelCredentials, errors.Error)
}
