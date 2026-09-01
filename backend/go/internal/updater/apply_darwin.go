//go:build darwin

// apply_darwin.go - macOS 原位替换更新
// 流程: 等待旧进程退出 → 备份旧版(.old) → 移入新版 → 失败回滚 → open 重启
package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// appBundlePathFromExe 由可执行文件路径上溯 .app 根
// 例: /Applications/AlchemyFurnace.app/Contents/MacOS/AlchemyFurnace → /Applications/AlchemyFurnace.app
// 非 .app 内运行(开发模式)返回错误,不应用更新
func appBundlePathFromExe(exe string) (string, error) {
	i := strings.Index(exe, ".app/")
	if i < 0 {
		return "", fmt.Errorf("updater: 非 .app 内运行,不应用更新: %s", exe)
	}
	appPath := exe[:i+len(".app")]
	// Finder 直接从 DMG/下载目录启动且应用带隔离属性时，macOS 会把它
	// 映射到 /private/var/.../AppTranslocation/...。该目录不可写，原位
	// 交换必然失败，必须在下载前阻止该操作并指导用户正确安装。
	if strings.Contains(filepath.ToSlash(appPath), "/AppTranslocation/") {
		return "", fmt.Errorf("%w：请先退出炼丹炉，将“炼丹炉.app”拖入“应用程序”目录后重新打开，再检查更新", ErrAppTranslocated)
	}
	return appPath, nil
}

// ValidateUpdateTarget 在下载前确认当前安装位置可执行原位更新。
func ValidateUpdateTarget() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("updater: 获取可执行路径失败: %w", err)
	}
	_, err = appBundlePathFromExe(exe)
	return err
}

// swapScript 等待旧进程退出 → 备份旧版 → 移入新版(失败回滚) → open 重启
//
// pid: 旧进程 ID(等它退出再 swap)
// newApp: 新版 .app 临时目录路径
// appPath: 目标 .app 路径(被替换的旧版)
func swapScript(pid int, newApp, appPath string) string {
	// 顺序: 等死 → 备份旧 → 移新(失败回滚) → 删 .old → open
	// 用 %d/%s/%s 格式化(后两参数 newApp/appPath)
	return fmt.Sprintf(`#!/bin/bash
set -e
APP_PATH=%q
NEW_APP=%q
OLD_PID=%d

# 等待旧进程退出(主进程已 os.Exit(0),但 wails webview 子进程可能未死)
while kill -0 $OLD_PID 2>/dev/null; do sleep 0.2; done
sleep 0.3

# 备份旧版
rm -rf "$APP_PATH.old"
mv "$APP_PATH" "$APP_PATH.old"

# 移入新版(失败回滚)
if ! mv "$NEW_APP" "$APP_PATH"; then
  mv "$APP_PATH.old" "$APP_PATH"
  exit 1
fi

# 清理旧版 + 重启
rm -rf "$APP_PATH.old"
open "$APP_PATH"
`, appPath, newApp, pid)
}

// findExtractedApp 在解压目录中定位受支持的 app bundle
// 新发布 DMG 使用 炼丹炉.app,更新 ZIP 根目录保持 AlchemyFurnace.app(旧版
// 更新器硬依赖);新版防御性接受两者,但解压目录中必须恰好存在一个
func findExtractedApp(root string) (string, error) {
	var found []string
	for _, name := range []string{"炼丹炉.app", "AlchemyFurnace.app"} {
		candidate := filepath.Join(root, name)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			found = append(found, candidate)
		}
	}
	if len(found) != 1 {
		return "", fmt.Errorf("updater: 解压目录中应有且仅有一个受支持的 app bundle,实际 %d 个", len(found))
	}
	return found[0], nil
}

// ApplyAndRestart macOS 实现:
// 1. 定位当前 .app 路径
// 2. 解压新 zip 到临时目录
// 3. 生成 swap 脚本 → 写文件 → exec 启动
// 4. 主进程 os.Exit(0) 让出文件句柄
func ApplyAndRestart(ctx context.Context, zipPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("updater: 获取可执行路径失败: %w", err)
	}
	appPath, err := appBundlePathFromExe(exe)
	if err != nil {
		return err
	}

	// 解压 zip → 临时目录
	tmpDir, err := os.MkdirTemp("", "alchemy-update-*")
	if err != nil {
		return fmt.Errorf("updater: 创建临时目录失败: %w", err)
	}
	if err := unzipToDir(ctx, zipPath, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("updater: 解压失败: %w", err)
	}
	// 解压产物必须是 炼丹炉.app 或旧版 AlchemyFurnace.app(且只能有一个)
	newApp, err := findExtractedApp(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return err
	}

	// 生成 + 启动 swap 脚本
	pid := os.Getpid()
	script := swapScript(pid, newApp, appPath)
	scriptPath := filepath.Join(tmpDir, "swap.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("updater: 写 swap 脚本失败: %w", err)
	}

	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("updater: 启动 swap 脚本失败: %w", err)
	}

	// 父进程退出,让 swap 脚本接管
	os.Exit(0)
	return nil
}
