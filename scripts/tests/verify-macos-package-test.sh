#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════
# 炼丹炉 · verify-macos-package.sh 回归测试(无框架 bash,红→绿)
#
# 覆盖契约:
#   1. 缺主程序 → 失败
#   2. 缺 python-runtime/bin/python3 → 失败
#   3. 错误架构(原生架构 fixture 传另一架构)→ 失败,不接受 Rosetta/Universal 冒充
#   4. 错误版本(CFBundleShortVersionString ≠ version 参数)→ 失败
#   5. 最低系统 < 12(LSMinimumSystemVersion=11.0)→ 失败
#   6. 全通过(主程序/Python/版本/最低系统/codesign/图标)→ 通过
#
# fixture 完全不依赖仓库内 app:clang 编译目标架构主程序与假 python3
# (都是真 Mach-O,可被 lipo 检架构、可被执行),手写 Info.plist + iconfile.icns,
# codesign 自签整个 bundle。
#
# 用法: bash scripts/tests/verify-macos-package-test.sh
# ═══════════════════════════════════════════════════════════

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VERIFIER="$SCRIPT_DIR/../verify-macos-package.sh"

for tool in clang lipo codesign; do
  command -v "$tool" >/dev/null 2>&1 || { printf 'FAIL: 缺少 %s(无法构造 fixture)\n' "$tool"; exit 1; }
done
[ -x /usr/libexec/PlistBuddy ] || { printf 'FAIL: 缺少 PlistBuddy\n'; exit 1; }

if [[ ! -f "$VERIFIER" ]]; then
  printf 'FAIL: 缺少 %s(red: verify-macos-package.sh 尚未实现)\n' "$VERIFIER"
  exit 1
fi

PASS=0
FAILED=0

check() { # check <描述> <实际> <期望>
  local desc="$1" actual="$2" expected="$3"
  if [[ "$actual" == "$expected" ]]; then
    PASS=$((PASS + 1))
    printf 'ok   - %s\n' "$desc"
  else
    FAILED=$((FAILED + 1))
    printf 'FAIL - %s\n      期望: %s\n      实际: %s\n' "$desc" "$expected" "$actual"
  fi
}

expect_accept() { # expect_accept <描述> <args...>
  local desc="$1"
  shift
  if "$VERIFIER" "$@" >/dev/null 2>&1; then
    PASS=$((PASS + 1))
    printf 'ok   - %s\n' "$desc"
  else
    FAILED=$((FAILED + 1))
    printf 'FAIL - %s(应 exit 0 却失败)\n' "$desc"
  fi
}

expect_reject() { # expect_reject <描述> <原因片段> <args...>
  local desc="$1" frag="$2"
  shift 2
  local out rc
  out="$("$VERIFIER" "$@" 2>&1)"
  rc=$?
  if [[ "$rc" -eq 0 ]]; then
    FAILED=$((FAILED + 1))
    printf 'FAIL - %s(应 exit 1 却 exit 0)\n      输出: %s\n' "$desc" "$out"
  elif [[ "$out" == *"$frag"* ]]; then
    PASS=$((PASS + 1))
    printf 'ok   - %s\n' "$desc"
  else
    FAILED=$((FAILED + 1))
    printf 'FAIL - %s(exit %s,但未打印原因片段: %s)\n      输出: %s\n' "$desc" "$rc" "$frag" "$out"
  fi
}

# ─── fixture 构造 ───
NATIVE_ARCH="$(uname -m)"
if [[ "$NATIVE_ARCH" == "arm64" ]]; then
  OTHER_ARCH="x86_64"
else
  OTHER_ARCH="arm64"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# 假 python3:编译成目标架构的真实 Mach-O(可被 lipo 检架构、可被执行),
# 模拟真实 embedded Python 的 platform.machine() 与 import app.main 自检。
build_fake_python() { # build_fake_python <arch> <out>
  local arch="$1" out="$2"
  cat > "$TMP/fake-python.c" <<EOF
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>

static int has_needle(int argc, char **argv, const char *needle) {
  int i;
  for (i = 0; i < argc; i++) {
    if (strstr(argv[i], needle) != NULL) return 1;
  }
  return 0;
}

int main(int argc, char **argv) {
  if (has_needle(argc, argv, "platform.machine")) {
    printf("${arch}\n");
    return 0;
  }
  if (has_needle(argc, argv, "import app.main")) {
    struct stat st;
    if (stat("app/main.py", &st) != 0) {
      fprintf(stderr, "missing app/main.py\n");
      return 1;
    }
    return 0;
  }
  return 0;
}
EOF
  clang -arch "$arch" -o "$out" "$TMP/fake-python.c"
}

# 构造一个完整合法的 fixture bundle(主程序 + 假 python3 + plist + 图标 + 自签),
# 打印 bundle 路径到 stdout。
build_good_bundle() { # build_good_bundle <arch> <bundle-path>
  local arch="$1" app="$2"
  local contents="$app/Contents" res="$app/Contents/Resources" macos="$app/Contents/MacOS"
  local runtime="$res/python-runtime"
  local engine="$runtime/engine"
  mkdir -p "$macos" "$engine/app" "$runtime/bin"

  # 主程序:clang 编译目标架构(真 Mach-O)
  printf 'int main(void){return 0;}\n' | clang -arch "$arch" -x c - -o "$macos/AlchemyFurnace"

  # 假 python3
  build_fake_python "$arch" "$runtime/bin/python3"
  chmod +x "$runtime/bin/python3"

  # engine/app/main.py:最小可 import 模块(自检不输出密钥)
  printf 'VERSION = "fixture"\n' > "$engine/app/main.py"
  printf '' > "$engine/app/__init__.py"

  cat > "$contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleIdentifier</key>
    <string>com.alchemyfurnace.desktop</string>
    <key>CFBundleExecutable</key>
    <string>AlchemyFurnace</string>
    <key>CFBundleIconFile</key>
    <string>iconfile</string>
    <key>CFBundleShortVersionString</key>
    <string>0.1.1</string>
    <key>CFBundleVersion</key>
    <string>0.1.1</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
</dict>
</plist>
PLIST

  # 图标:非空即可
  printf 'icns-fixture' > "$res/iconfile.icns"

  codesign --force --sign - "$app" >/dev/null 2>&1 || { printf 'FAIL: fixture codesign 自签失败\n' >&2; return 1; }
  printf '%s\n' "$app"
}

GOOD="$TMP/good.app"
build_good_bundle "$NATIVE_ARCH" "$GOOD" || exit 1

# ─── 1. 缺主程序 ───
MISSING_BIN="$TMP/missing-bin.app"
cp -R "$GOOD" "$MISSING_BIN"
rm -f "$MISSING_BIN/Contents/MacOS/AlchemyFurnace"
expect_reject "缺主程序时拒绝" "主程序" "$MISSING_BIN" "$NATIVE_ARCH" "0.1.1"

# ─── 2. 缺 Python runtime ───
MISSING_PY="$TMP/missing-py.app"
cp -R "$GOOD" "$MISSING_PY"
rm -f "$MISSING_PY/Contents/Resources/python-runtime/bin/python3"
expect_reject "缺 python3 时拒绝" "python3" "$MISSING_PY" "$NATIVE_ARCH" "0.1.1"

# ─── 3. 错误架构(原生架构 fixture 传另一架构,Rosetta/Universal 不能冒充)───
expect_reject "错误架构时拒绝" "期望仅" "$GOOD" "$OTHER_ARCH" "0.1.1"

# ─── 4. 错误版本 ───
WRONG_VER="$TMP/wrong-version.app"
cp -R "$GOOD" "$WRONG_VER"
/usr/libexec/PlistBuddy -c 'Set :CFBundleShortVersionString 9.9.9' "$WRONG_VER/Contents/Info.plist" >/dev/null 2>&1
expect_reject "错误版本时拒绝" "CFBundleShortVersionString" "$WRONG_VER" "$NATIVE_ARCH" "0.1.1"

# ─── 5. 最低系统 < 12 ───
OLD_MIN="$TMP/old-min.app"
cp -R "$GOOD" "$OLD_MIN"
/usr/libexec/PlistBuddy -c 'Set :LSMinimumSystemVersion 11.0' "$OLD_MIN/Contents/Info.plist" >/dev/null 2>&1
expect_reject "最低系统 <12 时拒绝" "LSMinimumSystemVersion" "$OLD_MIN" "$NATIVE_ARCH" "0.1.1"

# ─── 6. 全通过 ───
expect_accept "全通过(主程序/Python/版本/最低系统/codesign/图标)" "$GOOD" "$NATIVE_ARCH" "0.1.1"

# ─── 7. 用法:缺参数 ───
expect_reject "缺参数时拒绝" "用法" ""

# ─── 8. 仓库内 Info.plist 契约:LSMinimumSystemVersion 固定 12.0 ───
REAL_PLIST="$TEST_ROOT/backend/go/build/darwin/Info.plist"
if [[ -f "$REAL_PLIST" ]]; then
  MIN_SYS="$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' "$REAL_PLIST" 2>/dev/null || true)"
  check "仓库 Info.plist LSMinimumSystemVersion=12.0" "$MIN_SYS" "12.0"
else
  FAILED=$((FAILED + 1))
  printf 'FAIL - 缺少仓库 Info.plist: %s\n' "$REAL_PLIST"
fi

printf '\n%d passed, %d failed\n' "$PASS" "$FAILED"
if [[ "$FAILED" -gt 0 ]]; then
  exit 1
fi
exit 0
