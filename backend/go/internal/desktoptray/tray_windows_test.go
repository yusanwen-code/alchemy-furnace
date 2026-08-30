//go:build windows

package desktoptray

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fake 消息循环的终止哨兵: GetMessage 返回它立即退出循环(测试环境无真实消息)
var errTestMessageLoopEnd = errors.New("test: message loop end")

// ─── fake Win32 边界 ───

type fakeWinAPI struct {
	adds       int
	deletes    int
	created    bool
	destroyed  bool
	tips       []string
	popupMenus int
	menuBuilt  bool
	destroyedIcons int
	loads      int
	hwnd       uintptr
}

func (f *fakeWinAPI) registerWindowClass(name *uint16, wndProc uintptr) (uint16, error) {
	return 1, nil
}
func (f *fakeWinAPI) createHiddenWindow(name *uint16) (uintptr, error) {
	f.created = true
	return f.hwnd, nil
}
func (f *fakeWinAPI) destroyWindow(hwnd uintptr) error { f.destroyed = true; return nil }
func (f *fakeWinAPI) registerTaskbarCreated() uint32   { return 0x8000 }
func (f *fakeWinAPI) getMessage(msg *winMsg) (int32, error) {
	return -1, errTestMessageLoopEnd // 立即结束循环
}
func (f *fakeWinAPI) translateMessage(msg *winMsg) error  { return nil }
func (f *fakeWinAPI) dispatchMessage(msg *winMsg) uintptr { return 0 }
func (f *fakeWinAPI) postMessage(hwnd uintptr, msg uint32, w, l uintptr) error {
	return nil
}
func (f *fakeWinAPI) shellNotifyIcon(hwnd uintptr, add bool, nid *winNotifyData) error {
	if add {
		f.adds++
		f.tips = append(f.tips, utf16PtrToString(nid.tip[:]))
	} else {
		f.deletes++
	}
	return nil
}
func (f *fakeWinAPI) createPopupMenu() (uintptr, error) { f.popupMenus++; return 0x5001, nil }
func (f *fakeWinAPI) appendMenu(menu uintptr, flags uint32, id uintptr, text string) error {
	f.menuBuilt = true
	return nil
}
func (f *fakeWinAPI) trackPopupMenu(menu uintptr, hwnd uintptr) error { return nil }
func (f *fakeWinAPI) destroyMenu(menu uintptr) error                  { return nil }
func (f *fakeWinAPI) loadIconFromFile(path string) (uintptr, error)   { f.loads++; return 0x6001, nil }
func (f *fakeWinAPI) destroyIcon(h uintptr) error                     { f.destroyedIcons++; return nil }

// ─── 行为测试 ───

// Start 注册隐藏窗口并只 NIM_ADD 一次; tooltip 精确为 UTF-16 "炼丹炉"
func TestWindowsBackendStartRegistersIconOnce(t *testing.T) {
	f := &fakeWinAPI{hwnd: 0x1234}
	b := newWindowsBackend(f)
	open, quit := 0, 0
	require.NoError(t, b.Start(Callbacks{Open: func() { open++ }, Quit: func() { quit++ }}))
	require.Equal(t, 1, f.adds)
	require.True(t, f.created)
	require.Len(t, f.tips, 1)
	require.Equal(t, "炼丹炉", f.tips[0])
	require.NoError(t, b.Stop())
}

// 菜单命令 1001=Open, 1002=Quit
func TestWindowsTrayCommandsCallCallbacks(t *testing.T) {
	f := &fakeWinAPI{hwnd: 0x1234}
	b := newWindowsBackend(f)
	open, quit := 0, 0
	require.NoError(t, b.Start(Callbacks{Open: func() { open++ }, Quit: func() { quit++ }}))

	require.True(t, b.handleWindowMessage(wmCommand, cmdOpen, 0))
	require.Equal(t, 1, open)
	require.Equal(t, 0, quit)

	require.True(t, b.handleWindowMessage(wmCommand, cmdQuit, 0))
	require.Equal(t, 1, open)
	require.Equal(t, 1, quit)

	require.NoError(t, b.Stop())
}

// 左键双击恢复窗口
func TestWindowsTrayDoubleClickOpens(t *testing.T) {
	f := &fakeWinAPI{hwnd: 0x1234}
	b := newWindowsBackend(f)
	open := 0
	require.NoError(t, b.Start(Callbacks{Open: func() { open++ }}))

	require.True(t, b.handleWindowMessage(wmTray, 0, wmLButtonDoubleClick))
	require.Equal(t, 1, open)
	require.NoError(t, b.Stop())
}

// Explorer 重启广播 TaskbarCreated 后必须重新 NIM_ADD(且不重复)
func TestWindowsTaskbarCreatedReaddsIcon(t *testing.T) {
	f := &fakeWinAPI{hwnd: 0x1234}
	b := newWindowsBackend(f)
	require.NoError(t, b.Start(Callbacks{}))
	require.Equal(t, 1, f.adds)

	require.True(t, b.handleWindowMessage(b.taskbarMsg, 0, 0))
	require.True(t, b.handleWindowMessage(b.taskbarMsg, 0, 0))
	require.Equal(t, 3, f.adds)
	require.NoError(t, b.Stop())
}

// Stop 只 NIM_DELETE 一次, 销毁窗口; 二次 Stop 无副作用
func TestWindowsBackendStopDeletesOnce(t *testing.T) {
	f := &fakeWinAPI{hwnd: 0x1234}
	b := newWindowsBackend(f)
	require.NoError(t, b.Start(Callbacks{}))
	require.NoError(t, b.Stop())
	require.NoError(t, b.Stop())
	require.Equal(t, 1, f.adds)
	require.Equal(t, 1, f.deletes)
	require.True(t, f.destroyed)
}

// 右击弹出菜单(打开/分隔/退出)
func TestWindowsBackendMenuItems(t *testing.T) {
	f := &fakeWinAPI{hwnd: 0x1234}
	b := newWindowsBackend(f)
	require.NoError(t, b.Start(Callbacks{}))
	b.showMenu()
	require.Equal(t, 1, f.popupMenus)
	require.True(t, f.menuBuilt)
	require.NoError(t, b.Stop())
}
