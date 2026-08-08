// Package model 预置供应商模板常量
// 模板是「可用能力」契约的一部分，由后端单一数据源提供（GET /api/v1/providers/templates）
// 选中模板后前端自动填充 display_name/protocol/base_url，用户仅需补填 API Key
package model

// ProviderTemplate 预置供应商模板（静态常量，不入库）
type ProviderTemplate struct {
	ID              string   `json:"id"`               // 模板标识（如 openai / deepseek / ollama）
	DisplayName     string   `json:"display_name"`     // 显示名
	Protocol        string   `json:"protocol"`         // 协议类型（当前均为 openai-compatible）
	DefaultBaseURL  string   `json:"default_base_url"` // 预填 base_url（用户可修改后保存）
	RequiresAPIKey  bool     `json:"requires_api_key"` // 是否需要 API Key（Ollama 本地服务为 false）
	SuggestedModels []string `json:"suggested_models"` // 建议模型名列表
	Group           string   `json:"group"`            // 分组：domestic / international / local
}

// ProviderTemplates 首版预置模板清单（8 项）
// base_url 与建议模型名以 specs/003-provider-protocol-model-hub/data-model.md 表格为准
var ProviderTemplates = []ProviderTemplate{
	{
		ID:              "openai",
		DisplayName:     "OpenAI",
		Protocol:        "openai-compatible",
		DefaultBaseURL:  "https://api.openai.com/v1",
		RequiresAPIKey:  true,
		SuggestedModels: []string{"gpt-4o", "gpt-4o-mini"},
		Group:           "international",
	},
	{
		ID:              "deepseek",
		DisplayName:     "DeepSeek",
		Protocol:        "openai-compatible",
		DefaultBaseURL:  "https://api.deepseek.com/v1",
		RequiresAPIKey:  true,
		SuggestedModels: []string{"deepseek-chat", "deepseek-reasoner"},
		Group:           "domestic",
	},
	{
		ID:              "dashscope",
		DisplayName:     "通义千问",
		Protocol:        "openai-compatible",
		DefaultBaseURL:  "https://dashscope.aliyuncs.com/compatible-mode/v1",
		RequiresAPIKey:  true,
		SuggestedModels: []string{"qwen-max", "qwen-plus", "qwen-turbo"},
		Group:           "domestic",
	},
	{
		ID:              "zhipu",
		DisplayName:     "智谱 GLM",
		Protocol:        "openai-compatible",
		DefaultBaseURL:  "https://open.bigmodel.cn/api/paas/v4",
		RequiresAPIKey:  true,
		SuggestedModels: []string{"glm-4-plus", "glm-4-air"},
		Group:           "domestic",
	},
	{
		ID:              "moonshot",
		DisplayName:     "Moonshot（Kimi）",
		Protocol:        "openai-compatible",
		DefaultBaseURL:  "https://api.moonshot.cn/v1",
		RequiresAPIKey:  true,
		SuggestedModels: []string{"moonshot-v1-8k", "moonshot-v1-32k"},
		Group:           "domestic",
	},
	{
		ID:              "baichuan",
		DisplayName:     "百川智能",
		Protocol:        "openai-compatible",
		DefaultBaseURL:  "https://api.baichuan-ai.com/v1",
		RequiresAPIKey:  true,
		SuggestedModels: []string{"Baichuan4"},
		Group:           "domestic",
	},
	{
		ID:              "qianfan",
		DisplayName:     "文心一言（千帆）",
		Protocol:        "openai-compatible",
		DefaultBaseURL:  "https://qianfan.baidubce.com/v2",
		RequiresAPIKey:  true,
		SuggestedModels: []string{"ernie-4.0-8k", "ernie-3.5-8k"},
		Group:           "domestic",
	},
	{
		ID:              "ollama",
		DisplayName:     "Ollama（本地）",
		Protocol:        "openai-compatible",
		DefaultBaseURL:  "http://localhost:11434/v1",
		RequiresAPIKey:  false,
		SuggestedModels: []string{"llama3.1", "qwen2.5"},
		Group:           "local",
	},
}
