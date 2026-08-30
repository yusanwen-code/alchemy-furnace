#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════
# 炼丹炉 · 桌面「显示名/技术名」双层命名静态契约测试
#
# 契约(2026-08-30-desktop-identity-tray-no-console-design):
#   显示名 = 炼丹炉(用户可见: 窗口标题/安装器/快捷方式/托盘)
#   技术名 = AlchemyFurnace(可执行文件/数据目录/更新资产, ASCII 固定)
#   更新 ZIP 根目录必须保持 AlchemyFurnace.app(旧版更新器硬依赖)
#
# 用法: bash scripts/tests/desktop-name-contract-test.sh
# 退出码: 0=通过 1=契约破坏
# ═══════════════════════════════════════════════════════════

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

FAILED=0
fail() { printf '❌ %s\n' "$*" >&2; FAILED=1; }
ok()   { printf 'ok   - %s\n' "$*"; }

# ─── 静态命名契约(wails.json / Info.plist / installer.nsi) ───
python3 - "$ROOT" <<'PY' || fail "静态命名契约检查失败"
import json, pathlib, plistlib, sys
root = pathlib.Path(sys.argv[1])

wails = json.loads((root / "backend/go/wails.json").read_text(encoding="utf-8"))
assert wails["name"] == "炼丹炉", f'wails name={wails["name"]!r}'
assert wails["outputfilename"] == "AlchemyFurnace", f'outputfilename={wails["outputfilename"]!r}'
assert wails["info"]["productName"] == "炼丹炉", f'productName={wails["info"]["productName"]!r}'
assert wails["info"]["companyName"] == "炼丹炉", f'companyName={wails["info"]["companyName"]!r}'

plist = plistlib.loads((root / "backend/go/build/darwin/Info.plist").read_bytes())
assert plist["CFBundleName"] == "炼丹炉", f'CFBundleName={plist["CFBundleName"]!r}'
assert plist["CFBundleDisplayName"] == "炼丹炉", f'CFBundleDisplayName={plist["CFBundleDisplayName"]!r}'
assert plist["CFBundleExecutable"] == "AlchemyFurnace", f'CFBundleExecutable={plist["CFBundleExecutable"]!r}'
assert plist["CFBundleIdentifier"] == "com.alchemyfurnace.desktop", f'CFBundleIdentifier={plist["CFBundleIdentifier"]!r}'

nsi = (root / "backend/go/build/windows/installer.nsi").read_text(encoding="utf-8")
assert nsi.splitlines()[0] == "# -*- coding: utf-8 -*-", "installer.nsi 第一行必须是 UTF-8 编码声明"
assert '!define PRODUCT_DISPLAY_NAME "炼丹炉"' in nsi, "installer.nsi 缺 PRODUCT_DISPLAY_NAME"
assert '!define PRODUCT_NAME "AlchemyFurnace"' in nsi, "installer.nsi 缺 PRODUCT_NAME"

# 开始菜单: 安装建中文目录, 卸载同时清理旧英文目录(兼容已发布版本)
assert 'CreateDirectory "$SMPROGRAMS\\${PRODUCT_DISPLAY_NAME}"' in nsi, "安装需创建中文开始菜单目录"
assert 'RMDir /r "$SMPROGRAMS\\${PRODUCT_DISPLAY_NAME}"' in nsi, "卸载需清理中文开始菜单目录"
assert 'RMDir /r "$SMPROGRAMS\\${PRODUCT_NAME}"' in nsi, "卸载需清理旧英文开始菜单目录"

# Windows 发布流程禁止 -windowsconsole(黑框回归保护)
workflow = (root / ".github/workflows/release-desktop.yml").read_text(encoding="utf-8")
assert "-windowsconsole" not in workflow, "release-desktop.yml 不得出现 -windowsconsole"
PY
if [ "$FAILED" -eq 0 ]; then
  ok "wails/Info.plist/installer.nsi/release-desktop.yml 静态命名契约"
fi

# ─── Release 资产名必须保持 ASCII 技术名(旧版更新器 SelectAsset 依赖) ───
for want in "AlchemyFurnace-mac-arm64.zip" "AlchemyFurnace-mac-x64.zip" "AlchemyFurnace-Setup.exe"; do
  if grep -qF "$want" "$ROOT/backend/go/internal/updater/updater.go"; then
    ok "updater.go 含资产名 $want"
  else
    fail "updater.go 缺少资产名 $want"
  fi
done

# ─── 数据目录名不可迁移(用户已有数据, 改名即丢) ───
if grep -qF 'appDirName = "AlchemyFurnace"' "$ROOT/backend/go/internal/paths/paths.go"; then
  ok "paths.go 数据目录名 AlchemyFurnace"
else
  fail "paths.go 数据目录名被改动"
fi

if [ "$FAILED" -gt 0 ]; then
  printf '❌ 桌面命名契约测试失败\n' >&2
  exit 1
fi
printf '✅ 桌面命名契约测试通过\n'
exit 0
