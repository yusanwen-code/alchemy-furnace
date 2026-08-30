// lifecycle.go — 关闭到托盘的后台生命周期状态机(平台无关)
// 关闭按钮: 托盘可用 → 隐藏窗口; 托盘失败 → 关闭即真正退出(降级)
// 真正退出入口: 托盘菜单"退出炼丹炉"(经 RequestQuit → Wails runtime.Quit)
package desktop

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/alchemy-furnace/server/internal/desktoptray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// windowRuntime 窄接口: 生命周期层只依赖这几个窗口操作(测试用 fake 替换)
type windowRuntime interface {
	Hide(context.Context)
	Show(context.Context)
	Unminimise(context.Context)
	About(context.Context)
	Quit(context.Context)
}

// wailsWindowRuntime 无状态适配器: 窄接口 → Wails runtime
type wailsWindowRuntime struct{}

func (wailsWindowRuntime) Hide(ctx context.Context)           { wailsruntime.WindowHide(ctx) }
func (wailsWindowRuntime) Show(ctx context.Context)           { wailsruntime.WindowShow(ctx) }
func (wailsWindowRuntime) Unminimise(ctx context.Context)     { wailsruntime.WindowUnminimise(ctx) }
func (wailsWindowRuntime) Quit(ctx context.Context)           { wailsruntime.Quit(ctx) }
func (wailsWindowRuntime) About(ctx context.Context) {
	_, _ = wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Type:    wailsruntime.InfoDialog,
		Title:   "关于炼丹炉",
		Message: "Alchemy Furnace",
	})
}

// desktopLifecycle 关闭/恢复/退出状态机
type desktopLifecycle struct {
	mu     sync.RWMutex
	ctx    context.Context
	window windowRuntime
	tray   *desktoptray.Controller

	trayReady atomic.Bool
	quitting  atomic.Bool

	shutdownOnce sync.Once
}

func newDesktopLifecycle(window windowRuntime, tray *desktoptray.Controller) *desktopLifecycle {
	return &desktopLifecycle{window: window, tray: tray}
}

// Start 注入 Wails context 并启动托盘(幂等); 托盘失败仅记录错误, 不弹阻塞对话框
func (l *desktopLifecycle) Start(ctx context.Context) error {
	l.mu.Lock()
	l.ctx = ctx
	l.mu.Unlock()
	err := l.tray.Start(desktoptray.Callbacks{
		Open: l.ShowMainWindow,
		Quit: l.RequestQuit,
	})
	l.trayReady.Store(l.tray.Ready())
	return err
}

// BeforeClose 窗口关闭回调: true=吞掉关闭(隐藏到托盘), false=允许退出
// 先查 quitting(真正退出中不拦截), 再查 trayReady(托盘失败时关闭即退出)
func (l *desktopLifecycle) BeforeClose(ctx context.Context) bool {
	if l.quitting.Load() {
		return false
	}
	if !l.trayReady.Load() {
		return false
	}
	l.HideMainWindow()
	return true
}

// HideMainWindow 隐藏主窗口; context 未注入(Wails 未就绪)时不执行
func (l *desktopLifecycle) HideMainWindow() {
	l.mu.RLock()
	ctx := l.ctx
	l.mu.RUnlock()
	if ctx == nil {
		return
	}
	l.window.Hide(ctx)
}

// ShowMainWindow 恢复主窗口: 先取消最小化再显示
func (l *desktopLifecycle) ShowMainWindow() {
	l.mu.RLock()
	ctx := l.ctx
	l.mu.RUnlock()
	if ctx == nil {
		return
	}
	l.window.Unminimise(ctx)
	l.window.Show(ctx)
}

// ShowAbout 打开"关于炼丹炉"信息对话框
func (l *desktopLifecycle) ShowAbout() {
	l.mu.RLock()
	ctx := l.ctx
	l.mu.RUnlock()
	if ctx == nil {
		return
	}
	l.window.About(ctx)
}

// RequestQuit 完整退出(托盘"退出炼丹炉"): 只触发一次, 走 Wails OnShutdown
func (l *desktopLifecycle) RequestQuit() {
	if !l.quitting.CompareAndSwap(false, true) {
		return
	}
	l.mu.RLock()
	ctx := l.ctx
	l.mu.RUnlock()
	if ctx == nil {
		return
	}
	l.window.Quit(ctx)
}

// Shutdown 应用退出时的托盘清理(幂等, 由 OnShutdown 统一调用)
func (l *desktopLifecycle) Shutdown() {
	l.shutdownOnce.Do(func() {
		_ = l.tray.Stop()
	})
}
