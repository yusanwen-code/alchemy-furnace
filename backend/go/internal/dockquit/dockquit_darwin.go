//go:build darwin

package dockquit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <objc/runtime.h>
#include <objc/message.h>
#import <Foundation/Foundation.h>

int alchemy_dock_quit_install(void);

static int test_fire_terminate(void) {
	Class cls = objc_getClass("AppDelegate");
	if (cls == NULL) {
		return 0;
	}
	SEL sel = sel_registerName("applicationShouldTerminate:");
	Method m = class_getInstanceMethod(cls, sel);
	if (m == NULL) {
		return 0;
	}
	IMP imp = method_getImplementation(m);
	((NSInteger (*)(id, SEL, id))imp)(nil, sel, nil);
	return 1;
}

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

import "sync"

var (
	quitMu sync.RWMutex
	quitFn func()
)

//export alchemyDockQuit
func alchemyDockQuit() {
	quitMu.RLock()
	fn := quitFn
	quitMu.RUnlock()
	if fn != nil {
		fn()
	}
}

func install(quit func()) {
	quitMu.Lock()
	quitFn = quit
	quitMu.Unlock()
	_ = C.alchemy_dock_quit_install()
}

func fireTerminate() int { return int(C.test_fire_terminate()) }

func ensureAppDelegate() int { return int(C.test_ensure_appdelegate()) }
