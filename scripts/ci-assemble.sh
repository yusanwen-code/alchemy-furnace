#!/usr/bin/env bash
# Assemble target-native Wails output and Python runtime into release assets.
# Usage: scripts/ci-assemble.sh <darwin-arm64|darwin-amd64|windows-amd64> [version]

set -euo pipefail

PLATFORM="${1:?用法: $0 <darwin-arm64|darwin-amd64|windows-amd64> [version]}"
VERSION="${2:-${VERSION:-dev}}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/runtime-common.sh
source "$ROOT/scripts/lib/runtime-common.sh"
# Override is intended for isolated packaging tests and downstream reuse.
GO_DIR="${ALCHEMY_GO_DIR:-$ROOT/backend/go}"
BIN_DIR="$GO_DIR/build/bin"
DIST_DIR="$GO_DIR/build/dist"
RUNTIME_SRC="$GO_DIR/dist-runtime"

say()  { printf '\033[36m[assemble]\033[0m %s\n' "$*"; }
fail() { printf '\033[31m[assemble] ❌ %s\033[0m\n' "$*" >&2; exit 1; }

mkdir -p "$DIST_DIR"
[ -d "$RUNTIME_SRC/engine/app" ] || fail "运行时缺少 engine/app: $RUNTIME_SRC"

case "$PLATFORM" in
  darwin-arm64|darwin-amd64)
    APP="$BIN_DIR/AlchemyFurnace.app"
    [ -x "$APP/Contents/MacOS/AlchemyFurnace" ] || fail "未找到 Wails 应用: $APP"
    [ -x "$RUNTIME_SRC/bin/python3" ] || fail "运行时缺少 bin/python3: $RUNTIME_SRC"

    case "$PLATFORM" in
      darwin-arm64) ASSET_ARCH="arm64" ;;
      darwin-amd64) ASSET_ARCH="x64" ;;
    esac

    RES="$APP/Contents/Resources"
    RUNTIME_DST="$RES/python-runtime"
    rm -rf "$RUNTIME_DST"
    mkdir -p "$RUNTIME_DST"
    say "装入 Python 运行时 → $RUNTIME_DST"
    cp -R "$RUNTIME_SRC/." "$RUNTIME_DST/"

    # Wails build happens before runtime assembly. Re-sign afterwards or the
    # bundle's original signature becomes invalid as soon as Resources changes.
    PLIST="$APP/Contents/Info.plist"
    ICON_NAME="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIconFile' "$PLIST" 2>/dev/null || true)"
    [ -n "$ICON_NAME" ] || fail "Info.plist 缺少 CFBundleIconFile"
    ICON_PATH="$RES/${ICON_NAME%.icns}.icns"
    [ -s "$ICON_PATH" ] || fail "应用图标缺失或为空: $ICON_PATH"
    BUNDLE_VERSION="${VERSION#v}"
    BUNDLE_VERSION="${BUNDLE_VERSION%%-*}"
    if [[ ! "$BUNDLE_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      BUNDLE_VERSION="0.0.0"
    fi
    /usr/libexec/PlistBuddy -c "Set :CFBundleVersion $BUNDLE_VERSION" "$PLIST"
    /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $BUNDLE_VERSION" "$PLIST"

    # 版本字节断言(ldflags -X 注入):macOS verifier 只看 Info.plist,
    # 查不出二进制丢版本字节——rc.3 三平台版本缺失的教训,release 构建在此拦截。
    # 仅语义版本生效,本地 dev 构建(无注入)跳过。
    if [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
      assert_version_embedded "$APP/Contents/MacOS/AlchemyFurnace" "$VERSION"
    fi

    SIGN_IDENTITY="${MACOS_SIGN_IDENTITY:--}"
    say "签名应用(identity: $SIGN_IDENTITY)"
    codesign --force --deep --sign "$SIGN_IDENTITY" "$APP"
    codesign --verify --deep --strict --verbose=2 "$APP"

    # 平台 verifier(设计 §5 macOS):架构由平台参数推导,失败即退出
    # 注意:case 嵌套在 $(...) 里会被 bash 解析器误读,用 if 映射
    VERIFY_ARCH="arm64"
    if [[ "$PLATFORM" == darwin-amd64 ]]; then
      VERIFY_ARCH="x86_64"
    fi
    say "验证 macOS 包(架构 $VERIFY_ARCH,版本 $BUNDLE_VERSION)"
    "$ROOT/scripts/verify-macos-package.sh" "$APP" "$VERIFY_ARCH" "$BUNDLE_VERSION"

    DMG="$DIST_DIR/AlchemyFurnace-mac-${ASSET_ARCH}.dmg"
    ZIP="$DIST_DIR/AlchemyFurnace-mac-${ASSET_ARCH}.zip"
    STAGE="$(mktemp -d)"
    trap 'rm -rf "$STAGE"' EXIT
    cp -R "$APP" "$STAGE/"
    ln -s /Applications "$STAGE/Applications"
    say "生成 $(basename "$DMG")"
    rm -f "$DMG"
    # hdiutil create 偶发 "Resource busy"(rc.4 darwin 打包实测,磁盘映像服务
    # 瞬时占用),环境性失败——重试 3 次(间隔 5s)再放弃;真·失败原样报错。
    retry 3 5 hdiutil create -volname "AlchemyFurnace" -srcfolder "$STAGE" \
      -ov -format UDZO "$DMG" >/dev/null \
      || fail "hdiutil create 连续 3 次失败: $DMG"

    say "生成 $(basename "$ZIP")"
    rm -f "$ZIP"
    ditto -c -k --sequesterRsrc --keepParent "$APP" "$ZIP"

    # ZIP 根目录只能有一个 AlchemyFurnace.app(更新器按根目录解包)
    ZIP_APPS="$(unzip -l "$ZIP" | awk '{print $4}' | sed -n 's#^\(AlchemyFurnace\.app\)/.*#\1#p' | sort -u)"
    if [[ "$ZIP_APPS" != "AlchemyFurnace.app" ]]; then
      fail "ZIP 根目录应只有一个 AlchemyFurnace.app,实际: [$ZIP_APPS]"
    fi
    say "ZIP 根目录只有一个 AlchemyFurnace.app"
    ;;

  windows-amd64)
    EXE="$BIN_DIR/AlchemyFurnace.exe"
    [ -f "$EXE" ] || fail "未找到 Wails 可执行文件: $EXE"
    [ -f "$RUNTIME_SRC/python.exe" ] || fail "运行时缺少 python.exe: $RUNTIME_SRC"
    command -v makensis >/dev/null 2>&1 || fail "未找到 makensis(NSIS)"

    PACKAGE_DIR="$GO_DIR/build/package/windows"
    rm -rf "$PACKAGE_DIR"
    mkdir -p "$PACKAGE_DIR/runtime"
    cp "$EXE" "$PACKAGE_DIR/AlchemyFurnace.exe"
    cp -R "$RUNTIME_SRC/." "$PACKAGE_DIR/runtime/"

    # 版本字节断言:Windows verifier 虽查,但尽早失败省一次 NSIS 打包;
    # 与 darwin 段同一道防线(ldflags 注入防回归)。
    if [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
      assert_version_embedded "$EXE" "$VERSION"
    fi

    INSTALLER="$DIST_DIR/AlchemyFurnace-Setup.exe"
    rm -f "$INSTALLER"
    say "生成 $(basename "$INSTALLER")(包含 runtime/)"
    # makensis 默认 chdir 到脚本所在目录(无 /NOCD),相对路径会基于
    # build/windows/ 解析 → 必须传 Win32 绝对路径(cygpath -w 转换 MSYS 路径)
    APP_SOURCE_WIN="$(cygpath -w "$PACKAGE_DIR" 2>/dev/null || true)"
    OUTPUT_FILE_WIN="$(cygpath -w "$INSTALLER" 2>/dev/null || true)"
    if [[ -z "$APP_SOURCE_WIN" || -z "$OUTPUT_FILE_WIN" ]]; then
      fail "需要 cygpath 把输出路径转为 Windows 绝对路径(Git Bash 环境)"
    fi
    (
      cd "$GO_DIR"
      makensis \
        -DAPP_SOURCE="$APP_SOURCE_WIN" \
        -DOUTPUT_FILE="$OUTPUT_FILE_WIN" \
        build/windows/installer.nsi
    )
    [ -s "$INSTALLER" ] || fail "NSIS 未生成安装器: $INSTALLER"

    # 平台 verifier(设计 §5 Windows):pwsh 优先,回退 Windows PowerShell;失败即退出
    if command -v pwsh >/dev/null 2>&1; then
      PS_RUNNER="pwsh"
    elif command -v powershell >/dev/null 2>&1; then
      PS_RUNNER="powershell"
    else
      fail "未找到 pwsh 或 powershell,无法验证 Windows 包"
    fi
    say "验证 Windows x64 包(版本 $VERSION)"
    "$PS_RUNNER" -NoProfile -ExecutionPolicy Bypass -File "$ROOT/scripts/verify-windows-package.ps1" \
      -PackageDir "$PACKAGE_DIR" \
      -Installer "$INSTALLER" \
      -ExpectedVersion "$VERSION"
    ;;

  *) fail "未知平台: $PLATFORM" ;;
esac

say "✅ $PLATFORM ${VERSION} 产物:"
find "$DIST_DIR" -maxdepth 1 -type f -name 'AlchemyFurnace-*' -print
