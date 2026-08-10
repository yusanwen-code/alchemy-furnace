// serve 子命令: 启动 HTTP 服务
// 启动 HTTP 服务(根命令无参时亦走此入口),由 web.Register 集中注册所有路由
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

	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/internal/logger"
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

	r := gin.New()

	// 中间件: request_id 最先注入,保证响应包络与日志均可取到
	r.Use(middleware.RequestID())
	r.Use(middleware.ErrorRecovery())
	r.Use(middleware.GinLogger())
	r.Use(middleware.CORS(cfg.Server.AllowOrigins))

	// 新网关路由(集中注册)
	if err := web.Register(r); err != nil {
		log.Fatalf("[炼丹炉] 注册新网关路由失败: %v", err)
	}

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
