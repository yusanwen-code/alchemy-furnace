package service

import (
	"context"

	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

// PillCompositionItem 完整服丹编排项(对外金丹 UUID + 权重)
// 供 ReplacePillComposition 签名(该入口任务 3 起 410 下线,类型仅保留契约兼容)
type PillCompositionItem struct {
	PillUUID uuid.UUID
	Weight   float64
}

// Agent 道人业务逻辑接口
type Agent interface {
	// ListAgents 分页查询道人列表(status 为空不过滤)
	ListAgents(ctx context.Context, page int, size int, status string) (int64, []*model.DaoAgent, errors.Error)

	// GetAgentDetailByUUID 按 UUID 获取道人详情(含服用记录与语言模式缓存)
	GetAgentDetailByUUID(ctx context.Context, uid uuid.UUID) (*model.DaoAgent, errors.Error)

	// CreateAgent 创建道人;model_name 非空时校验其引用已启用模型配置
	// proactivity 为 nil 时取默认值 50;合法区间 0-100,越界返回 InvalidRequest
	CreateAgent(ctx context.Context, name string, avatar string, personality string, modelName string, proactivity *int) (*model.DaoAgent, errors.Error)

	// UpdateAgent 按 UUID 部分更新道人(nil 字段不更新);性格变化时失效语言模式缓存
	// proactivity 合法区间 0-100;nil=不更新;memoryEnabled nil=不更新
	UpdateAgent(ctx context.Context, uid uuid.UUID, name *string, avatar *string, personality *string, modelName *string, status *string, proactivity *int, memoryEnabled *bool) (*model.DaoAgent, errors.Error)

	// DeleteAgent 按 UUID 删除道人
	DeleteAgent(ctx context.Context, uid uuid.UUID) errors.Error

	// BindPill 道人服用金丹实例(第二参数为金丹实例 UUID,非丹方/金丹 UUID;任务 3)
	// 经库存幂等消费: available→consumed_by_agent + 生成能力快照(单事务);
	// 实例不可用/重复服用返回 ErrorTypeConflict(pill.not_available);
	// 实例不存在/未知道人返回 ErrorTypeRecordNotFound
	BindPill(ctx context.Context, agentUID uuid.UUID, pillUID uuid.UUID, weight float64, sortOrder int) errors.Error

	// UpdateAgentPill 调整已吸收能力权重/顺序(第二参数为金丹实例 UUID;任务 3)
	// weight/sort_order 为 nil 时不更新对应字段(均为 nil 时仅校验存在);
	// 无活跃能力(未吸收/已移除)返回 ErrorTypeRecordNotFound
	UpdateAgentPill(ctx context.Context, agentUID uuid.UUID, pillUID uuid.UUID, weight *float64, sortOrder *int) errors.Error

	// UnbindPill 移除道人的已吸收能力(第二参数为金丹实例 UUID;任务 3)
	// 软删(removed_at 保留历史);原实例保持 consumed_by_agent 不返还库存;
	// 无活跃能力返回 ErrorTypeRecordNotFound
	UnbindPill(ctx context.Context, agentUID uuid.UUID, pillUID uuid.UUID) errors.Error

	// ListAgentPills 道人已服用金丹列表(按服用顺序)
	ListAgentPills(ctx context.Context, agentUID uuid.UUID) ([]*model.ElixirPill, errors.Error)

	// ReplacePillComposition 已下线(任务 3): 完整服丹编排绕过库存,不再保留。
	// 任意输入一律返回 ErrorTypeGone(pill.legacy_api_removed,→410),不产生任何写入;
	// 服用改走 BindPill、移除改走 UnbindPill
	ReplacePillComposition(ctx context.Context, agentUID uuid.UUID, items []PillCompositionItem) (*model.DaoAgent, errors.Error)

	// ListEffects 道人活跃能力列表（按 sort_order 升序；任务 5，effect UUID 语义）
	// 同时返回 effects_revision（乐观锁版本），前端 PUT 全量编排需携带
	ListEffects(ctx context.Context, agentUID uuid.UUID) ([]*EffectWithSource, int, errors.Error)

	// UpdateEffects 全量编排（任务 5）：提交集必须等于活跃集，
	// 缺少/重复/外部道人 effect → 409 agent.effects_conflict；expectedEffectsRevision
	// 乐观锁过期同 409；成功返回更新后的道人（含新 EffectsRevision），并失效语言模式缓存
	UpdateEffects(ctx context.Context, agentUID uuid.UUID, expectedEffectsRevision int, items []EffectUpdateItem) (*model.DaoAgent, errors.Error)

	// RemoveEffect 显式移除能力（按能力 UUID，任务 5）：软删 + EffectsRevision++ + 缓存失效；
	// 不存在/已移除/跨道人 → 404；原实例保持 consumed_by_agent 不返还
	RemoveEffect(ctx context.Context, agentUID uuid.UUID, effectUUID uuid.UUID) errors.Error
}

// EffectUpdateItem 全量编排条目（能力 UUID + 权重 + 顺序；任务 5）
type EffectUpdateItem struct {
	EffectID  uuid.UUID
	Weight    float64
	SortOrder int
}

// EffectWithSource 能力快照 + 来源实例/版本对外标识（任务 5 列表输出；
// 模型上的 UUID 字段均为 json:"-"，对外展示需显式携带）
type EffectWithSource struct {
	Effect       model.AgentPillEffect
	ItemUUID     uuid.UUID
	RevisionUUID uuid.UUID
}
