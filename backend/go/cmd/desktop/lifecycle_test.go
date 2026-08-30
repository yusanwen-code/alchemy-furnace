package desktop

import (
	"context"
	"errors"
	"testing"

	"github.com/alchemy-furnace/server/internal/desktoptray"
	"github.com/stretchr/testify/require"
)

// fakeWindowRuntime 记录窄接口调用
type fakeWindowRuntime struct {
	hides       int
	shows       int
	unminimises int
	abouts      int
	quits       int
}

func (f *fakeWindowRuntime) Hide(context.Context)       { f.hides++ }
func (f *fakeWindowRuntime) Show(context.Context)       { f.shows++ }
func (f *fakeWindowRuntime) Unminimise(context.Context) { f.unminimises++ }
func (f *fakeWindowRuntime) About(context.Context)      { f.abouts++ }
func (f *fakeWindowRuntime) Quit(context.Context)       { f.quits++ }

type fakeTrayBackend struct {
	err   error
	stops int
}

func (f *fakeTrayBackend) Start(desktoptray.Callbacks) error { return f.err }
func (f *fakeTrayBackend) Stop() error                       { f.stops++; return nil }

// fixture: trayErr=nil → 托盘就绪; 非 nil → Start 失败(降级路径)
func newLifecycleFixture(trayErr error) (*desktopLifecycle, *fakeWindowRuntime, *fakeTrayBackend) {
	win := &fakeWindowRuntime{}
	tb := &fakeTrayBackend{err: trayErr}
	l := newDesktopLifecycle(win, desktoptray.NewController(tb))
	return l, win, tb
}

// 托盘就绪: 关闭 → 隐藏窗口并返回 true(吞掉关闭)
func TestLifecycleBeforeCloseHidesWhenTrayReady(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	require.True(t, l.BeforeClose(context.Background()))
	require.Equal(t, 1, win.hides)
}

// 托盘失败: 关闭 → 不隐藏, 返回 false(真正退出)
func TestLifecycleBeforeCloseQuitsWhenTrayNotReady(t *testing.T) {
	l, win, _ := newLifecycleFixture(errors.New("tray boom"))
	require.Error(t, l.Start(context.Background()))
	require.False(t, l.BeforeClose(context.Background()))
	require.Equal(t, 0, win.hides)
}

// context 未注入时 HideMainWindow 不执行(不 panic, 不调 Wails runtime)
func TestLifecycleHideWithoutContextIsNoop(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	l.HideMainWindow() // 未 Start: ctx 为 nil
	require.Equal(t, 0, win.hides)
	require.NoError(t, l.Start(context.Background()))
	l.HideMainWindow()
	require.Equal(t, 1, win.hides)
}

// 恢复窗口: 先 Unminimise 再 Show
func TestLifecycleShowMainWindowUnminimiseThenShow(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	l.ShowMainWindow()
	require.Equal(t, 1, win.unminimises)
	require.Equal(t, 1, win.shows)
}

// 关于对话框: context 可用时打开
func TestLifecycleShowAboutOpensDialog(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	l.ShowAbout()
	require.Equal(t, 1, win.abouts)
}

// RequestQuit 只触发一次 Quit; 二次调用无动作
func TestLifecycleRequestQuitOnce(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	l.RequestQuit()
	l.RequestQuit()
	require.Equal(t, 1, win.quits)
}

// quitting 后 BeforeClose 直接返回 false(允许退出流程继续)
func TestLifecycleBeforeCloseAfterQuitReturnsFalse(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	l.RequestQuit()
	require.False(t, l.BeforeClose(context.Background()))
	require.Equal(t, 0, win.hides)
}

// Shutdown 只调一次托盘 Stop
func TestLifecycleShutdownStopsTrayOnce(t *testing.T) {
	_, _, tb := newLifecycleFixture(nil)
	tray := desktoptray.NewController(tb)
	l := newDesktopLifecycle(&fakeWindowRuntime{}, tray)
	require.NoError(t, l.Start(context.Background()))
	l.Shutdown()
	l.Shutdown()
	require.Equal(t, 1, tb.stops)
}
