// Package model 视图包装类型(新架构): 服务层返回的富视图,内嵌 GORM 模型并附加展示用计算字段
// handler 层据此转换为对外 DTO(UUID 字符串化,内部自增 ID 不外泄)
package model

// ProviderView 供应商视图: 内嵌供应商配置 + 掩码后的 api_key + 关联模型数量
// api_key 永不明文出现在任何输出中,APIKeyMasked 仅供 handler 转换为响应
type ProviderView struct {
	*LLMProvider
	APIKeyMasked string // 解密后掩码(如 sk-****wxyz),未配置时为空
	HasAPIKey    bool   // 是否已配置 api_key
	ModelCount   int64  // 该供应商下的模型数量
}

// ModelView 模型视图: 内嵌模型配置 + 被道人引用的数量
type ModelView struct {
	*LLMModel
	ReferencedBy int64 // 引用该模型名(model_name)的道人数量
}
