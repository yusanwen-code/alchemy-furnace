// serve 子命令: 启动 HTTP 服务
// 迁移期共存策略: 新网关(web.Register)优先注册;未迁移域由 registerLegacyRoutes 兜底,
// 每迁移一个域即从 registerLegacyRoutes 注销对应路由组,直至旧代码全部删除(T036)
package command

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	olddao "github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/internal/logger"
	oldconfig "github.com/alchemy-furnace/server/pkg/config"
	"github.com/alchemy-furnace/server/server/http/gateway/web"
	"github.com/alchemy-furnace/server/server/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

// NewServeCommand serve 子命令
func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "serve",
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runServe(cmd)
		},
	}
}

// runServe 启动 HTTP 服务(根命令无参时亦走此入口)
func runServe(cmd *cobra.Command) {
	cfg := &configuration.Configuration

	gin.SetMode(cfg.Server.Mode)

	// 数据库连接(迁移/种子由子命令显式执行,启动不自动建表)
	if err := dao.InitDatabase(&cfg.Database); err != nil {
		log.Fatalf("[炼丹炉] 初始化数据库失败: %v", err)
	}
	defer dao.CloseDatabase()
	defer logger.Sync()

	// 迁移期桥接: 旧架构 service 仍读旧 dao 全局 DB 与旧 pkg/config,指向同一连接
	olddao.DB = dao.GetDB()
	if _, err := oldconfig.Load(); err != nil {
		log.Fatalf("[炼丹炉] 加载旧版配置失败: %v", err)
	}

	r := gin.New()

	// 中间件: request_id 最先注入,保证响应包络与日志均可取到
	r.Use(middleware.RequestID())
	r.Use(middleware.ErrorRecovery())
	r.Use(middleware.GinLogger())
	r.Use(middleware.CORS(cfg.Server.AllowOrigins))

	// 新网关路由(已迁移域)
	if err := web.Register(r); err != nil {
		log.Fatalf("[炼丹炉] 注册新网关路由失败: %v", err)
	}

	// 旧架构路由(未迁移域;逐域注销直至清空)
	registerLegacyRoutes(r)

	r.NoRoute(middleware.NoRouteHandler())
	r.NoMethod(middleware.NoMethodHandler())

	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("[炼丹炉] HTTP 服务已启动,监听地址: %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[炼丹炉] HTTP 服务启动失败: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[炼丹炉] 正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[炼丹炉] 服务强制关闭: %v", err)
	}
	log.Println("[炼丹炉] 服务已安全关闭,期待下次再会")
}
