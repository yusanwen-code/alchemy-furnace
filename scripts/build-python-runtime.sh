#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════
# 炼丹炉 · 内嵌 Python 运行时打包
#
# 用 python-build-standalone 拉取免安装 CPython + 预装业务依赖 +
# 拷贝 engine 源码,产出 <RUNTIME_ROOT>/(可执行 + 可 import)
#
# 用法: scripts/build-python-runtime.sh <darwin-arm64|darwin-amd64|windows-amd64> <运行时根目录>
# 消费: 任务 12 CI matrix + 本地打包脚本
# ═══════════════════════════════════════════════════════════

set -euo pipefail

PLATFORM="${1:?用法: $0 <darwin-arm64|darwin-amd64|windows-amd64> <输出目录>}"
RUNTIME_ROOT="${2:?用法: $0 <平台> <运行时根目录>}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# 跨平台公共库:绝对路径规范化 + 固定 UTF-8 环境运行 Python
# shellcheck source=lib/runtime-common.sh
source "$ROOT/scripts/lib/runtime-common.sh"

# 输出目录一律规范化为绝对路径(含宽目录拒绝:空值、/、.、..、仓库根等),
# 必须在 mkdir/mktemp 之前完成,否则失败时已创建目录无法回滚。
RUNTIME_ROOT="$(runtime_resolve_output_path "$RUNTIME_ROOT")"

# ─── 版本钉死(升级时同步到 astral-sh/python-build-standalone releases)───
PBS_TAG="20250712"
PBS_VER="3.12.11"
case "$PLATFORM" in
  darwin-arm64)
    PBS_TARGET="aarch64-apple-darwin"
    # 20250712 release 实测 SHA256(可被 PBS_SHA256_DARWIN_ARM64 等 env 覆盖)
    PBS_SHA256="${PBS_SHA256_DARWIN_ARM64:-8e8c0c478feefefdfb851d834f87fddb155f9eaf90694cd5a370399e6a8572aa}"
    PYBIN_BIN="bin/python3"
    ;;
  darwin-amd64)
    PBS_TARGET="x86_64-apple-darwin"
    PBS_SHA256="${PBS_SHA256_DARWIN_AMD64:-ec64e598489e59aa8fee5601a1fbca7c3abf6be854fddb5905c7a1488147daa7}"
    PYBIN_BIN="bin/python3"
    ;;
  windows-amd64)
    PBS_TARGET="x86_64-pc-windows-msvc"
    PBS_SHA256="${PBS_SHA256_WINDOWS_AMD64:-1ec21ef425f7eb596aae46143720ba91fe080d97fcff56a7fec81b9fb97d0024}"
    PYBIN_BIN="python.exe"
    ;;
  *) printf '未知平台: %s\n(支持: darwin-arm64|darwin-amd64|windows-amd64)\n' "$PLATFORM" >&2; exit 1 ;;
esac

PBS_NAME="cpython-${PBS_VER}+${PBS_TAG}-${PBS_TARGET}-install_only.tar.gz"
PBS_URL="https://github.com/astral-sh/python-build-standalone/releases/download/${PBS_TAG}/${PBS_NAME}"

# shasum 是 macOS 自带;Linux 才有 sha256sum
SHA256SUM="sha256sum"
command -v shasum >/dev/null 2>&1 && SHA256SUM="shasum -a 256"

say()   { printf '\033[36m[runtime]\033[0m %s\n' "$*"; }
fail()  { printf '\033[31m[runtime] ❌ %s\033[0m\n' "$*" >&2; exit 1; }

# ─── 1. 临时工作区(失败自动清理)───
RUNTIME_PARENT="$(dirname "$RUNTIME_ROOT")"
mkdir -p "$RUNTIME_PARENT"
WORK="$(mktemp -d)"
STAGED_RUNTIME="$(mktemp -d "${RUNTIME_ROOT}.tmp.XXXXXX")"
trap 'rm -rf "$WORK" "$STAGED_RUNTIME"' EXIT

say "下载 $PBS_NAME"
curl -fL "$PBS_URL" -o "$WORK/pbs.tar.gz" || fail "下载失败: $PBS_URL"
say "校验 SHA256"
echo "$PBS_SHA256  $WORK/pbs.tar.gz" | $SHA256SUM -c - || fail "SHA256 校验失败(请更新 PBS_SHA256 常量)"

say "解压到临时运行时 $STAGED_RUNTIME"
tar -xzf "$WORK/pbs.tar.gz" -C "$STAGED_RUNTIME" --strip-components=1

PYBIN="$STAGED_RUNTIME/$PYBIN_BIN"
[ -x "$PYBIN" ] || fail "未找到解释器: $PYBIN"

say "安装生产依赖(backend/python/requirements-runtime.txt)"
runtime_python "$PYBIN" -m pip install --no-cache-dir -r "$ROOT/backend/python/requirements-runtime.txt" \
  || fail "pip install 失败"

say "瘦身: 清理 __pycache__"
find "$STAGED_RUNTIME" -type d -name '__pycache__' -prune -exec rm -rf {} + 2>/dev/null || true

say "拷贝引擎源码到 engine/"
mkdir -p "$STAGED_RUNTIME/engine"
cp -R "$ROOT/backend/python/app" "$STAGED_RUNTIME/engine/"

say "自检: 引擎可 import + UTF-8 模式生效"
# STAGED_RUNTIME 由绝对 RUNTIME_ROOT 生成,PYBIN 为绝对路径;
# 自检只输出解释器路径/UTF-8 模式/import 结果,不输出密钥或环境变量。
( cd "$STAGED_RUNTIME/engine" && unset LOG_FORMAT LOG_LEVEL RUST_LOG &&
    runtime_python "$PYBIN" -c "import sys, app.main; print('import ok'); print('sys.executable=' + sys.executable); print('utf8_mode=' + str(sys.flags.utf8_mode)); assert sys.flags.utf8_mode == 1, 'PYTHONUTF8 未生效'" ) \
  || fail "引擎 import 失败"

# 自检和 pip 可能加载模块；发布运行时不携带可变 bytecode 缓存。
find "$STAGED_RUNTIME" -type d -name '__pycache__' -prune -exec rm -rf {} + 2>/dev/null || true

say "原子替换运行时: $RUNTIME_ROOT"
rm -rf "$RUNTIME_ROOT"
mv "$STAGED_RUNTIME" "$RUNTIME_ROOT"

say "✅ 完成: $RUNTIME_ROOT"
