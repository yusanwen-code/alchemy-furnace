#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════
# 炼丹炉 · macOS 桌面包验证器
#
# 校验 .app 包的架构与最低系统契约:
#   1. 主程序与 python-runtime/bin/python3 存在且可执行;
#   2. lipo 确认主程序与 python3 只含目标架构(arm64/x86_64),
#      不接受 Rosetta/Universal 冒充;
#   3. PlistBuddy 校验 CFBundleIdentifier、CFBundleExecutable、
#      CFBundleShortVersionString=version、LSMinimumSystemVersion=12.0;
#   4. embedded Python 报告 platform.machine()=目标架构,
#      且能 import app.main(runtime_python 固定 UTF-8 环境);
#   5. codesign --verify --deep --strict 通过,iconfile.icns 存在。
#
# 用法: scripts/verify-macos-package.sh <app-path> <arm64|x86_64> <version>
# 全部通过 exit 0;任一失败 exit 1 并打印原因。
# ═══════════════════════════════════════════════════════════

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/runtime-common.sh
source "$SCRIPT_DIR/lib/runtime-common.sh" # runtime_python

APP="${1:-}"
EXPECTED_ARCH="${2:-}"
VERSION="${3:-}"
if [[ -z "$APP" || -z "$EXPECTED_ARCH" || -z "$VERSION" ]]; then
  fail "用法: $0 <app-path> <arm64|x86_64> <version>"
fi
if [[ "$EXPECTED_ARCH" != "arm64" && "$EXPECTED_ARCH" != "x86_64" ]]; then
  fail "目标架构只能是 arm64 或 x86_64: $EXPECTED_ARCH"
fi

# PYBIN 一律用绝对路径(切子目录后依然有效)
[[ "$APP" == /* ]] || APP="$PWD/$APP"

CONTENTS="$APP/Contents"
RES="$CONTENTS/Resources"
PLIST="$CONTENTS/Info.plist"
MACOS_DIR="$CONTENTS/MacOS"
RUNTIME="$RES/python-runtime"
PYBIN="$RUNTIME/bin/python3"
ENGINE="$RUNTIME/engine"

FAILED=0
fail_check() {
  printf '❌ %s\n' "$*" >&2
  FAILED=1
}
ok() { printf 'ok   - %s\n' "$*"; }

# ─── 1. 包结构:主程序与 python3 存在且可执行 ───
[ -d "$APP" ] || fail_check "应用包不存在: $APP"
[ -f "$PLIST" ] || fail_check "缺少 Info.plist: $PLIST"

BIN_NAME="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "$PLIST" 2>/dev/null || true)"
[ -n "$BIN_NAME" ] || fail_check "Info.plist 缺少 CFBundleExecutable"
MAIN_BIN="$MACOS_DIR/$BIN_NAME"
if [ -f "$MAIN_BIN" ] && [ -x "$MAIN_BIN" ]; then
  ok "主程序存在且可执行: $MAIN_BIN"
else
  fail_check "主程序不存在或不可执行: $MAIN_BIN"
fi
if [ -f "$PYBIN" ] && [ -x "$PYBIN" ]; then
  ok "python-runtime/bin/python3 存在且可执行: $PYBIN"
else
  fail_check "python-runtime/bin/python3 不存在或不可执行: $PYBIN"
fi

# ─── 2. 架构:主程序与 python3 只含目标架构 ───
check_arch() { # check_arch <描述> <文件>
  local archs
  archs="$(lipo -archs "$2" 2>&1 || true)"
  if [[ "$archs" == "$EXPECTED_ARCH" ]]; then
    ok "$1 仅含 $EXPECTED_ARCH"
  else
    fail_check "$1 架构为 [$archs],期望仅 $EXPECTED_ARCH(Rosetta/Universal 视为失败)"
  fi
}
check_arch "主程序" "$MAIN_BIN"
check_arch "python-runtime/bin/python3" "$PYBIN"

# ─── 3. Plist 契约 ───
BUNDLE_ID="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$PLIST" 2>/dev/null || true)"
[ -n "$BUNDLE_ID" ] || fail_check "Info.plist 缺少 CFBundleIdentifier"

SHORT_VER="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$PLIST" 2>/dev/null || true)"
if [[ "$SHORT_VER" == "$VERSION" ]]; then
  ok "CFBundleShortVersionString=$SHORT_VER"
else
  fail_check "CFBundleShortVersionString=$SHORT_VER,期望 $VERSION"
fi

MIN_SYS="$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' "$PLIST" 2>/dev/null || true)"
if [[ "$MIN_SYS" == "12.0" ]]; then
  ok "LSMinimumSystemVersion=12.0"
else
  fail_check "LSMinimumSystemVersion=$MIN_SYS,期望 12.0(macOS 12+)"
fi

# ─── 4. embedded Python 自检:架构与 import ───
MACHINE="$(runtime_python "$PYBIN" -c 'import platform; print(platform.machine())' 2>&1 || true)"
if [[ "$MACHINE" == "$EXPECTED_ARCH" ]]; then
  ok "python3 platform.machine()=$MACHINE"
else
  fail_check "python3 platform.machine()=$MACHINE,期望 $EXPECTED_ARCH"
fi
[ -f "$ENGINE/app/main.py" ] || fail_check "engine/app/main.py 缺失: $ENGINE/app/main.py"
if ( cd "$ENGINE" && runtime_python "$PYBIN" -c 'import app.main' >/dev/null 2>&1 ); then
  ok "engine 目录下 import app.main 成功"
else
  fail_check "engine 目录下 import app.main 失败(PYBIN=$PYBIN)"
fi

# ─── 5. 签名与图标 ───
if codesign --verify --deep --strict "$APP" >/dev/null 2>&1; then
  ok "codesign --verify --deep --strict 通过"
else
  fail_check "codesign --verify --deep --strict 失败: $APP"
fi
ICON_NAME="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIconFile' "$PLIST" 2>/dev/null || true)"
ICON_PATH="$RES/${ICON_NAME%.icns}.icns"
if [ -s "$ICON_PATH" ]; then
  ok "应用图标存在: $ICON_PATH"
else
  fail_check "应用图标缺失或为空: $ICON_PATH"
fi

if [[ "$FAILED" -gt 0 ]]; then
  printf '❌ macOS 包验证失败: %s(%s %s)\n' "$APP" "$EXPECTED_ARCH" "$VERSION" >&2
  exit 1
fi
printf '✅ macOS 包验证通过: %s(%s %s)\n' "$APP" "$EXPECTED_ARCH" "$VERSION"
exit 0
