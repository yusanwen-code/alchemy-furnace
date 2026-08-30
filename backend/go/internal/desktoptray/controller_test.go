package desktoptray

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// 记录调用序列的 fake backend
type fakeBackend struct {
	starts    int
	stops     int
	callbacks Callbacks
	err       error
}

func (f *fakeBackend) Start(c Callbacks) error {
	f.starts++
	f.callbacks = c
	return f.err
}

func (f *fakeBackend) Stop() error {
	f.stops++
	return nil
}

// Start/Stop 幂等: 二次 Start 不重复启动, 二次 Stop 不重复清理
func TestControllerStartStopAreIdempotent(t *testing.T) {
	f := &fakeBackend{}
	c := newController(f)
	require.NoError(t, c.Start(Callbacks{Open: func() {}, Quit: func() {}}))
	require.NoError(t, c.Start(Callbacks{}))
	require.True(t, c.Ready())
	require.NoError(t, c.Stop())
	require.NoError(t, c.Stop())
	require.Equal(t, 1, f.starts)
	require.Equal(t, 1, f.stops)
}

// Start 失败: Ready()==false, 后续 Stop 不触碰 backend
func TestControllerStartFailureKeepsNotReady(t *testing.T) {
	f := &fakeBackend{err: os.ErrInvalid}
	c := newController(f)
	require.Error(t, c.Start(Callbacks{}))
	require.False(t, c.Ready())
	require.NoError(t, c.Stop())
	require.Equal(t, 1, f.starts)
	require.Equal(t, 0, f.stops)
}

// nil 回调必须被替换为空函数: backend 收到后可安全调用不 panic
func TestControllerReplacesNilCallbacks(t *testing.T) {
	f := &fakeBackend{}
	c := newController(f)
	require.NoError(t, c.Start(Callbacks{}))
	require.NotNil(t, f.callbacks.Open)
	require.NotNil(t, f.callbacks.Quit)
	require.NotPanics(t, func() { f.callbacks.Open() })
	require.NotPanics(t, func() { f.callbacks.Quit() })
}

// 环境开关: ALCHEMY_TRAY_DISABLE=1 时禁用, 返回明确错误(仅用于失败降级测试)
func TestDisabledByEnvironmentReturnsExplicitError(t *testing.T) {
	t.Setenv("ALCHEMY_TRAY_DISABLE", "1")
	c := newController(disabledBackend{})
	err := c.Start(Callbacks{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ALCHEMY_TRAY_DISABLE")
	require.False(t, c.Ready())
}
