//go:build wireinject
// +build wireinject

// Package handler 新网关 handler 装配入口(wire)
package handler

import (
	"github.com/alchemy-furnace/server/server/http/gateway/web/handler/agent"
	"github.com/alchemy-furnace/server/server/http/gateway/web/handler/pill"
	"github.com/alchemy-furnace/server/server/http/service"
	"github.com/google/wire"
)

// NewPill 金丹处理器装配
func NewPill() *pill.Pill {
	panic(wire.Build(
		service.PillService, pill.New,
	))
}

// NewAgent 道人处理器装配
func NewAgent() *agent.Agent {
	panic(wire.Build(
		service.AgentService, agent.New,
	))
}
