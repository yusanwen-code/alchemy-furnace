//go:build darwin

#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

// Go 侧 //export 的回调(链接期由 cgo 提供)
extern void alchemyDockReopen(void);

// 动态注册到 wails AppDelegate 类的 applicationShouldHandleReopen:hasVisibleWindows:
// 实现。IMP 不访问 self/sender(测试可传 nil 直接调用); 返回 YES 让系统继续默认激活逻辑。
static BOOL alchemyDockReopenIMP(id self, SEL _cmd, NSApplication *sender, BOOL hasVisibleWindows) {
    alchemyDockReopen();
    return YES;
}

int alchemy_dock_reopen_install(void) {
    Class cls = objc_getClass("AppDelegate");
    if (cls == NULL) {
        return 1; // wails 尚未创建 delegate 类(OnStartup 之前调用)
    }
    SEL sel = sel_registerName("applicationShouldHandleReopen:hasVisibleWindows:");
    if (class_getInstanceMethod(cls, sel) != NULL) {
        return 2; // wails 未来版本已实现, 不覆盖其行为
    }
    // BOOL(id, SEL, NSApplication*, BOOL) 类型编码: c@:@c (BOOL=signed char)
    class_addMethod(cls, sel, (IMP)alchemyDockReopenIMP, "c@:@c");
    return 0;
}
