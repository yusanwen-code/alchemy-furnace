//go:build darwin

package desktoptray

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeDarwinNative 记录 C 边界调用
type fakeDarwinNative struct {
	starts   int
	stops    int
	startErr error
	lastIcon []byte
}

func (f *fakeDarwinNative) start(icon []byte, iconLen int) int {
	f.starts++
	f.lastIcon = icon
	if f.startErr != nil {
		return 1
	}
	return 0
}

func (f *fakeDarwinNative) stop() { f.stops++ }

// 导出回调转发: C 侧菜单触发 alchemyTrayOpen/alchemyTrayQuit → Go 回调
func TestDarwinCallbacksForwardToGo(t *testing.T) {
	open, quit := 0, 0
	b := newDarwinBackend(&fakeDarwinNative{})
	require.NoError(t, b.Start(Callbacks{Open: func() { open++ }, Quit: func() { quit++ }}))

	alchemyTrayOpen()
	require.Equal(t, 1, open)
	alchemyTrayQuit()
	require.Equal(t, 1, quit)
	require.NoError(t, b.Stop())
}

// Stop 后迟到回调被忽略(菜单已移除, 残留事件不得再触达 Go 回调)
func TestDarwinLateCallbacksIgnored(t *testing.T) {
	open := 0
	b := newDarwinBackend(&fakeDarwinNative{})
	require.NoError(t, b.Start(Callbacks{Open: func() { open++ }}))
	require.NoError(t, b.Stop())
	alchemyTrayOpen()
	alchemyTrayQuit()
	require.Equal(t, 0, open)
}

// Start/Stop 各只调一次 native(幂等)
func TestDarwinStartStopOnce(t *testing.T) {
	f := &fakeDarwinNative{}
	b := newDarwinBackend(f)
	require.NoError(t, b.Start(Callbacks{}))
	require.NoError(t, b.Start(Callbacks{}))
	require.NoError(t, b.Stop())
	require.NoError(t, b.Stop())
	require.Equal(t, 1, f.starts)
	require.Equal(t, 1, f.stops)
}

// native start 失败 → 明确错误
func TestDarwinStartFailure(t *testing.T) {
	f := &fakeDarwinNative{startErr: errors.New("boom")}
	b := newDarwinBackend(f)
	err := b.Start(Callbacks{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "macOS status item start failed")
}

// NSImage 会把资源设置为 22pt；Retina 屏必须传 44px 的 @2x 图，不能放大 22px 图。
func TestDarwinPassesRetinaTemplateIcon(t *testing.T) {
	f := &fakeDarwinNative{}
	b := newDarwinBackend(f)
	require.NoError(t, b.Start(Callbacks{}))
	require.Equal(t, macTemplateIcon2x, f.lastIcon)
	require.NoError(t, b.Stop())
}

// 环境开关: ALCHEMY_TRAY_DISABLE=1 时平台工厂走禁用 backend
func TestDarwinNewRespectsDisableEnv(t *testing.T) {
	t.Setenv("ALCHEMY_TRAY_DISABLE", "1")
	c := New()
	require.Error(t, c.Start(Callbacks{}))
	require.False(t, c.Ready())
}
