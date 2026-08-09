// Package service wire 装配集(对齐 Luna-CY 模板 server/http/service)
// 每个域一个 wire.NewSet:dao 实现绑定接口 + service 实现绑定接口
package service

import (
	"github.com/alchemy-furnace/server/internal/dao"
	idao "github.com/alchemy-furnace/server/internal/interface/dao"
	iservice "github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/internal/service/agent_service"
	"github.com/alchemy-furnace/server/internal/service/pill_service"
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
