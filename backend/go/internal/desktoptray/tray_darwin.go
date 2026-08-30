//go:build darwin

package desktoptray

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>

// 由 tray_darwin.m 提供; 非零=启动失败(超时/创建失败), 供 Go 侧降级
int alchemy_tray_start(const unsigned char *icon, int icon_len);
void alchemy_tray_stop(void);
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

// darwinNative 可替换的 AppKit 边界(fake 记录调用, 真实实现走 C)
type darwinNative interface {
	start(icon []byte, iconLen int) int
	stop()
}

type realDarwinNative struct{}

func (realDarwinNative) start(icon []byte, iconLen int) int {
	return int(C.alchemy_tray_start((*C.uchar)(unsafe.Pointer(&icon[0])), C.int(iconLen)))
}

func (realDarwinNative) stop() {
	C.alchemy_tray_stop()
}

// 包级回调: C 侧菜单经 //export 回调时, 在锁内取当前回调(Stop 清空后自然忽略)
var (
	cbMu   sync.RWMutex
	openCb func()
	quitCb func()
)

//export alchemyTrayOpen
func alchemyTrayOpen() {
	cbMu.RLock()
	fn := openCb
	cbMu.RUnlock()
	if fn != nil {
		fn()
	}
}

//export alchemyTrayQuit
func alchemyTrayQuit() {
	cbMu.RLock()
	fn := quitCb
	cbMu.RUnlock()
	if fn != nil {
		fn()
	}
}

type darwinBackend struct {
	native    darwinNative
	startOnce sync.Once
	stopOnce  sync.Once
	startErr  error
}

func newDarwinBackend(n darwinNative) *darwinBackend {
	return &darwinBackend{native: n}
}

// Start 注册回调并创建 NSStatusItem(幂等); native 非零返回 → 明确错误
func (b *darwinBackend) Start(cb Callbacks) error {
	b.startOnce.Do(func() {
		cbMu.Lock()
		openCb, quitCb = cb.Open, cb.Quit
		cbMu.Unlock()
		// AppKit 中该 NSImage 的逻辑尺寸会设为 22pt。传入 44px 的 @2x 资源，
		// Retina 菜单栏可一一映射到物理像素；若传 22px，系统会放大并产生模糊边缘。
		if rc := b.native.start(macTemplateIcon2x, len(macTemplateIcon2x)); rc != 0 {
			b.startErr = errors.New("desktoptray: macOS status item start failed")
		}
	})
	return b.startErr
}

// Stop 移除状态项并清空回调(幂等); 迟到回调因 openCb/quitCb 为 nil 被忽略
func (b *darwinBackend) Stop() error {
	b.stopOnce.Do(func() {
		cbMu.Lock()
		openCb, quitCb = nil, nil
		cbMu.Unlock()
		b.native.stop()
	})
	return nil
}

// New macOS 平台工厂
func New() *Controller {
	if disabledByEnvironment() {
		return newController(disabledBackend{})
	}
	return newController(newDarwinBackend(realDarwinNative{}))
}
