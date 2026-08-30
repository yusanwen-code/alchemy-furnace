// cmd/desktop/app.go - Wails 绑定对象(任务 11 扩展)
// 注意: webview 重定向到 http origin 后 Wails Bind 不可达,仅作兜底
// 版本/更新等正式接口走 HTTP 端点(/api/v1/version 全模式;update 三端点仅 desktop,Guard 保护)
// 生命周期(OnStartup/OnBeforeClose/OnShutdown)由 lifecycle.go 装配,App 不再持有 ctx
package desktop

import (
	"github.com/alchemy-furnace/server/internal/buildinfo"
)

// App 桌面绑定对象(Wails 自动扫描导出方法,经 JS 桥调用)
type App struct{}

// NewApp 构造应用对象
func NewApp() *App { return &App{} }

// GetVersion 版本信息(经 HTTP /api/v1/version 也可取,任务 11 落 HTTP)
func (a *App) GetVersion() map[string]string {
	return map[string]string{
		"version":   buildinfo.Version,
		"commit":    buildinfo.Commit,
		"buildDate": buildinfo.BuildDate,
	}
}
