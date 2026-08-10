// Package service wire 装配集(对齐 Luna-CY 模板 server/http/service)
// 每个域一个 wire.NewSet:dao 实现绑定接口 + service 实现绑定接口
package service

import (
	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/alchemy-furnace/server/internal/dao"
	idao "github.com/alchemy-furnace/server/internal/interface/dao"
	iservice "github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/internal/service/agent_service"
	"github.com/alchemy-furnace/server/internal/service/chat_service"
	"github.com/alchemy-furnace/server/internal/service/credential"
	"github.com/alchemy-furnace/server/internal/service/language_pattern_service"
	"github.com/alchemy-furnace/server/internal/service/model_service"
	"github.com/alchemy-furnace/server/internal/service/pill_service"
	"github.com/alchemy-furnace/server/internal/service/provider_service"
	"github.com/alchemy-furnace/server/internal/service/trial_service"
	"github.com/alchemy-furnace/server/internal/synthesis"
	"github.com/google/wire"
)

// PillService 金丹域装配集
var PillService = wire.NewSet(
	dao.NewPillDao, wire.Bind(new(idao.Pill), new(*dao.PillDao)),
	pill_service.New, wire.Bind(new(iservice.Pill), new(*pill_service.Pill)),
)

// AgentService 道人域装配集(依赖金丹 DAO 与模型查询 DAO)
var AgentService = wire.NewSet(
	dao.NewAgentDao, wire.Bind(new(idao.Agent), new(*dao.AgentDao)),
	dao.NewPillDao, wire.Bind(new(idao.Pill), new(*dao.PillDao)),
	dao.NewModelDao, wire.Bind(new(idao.Model), new(*dao.ModelDao)),
	agent_service.New, wire.Bind(new(iservice.Agent), new(*agent_service.Agent)),
)

// NewSynthesisClient 合成引擎客户端 provider(从全局配置取 Python 引擎 BaseURL)
func NewSynthesisClient() synthesis.Client {
	return synthesis.New(configuration.Configuration.PythonEngine.BaseURL)
}

// NewEngineBaseURL 语言引擎 BaseURL provider(对话流式接口直连 Python 引擎)
func NewEngineBaseURL() string {
	return configuration.Configuration.PythonEngine.BaseURL
}

// ChatService 对话域装配集(依赖道人 DAO + 合成客户端 + 凭证解析器 + 语言模式服务)
var ChatService = wire.NewSet(
	dao.NewChatDao, wire.Bind(new(idao.Chat), new(*dao.ChatDao)),
	dao.NewAgentDao, wire.Bind(new(idao.Agent), new(*dao.AgentDao)),
	NewSynthesisClient,
	credential.NewResolver, wire.Bind(new(credential.Resolver), new(*credential.ModelResolver)),
	NewEngineBaseURL,
	language_pattern_service.New, wire.Bind(new(iservice.LanguagePatternProvider), new(*language_pattern_service.LanguagePatternService)),
	chat_service.New, wire.Bind(new(iservice.Chat), new(*chat_service.Chat)),
)

// TrialService 试丹域装配集(依赖金丹 DAO + 合成客户端 + 凭证解析器)
var TrialService = wire.NewSet(
	dao.NewPillDao, wire.Bind(new(idao.Pill), new(*dao.PillDao)),
	NewSynthesisClient,
	credential.NewResolver, wire.Bind(new(credential.Resolver), new(*credential.ModelResolver)),
	trial_service.New, wire.Bind(new(iservice.Trial), new(*trial_service.Trial)),
)

// ProviderService 供应商域装配集(连接测试回退取首个启用模型,故依赖模型 DAO)
var ProviderService = wire.NewSet(
	dao.NewProviderDao, wire.Bind(new(idao.Provider), new(*dao.ProviderDao)),
	dao.NewModelDao, wire.Bind(new(idao.Model), new(*dao.ModelDao)),
	provider_service.New, wire.Bind(new(iservice.Provider), new(*provider_service.ProviderService)),
)

// ModelService 模型域装配集(凭证解析链需供应商 DAO)
var ModelService = wire.NewSet(
	dao.NewModelDao, wire.Bind(new(idao.Model), new(*dao.ModelDao)),
	dao.NewProviderDao, wire.Bind(new(idao.Provider), new(*dao.ProviderDao)),
	model_service.New, wire.Bind(new(iservice.Model), new(*model_service.ModelService)),
)
