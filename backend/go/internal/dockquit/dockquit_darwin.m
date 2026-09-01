//go:build darwin

#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

extern void alchemyDockQuit(void);

// Wails v2 routes applicationShouldTerminate through the same Q message as a
// window close. Replace that delegate method so Dock's Quit action can enter
// the application's explicit shutdown path instead.
static NSApplicationTerminateReply alchemyDockQuitIMP(id self, SEL _cmd, NSApplication *sender) {
    alchemyDockQuit();
    return NSTerminateCancel;
}

int alchemy_dock_quit_install(void) {
    Class cls = objc_getClass("AppDelegate");
    if (cls == NULL) {
        return 1;
    }
    SEL sel = sel_registerName("applicationShouldTerminate:");
    // The Wails delegate already implements this selector. class_replaceMethod
    // also handles the test fixture, where it has not been defined yet.
    class_replaceMethod(cls, sel, (IMP)alchemyDockQuitIMP, "q@:@");
    return 0;
}
