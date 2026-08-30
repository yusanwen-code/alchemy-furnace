//go:build darwin

#import <Cocoa/Cocoa.h>

// Go 侧 //export 的回调(cgo 链接期提供)
extern void alchemyTrayOpen(void);
extern void alchemyTrayQuit(void);

// 菜单 action target: NSMenuItem.target 是 weak 引用, 必须 static 强引用保活
@interface AlchemyTrayActionTarget : NSObject
@end

@implementation AlchemyTrayActionTarget
- (void)openTray:(id)sender { alchemyTrayOpen(); }
- (void)quitTray:(id)sender { alchemyTrayQuit(); }
@end

static NSStatusItem *gStatusItem = nil;
static AlchemyTrayActionTarget *gTarget = nil;

// 主线程上实际创建(调用方保证已到主线程); 0=成功, 非零=失败
static void trayStartOnMain(const unsigned char *icon, int icon_len, int *result) {
    NSData *data = [NSData dataWithBytes:icon length:(NSUInteger)icon_len];
    NSImage *img = [[NSImage alloc] initWithData:data];
    if (img == nil) {
        *result = 1;
        return;
    }
    [img setSize:NSMakeSize(22.0, 22.0)];
    [img setTemplate:YES]; // 菜单栏深浅色自动反色

    NSStatusItem *item = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
    [item.button setImage:img];

    gTarget = [[AlchemyTrayActionTarget alloc] init];
    NSMenu *menu = [[NSMenu alloc] init];
    NSMenuItem *open = [menu addItemWithTitle:@"打开炼丹炉" action:@selector(openTray:) keyEquivalent:@""];
    [open setTarget:gTarget];
    [menu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *quit = [menu addItemWithTitle:@"退出炼丹炉" action:@selector(quitTray:) keyEquivalent:@""];
    [quit setTarget:gTarget];
    [item setMenu:menu];

    gStatusItem = item;
    *result = 0;
}

// 主线程上移除状态项并释放引用
static void trayStopOnMain(void) {
    if (gStatusItem != nil) {
        [[NSStatusBar systemStatusBar] removeStatusItem:gStatusItem];
        gStatusItem = nil;
    }
    gTarget = nil;
}

int alchemy_tray_start(const unsigned char *icon, int icon_len) {
    __block int result = 1;
    if ([NSThread isMainThread]) {
        // 主线程调用时直接执行, 不得等待 semaphore(等自己=死锁)
        trayStartOnMain(icon, icon_len, &result);
        return result;
    }
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);
    dispatch_async(dispatch_get_main_queue(), ^{
        trayStartOnMain(icon, icon_len, &result);
        dispatch_semaphore_signal(sem);
    });
    // 最长 3 秒等待主线程完成; 超时=失败, 供 Go 侧降级为关闭即退出
    if (dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, 3 * NSEC_PER_SEC)) != 0) {
        return 2;
    }
    return result;
}

void alchemy_tray_stop(void) {
    if ([NSThread isMainThread]) {
        trayStopOnMain();
        return;
    }
    dispatch_async(dispatch_get_main_queue(), ^{
        trayStopOnMain();
    });
}
