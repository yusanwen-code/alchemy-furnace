//go:build windows

package desktoptray

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unicode/utf16"
	"unsafe"

	winapi "golang.org/x/sys/windows"
)

// ─── 固定消息与命令 ID ───
const (
	wmTray               = 0x0400 + 1 // WM_APP+1: 托盘回调消息
	wmCommand            = 0x0111
	wmClose              = 0x0010
	wmQuit               = 0x0012
	wmRButtonUp          = 0x0205
	wmLButtonDoubleClick = 0x0203
	cmdOpen              = 1001
	cmdQuit              = 1002
	windowsTip           = "炼丹炉"
)

// NIF_* / NIM_* / 菜单 flags
const (
	nifMessage  = 0x1
	nifIcon     = 0x2
	nifTip      = 0x4
	nimAdd      = 0x1
	nimDelete   = 0x2
	mfString    = 0x0
	mfSeparator = 0x800
	tpmRightBtn = 0x2
	imgIcon     = 0x1
	lrLoadFile  = 0x10
	lrDefSize   = 0x40
)

// winMsg 与 Windows MSG 布局一致(供 GetMessageW 直接读写)
type winMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	ptX     int32
	ptY     int32
}

// winNotifyData NOTIFYICONDATAW(x64 布局, cbSize=sizeof 精确匹配 Vista+)
type winNotifyData struct {
	cbSize            uint32
	hwnd              uintptr
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	hIcon             uintptr
	tip               [128]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [256]uint16
	uTimeoutOrVersion uint32
	szInfoTitle       [64]uint16
	dwInfoFlags       uint32
	guidItem          [16]byte
	hBalloonIcon      uintptr
}

// wndClassEx WNDCLASSEXW(RegisterClassExW 用)
type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

// ─── Win32 边界 ───

// windowsAPI 可替换的 Win32 调用边界(测试用 fake 记录, 真实实现走 LazyDLL)
type windowsAPI interface {
	registerWindowClass(name *uint16, wndProc uintptr) (uint16, error)
	createHiddenWindow(name *uint16) (uintptr, error)
	destroyWindow(hwnd uintptr) error
	registerTaskbarCreated() uint32
	getMessage(msg *winMsg) (int32, error)
	translateMessage(msg *winMsg) error
	dispatchMessage(msg *winMsg) uintptr
	postMessage(hwnd uintptr, msg uint32, w, l uintptr) error
	shellNotifyIcon(hwnd uintptr, add bool, nid *winNotifyData) error
	createPopupMenu() (uintptr, error)
	appendMenu(menu uintptr, flags uint32, id uintptr, text string) error
	trackPopupMenu(menu uintptr, hwnd uintptr) error
	destroyMenu(menu uintptr) error
	loadIconFromFile(path string) (uintptr, error)
	destroyIcon(h uintptr) error
}

// 隐藏窗口的窗口过程: 所有消息由消息循环手动分派, 这里只需转发 DefWindowProcW
var wndProcPtr uintptr

func init() {
	proc := winapi.NewLazySystemDLL("user32.dll").NewProc("DefWindowProcW")
	wndProcPtr = winapi.NewCallback(func(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
		r, _, _ := proc.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	})
}

// utf16PtrToString []uint16 → Go string(供 fake 断言 tooltip)
func utf16PtrToString(u []uint16) string {
	return winapi.UTF16ToString(u)
}

// ─── 真实 Win32 实现 ───

var (
	kernel32 = winapi.NewLazySystemDLL("kernel32.dll")
	user32   = winapi.NewLazySystemDLL("user32.dll")
	shell32  = winapi.NewLazySystemDLL("shell32.dll")

	procGetModuleHandleW      = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW      = user32.NewProc("RegisterClassExW")
	procCreateWindowExW       = user32.NewProc("CreateWindowExW")
	procRegisterWindowMessage = user32.NewProc("RegisterWindowMessageW")
	procGetMessageW           = user32.NewProc("GetMessageW")
	procTranslateMessage      = user32.NewProc("TranslateMessage")
	procDispatchMessageW      = user32.NewProc("DispatchMessageW")
	procPostMessageW          = user32.NewProc("PostMessageW")
	procGetCursorPos          = user32.NewProc("GetCursorPos")
	procDestroyWindow         = user32.NewProc("DestroyWindow")
	procCreatePopupMenu       = user32.NewProc("CreatePopupMenu")
	procAppendMenuW           = user32.NewProc("AppendMenuW")
	procTrackPopupMenu        = user32.NewProc("TrackPopupMenu")
	procDestroyMenu           = user32.NewProc("DestroyMenu")
	procDestroyIcon           = user32.NewProc("DestroyIcon")
	procLoadImageW            = user32.NewProc("LoadImageW")
	procShellNotifyIconW      = shell32.NewProc("Shell_NotifyIconW")
)

type realWindowsAPI struct{}

// hInstance 取当前进程模块句柄(GetModuleHandleW(nil))
func processInstance() uintptr {
	r, _, _ := procGetModuleHandleW.Call(0)
	return r
}

func (realWindowsAPI) registerWindowClass(name *uint16, wndProc uintptr) (uint16, error) {
	wc := &wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		style:         0, // 隐藏窗口: 无重绘/无消息重排需求
		lpfnWndProc:   wndProc,
		hInstance:     processInstance(),
		lpszClassName: name,
	}
	r, _, err2 := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
	if r == 0 {
		if err2 == winapi.ERROR_CLASS_ALREADY_EXISTS {
			return 0, nil // 单实例进程内重复注册无害
		}
		return 0, err2
	}
	return uint16(r), nil
}

func (realWindowsAPI) createHiddenWindow(name *uint16) (uintptr, error) {
	r, _, err2 := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(name)), 0, // exStyle, className, 无标题
		0, // style=0: 无可见/任务栏/标题样式
		0, 0, 0, 0,
		0, 0, processInstance(), 0,
	)
	if r == 0 {
		return 0, err2
	}
	return r, nil
}

func (realWindowsAPI) destroyWindow(hwnd uintptr) error {
	r, _, err := procDestroyWindow.Call(hwnd)
	if r == 0 {
		return err
	}
	return nil
}

func (realWindowsAPI) registerTaskbarCreated() uint32 {
	name := winapi.StringToUTF16Ptr("TaskbarCreated")
	r, _, _ := procRegisterWindowMessage.Call(uintptr(unsafe.Pointer(name)))
	return uint32(r)
}

func (realWindowsAPI) getMessage(msg *winMsg) (int32, error) {
	r, _, err := procGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)), 0, 0, 0,
	)
	if int32(r) == -1 {
		return -1, err
	}
	return int32(r), nil // 0=WM_QUIT, >0=普通消息
}

func (realWindowsAPI) translateMessage(msg *winMsg) error {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
	return nil
}

func (realWindowsAPI) dispatchMessage(msg *winMsg) uintptr {
	r, _, _ := procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
	return r
}

func (realWindowsAPI) postMessage(hwnd uintptr, msg uint32, w, l uintptr) error {
	r, _, err := procPostMessageW.Call(hwnd, uintptr(msg), w, l)
	if r == 0 {
		return err
	}
	return nil
}

func (realWindowsAPI) shellNotifyIcon(hwnd uintptr, add bool, nid *winNotifyData) error {
	cmd := uintptr(nimAdd)
	if !add {
		cmd = nimDelete
	}
	r, _, err := procShellNotifyIconW.Call(cmd, uintptr(unsafe.Pointer(nid)))
	if r == 0 {
		return err
	}
	return nil
}

func (realWindowsAPI) createPopupMenu() (uintptr, error) {
	r, _, err := procCreatePopupMenu.Call()
	if r == 0 {
		return 0, err
	}
	return r, nil
}

func (realWindowsAPI) appendMenu(menu uintptr, flags uint32, id uintptr, text string) error {
	r, _, err := procAppendMenuW.Call(
		menu, uintptr(flags), id,
		uintptr(unsafe.Pointer(winapi.StringToUTF16Ptr(text))),
	)
	if r == 0 {
		return err
	}
	return nil
}

func (realWindowsAPI) trackPopupMenu(menu uintptr, hwnd uintptr) error {
	var pt struct{ X, Y int32 }
	if r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
		pt.X, pt.Y = 0, 0
	}
	// TPM_RIGHTBUTTON: 命令经 WM_COMMAND 发给 hwnd(与 cmdOpen/cmdQuit 模型一致)
	procTrackPopupMenu.Call(menu, tpmRightBtn, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	return nil
}

func (realWindowsAPI) destroyMenu(menu uintptr) error {
	r, _, err := procDestroyMenu.Call(menu)
	if r == 0 {
		return err
	}
	return nil
}

func (realWindowsAPI) destroyIcon(h uintptr) error {
	r, _, err := procDestroyIcon.Call(h)
	if r == 0 {
		return err
	}
	return nil
}

func (realWindowsAPI) loadIconFromFile(path string) (uintptr, error) {
	r, _, err := procLoadImageW.Call(
		0, // hinst: 从文件加载时忽略
		uintptr(unsafe.Pointer(winapi.StringToUTF16Ptr(path))),
		imgIcon, 0, 0,
		lrLoadFile|lrDefSize,
	)
	if r == 0 {
		return 0, err
	}
	return r, nil
}

// ─── backend ───

type windowsBackend struct {
	api         windowsAPI
	callbacks   Callbacks
	hwnd        uintptr
	menu        uintptr
	icon        uintptr
	iconPath    string
	taskbarMsg  uint32
	msgLoopDone chan struct{}
	cleaned     bool

	startOnce sync.Once
	startErr  error
	stopOnce  sync.Once
}

func newWindowsBackend(api windowsAPI) *windowsBackend {
	return &windowsBackend{api: api}
}

// New Windows 平台工厂
func New() *Controller {
	if disabledByEnvironment() {
		return newController(disabledBackend{})
	}
	return newController(newWindowsBackend(realWindowsAPI{}))
}

// Start 注册隐藏窗口、TaskbarCreated 消息、托盘图标, 并启动消息循环 goroutine
func (b *windowsBackend) Start(cb Callbacks) error {
	b.startOnce.Do(func() {
		b.callbacks = cb
		className := winapi.StringToUTF16Ptr("AlchemyFurnaceTrayWindow")
		if _, err := b.api.registerWindowClass(className, wndProcPtr); err != nil {
			b.startErr = fmt.Errorf("desktoptray: register window class: %w", err)
			return
		}
		hwnd, err := b.api.createHiddenWindow(className)
		if err != nil {
			b.startErr = fmt.Errorf("desktoptray: create hidden window: %w", err)
			return
		}
		b.hwnd = hwnd
		b.taskbarMsg = b.api.registerTaskbarCreated()
		if err := b.writeIconFile(); err != nil {
			b.startErr = fmt.Errorf("desktoptray: cache tray icon: %w", err)
			return
		}
		hicon, err := b.api.loadIconFromFile(b.iconPath)
		if err != nil {
			b.startErr = fmt.Errorf("desktoptray: load tray icon: %w", err)
			return
		}
		b.icon = hicon
		if err := b.api.shellNotifyIcon(hwnd, true, b.nid()); err != nil {
			b.startErr = fmt.Errorf("desktoptray: notify icon add: %w", err)
			return
		}
		b.msgLoopDone = make(chan struct{})
		go b.messageLoop()
	})
	return b.startErr
}

// Stop 通知消息线程关闭并等待清理; 消息循环已退出时兜底就地清理(幂等)
func (b *windowsBackend) Stop() error {
	b.stopOnce.Do(func() {
		if b.hwnd != 0 {
			_ = b.api.postMessage(b.hwnd, wmClose, 0, 0)
			if b.msgLoopDone != nil {
				select {
				case <-b.msgLoopDone:
				case <-time.After(3 * time.Second):
				}
			}
			// 真实路径清理发生在消息线程(WM_CLOSE); 循环已死时这里兜底
			b.cleanup()
		}
	})
	return nil
}

// nid 组装 NOTIFYICONDATAW: 消息+图标+tooltip(精确 "炼丹炉")
func (b *windowsBackend) nid() *winNotifyData {
	nid := &winNotifyData{
		cbSize:           uint32(unsafe.Sizeof(winNotifyData{})),
		hwnd:             b.hwnd,
		uID:              1,
		uFlags:           nifMessage | nifIcon | nifTip,
		uCallbackMessage: wmTray,
		hIcon:            b.icon,
	}
	u := utf16.Encode([]rune(windowsTip))
	copy(nid.tip[:], u)
	return nid
}

// writeIconFile 嵌入 ICO 写入缓存(文件名含 SHA-256 前 16 位, 幂等复用)
func (b *windowsBackend) writeIconFile() error {
	base, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(base, "AlchemyFurnace", "tray")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	sum := sha256.Sum256(windowsIcon)
	name := fmt.Sprintf("furnace-%x.ico", sum[:8])
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		b.iconPath = path
		return nil
	}
	if err := os.WriteFile(path, windowsIcon, 0o600); err != nil {
		return err
	}
	b.iconPath = path
	return nil
}

// messageLoop 专用线程跑 GetMessageW 泵; 每条消息先走 handleWindowMessage
func (b *windowsBackend) messageLoop() {
	defer close(b.msgLoopDone)
	runtime.LockOSThread()
	var msg winMsg
	for {
		n, err := b.api.getMessage(&msg)
		if err != nil || n == 0 {
			return // error=循环异常, 0=WM_QUIT
		}
		if !b.handleWindowMessage(msg.message, msg.wParam, msg.lParam) {
			_ = b.api.translateMessage(&msg)
			b.api.dispatchMessage(&msg)
		}
	}
}

// handleWindowMessage 分派一条窗口消息; 返回 true 表示已处理
func (b *windowsBackend) handleWindowMessage(msg uint32, wParam, lParam uintptr) bool {
	switch {
	case b.taskbarMsg != 0 && msg == b.taskbarMsg:
		// Explorer 重启: 托盘图标被系统清除, 重新注册(幂等, 不产生重复图标)
		if b.hwnd != 0 {
			_ = b.api.shellNotifyIcon(b.hwnd, true, b.nid())
		}
		return true
	case msg == wmTray:
		switch lParam {
		case wmLButtonDoubleClick:
			b.callbacks.Open()
		case wmRButtonUp:
			b.showMenu()
		}
		return true
	case msg == wmCommand:
		switch wParam {
		case cmdOpen:
			b.callbacks.Open()
		case cmdQuit:
			b.callbacks.Quit()
		}
		return true
	case msg == wmClose:
		b.cleanup()
		_ = b.api.postMessage(b.hwnd, wmQuit, 0, 0)
		return true
	}
	return false
}

// showMenu 右击弹出菜单: 打开炼丹炉 / 分隔 / 退出炼丹炉
func (b *windowsBackend) showMenu() {
	menu, err := b.api.createPopupMenu()
	if err != nil {
		return
	}
	b.menu = menu
	_ = b.api.appendMenu(menu, mfString, cmdOpen, "打开炼丹炉")
	_ = b.api.appendMenu(menu, mfSeparator, 0, "")
	_ = b.api.appendMenu(menu, mfString, cmdQuit, "退出炼丹炉")
	_ = b.api.trackPopupMenu(menu, b.hwnd)
}

// cleanup 按固定顺序释放: NIM_DELETE → DestroyMenu → DestroyIcon → DestroyWindow → 删缓存文件
// 幂等: cleaned 置位后第二次调用直接返回(Stop 兜底与消息线程路径共用)
func (b *windowsBackend) cleanup() {
	if b.cleaned {
		return
	}
	b.cleaned = true
	if b.hwnd != 0 {
		_ = b.api.shellNotifyIcon(b.hwnd, false, b.nid())
	}
	if b.menu != 0 {
		_ = b.api.destroyMenu(b.menu)
		b.menu = 0
	}
	if b.icon != 0 {
		_ = b.api.destroyIcon(b.icon)
		b.icon = 0
	}
	if b.hwnd != 0 {
		_ = b.api.destroyWindow(b.hwnd)
		b.hwnd = 0
	}
	if b.iconPath != "" {
		_ = os.Remove(b.iconPath)
		b.iconPath = ""
	}
}
