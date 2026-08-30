//go:build darwin

package dockreopen

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <objc/runtime.h>
#include <objc/message.h>
#import <Foundation/Foundation.h>

// 由 dockreopen_darwin.m 提供; 0=成功, 1=AppDelegate 类不可用, 2=wails 已实现(跳过)
int alchemy_dock_reopen_install(void);

// ---- 测试辅助(Go 侧包装后供 _test.go 调用; _test.go 不能直接 import "C") ----

// 检查 wails AppDelegate 类是否已获得 applicationShouldHandleReopen: 方法
// -1=类不可用, 0=未注册, 1=已注册
static int test_has_reopen_method(void) {
	Class cls = objc_getClass("AppDelegate");
	if (cls == NULL) {
		return -1;
	}
	SEL sel = sel_registerName("applicationShouldHandleReopen:hasVisibleWindows:");
	return class_getInstanceMethod(cls, sel) != NULL ? 1 : 0;
}

// 直接调用 reopen IMP(模拟点击 Dock 图标); IMP 不访问 self/sender, 传 nil 安全
// 返回 0=未能触达, 1=已触发
static int test_fire_reopen(void) {
	Class cls = objc_getClass("AppDelegate");
	if (cls == NULL) {
		return 0;
	}
	SEL sel = sel_registerName("applicationShouldHandleReopen:hasVisibleWindows:");
	// class_getMethodImplementation 对不存在的方法返回转发桩而非 NULL, 必须查实例方法
	Method m = class_getInstanceMethod(cls, sel);
	if (m == NULL) {
		return 0;
	}
	IMP imp = method_getImplementation(m);
	((BOOL (*)(id, SEL, id, BOOL))imp)(nil, sel, nil, NO);
	return 1;
}

// 模拟 wails 运行环境: AppDelegate 类不存在时创建之(真实环境由 wails 创建)
// 返回 0=就绪, -1=创建失败
static int test_ensure_appdelegate(void) {
	if (objc_getClass("AppDelegate") != NULL) {
		return 0;
	}
	Class cls = objc_allocateClassPair([NSObject class], "AppDelegate", 0);
	if (cls == NULL) {
		return -1;
	}
	objc_registerClassPair(cls);
	return 0;
}
*/
import "C"

import (
	"log"
	"sync"
)

var (
	showMu sync.RWMutex
	showFn func()
)

//export alchemyDockReopen
// 由 dockreopen_darwin.m 的 reopen IMP 调用(主线程); 转发到最新注册的业务回调
func alchemyDockReopen() {
	showMu.RLock()
	fn := showFn
	showMu.RUnlock()
	if fn != nil {
		fn()
	}
}

func install(show func()) {
	showMu.Lock()
	showFn = show
	showMu.Unlock()
	// 注册到 wails AppDelegate 类; 类不可用(启动早期)/已存在方法(未来 wails 版本)均跳过
	switch C.alchemy_dock_reopen_install() {
	case 0:
		// 已注册
	case 1:
		log.Printf("[dockreopen] AppDelegate 类不可用, Dock 重开处理未注册(启动顺序问题?)")
	case 2:
		// wails 已实现, 不覆盖
	}
}

// hasReopenMethod 测试辅助: wails AppDelegate 是否已注册 reopen 方法(-1/0/1)
func hasReopenMethod() int { return int(C.test_has_reopen_method()) }

// fireReopen 测试辅助: 直接触发 reopen IMP(0=未能触达, 1=已触发)
func fireReopen() int { return int(C.test_fire_reopen()) }

// ensureAppDelegate 测试夹具: 模拟 wails 的 AppDelegate 类(真实环境已存在则跳过)
func ensureAppDelegate() int { return int(C.test_ensure_appdelegate()) }
