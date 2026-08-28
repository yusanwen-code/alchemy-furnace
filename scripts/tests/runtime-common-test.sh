#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════
# 炼丹炉 · runtime-common.sh 回归测试(无框架 bash,红→绿)
#
# 覆盖根因:
#   1. 相对输出路径在切换子目录后失效 → 必须规范化为绝对路径
#   2. 宽目录(空值、/、.、..、仓库根、backend、backend/go)必须被拒绝
#   3. Windows 编码问题 → runtime_python 必须固定 UTF-8 环境
#   4. 含空格路径未引用 → runtime_python 必须正确引用可执行文件与参数
#
# 用法: bash scripts/tests/runtime-common-test.sh
# ═══════════════════════════════════════════════════════════

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LIB="$SCRIPT_DIR/../lib/runtime-common.sh"

if [[ ! -f "$LIB" ]]; then
  printf 'FAIL: 缺少 %s(red: runtime-common.sh 尚未实现)\n' "$LIB"
  exit 1
fi
# shellcheck source=../lib/runtime-common.sh
source "$LIB"

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

contains() { # contains <描述> <内容> <片段>
  local desc="$1" hay="$2" needle="$3"
  if [[ "$hay" == *"$needle"* ]]; then
    PASS=$((PASS + 1))
    printf 'ok   - %s\n' "$desc"
  else
    FAILED=$((FAILED + 1))
    printf 'FAIL - %s(缺少片段: %s)\n      实际输出: %s\n' "$desc" "$needle" "$hay"
  fi
}

expect_reject() { # expect_reject <描述> <路径...>
  local desc="$1"
  shift
  if ( runtime_resolve_output_path "$@" >/dev/null 2>&1 ); then
    FAILED=$((FAILED + 1))
    printf 'FAIL - %s(应拒绝却被接受: %s)\n' "$desc" "$*"
  else
    PASS=$((PASS + 1))
    printf 'ok   - %s\n' "$desc"
  fi
}

# 保证环境断言确定性:先清掉可能从外部带入的变量
unset PYTHONUTF8 PYTHONIOENCODING PIP_DISABLE_PIP_VERSION_CHECK PYTHONDONTWRITEBYTECODE 2>/dev/null || true

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# ─── 1. 相对路径规范化为绝对路径 ───
cd "$TEST_ROOT"
REL_OUT="$(runtime_resolve_output_path "backend/go/dist-runtime")"
check "相对路径 backend/go/dist-runtime 规范化为绝对路径" "$REL_OUT" "$TEST_ROOT/backend/go/dist-runtime"

# ─── 2. 切换子目录后依然有效(返回的是绝对路径,不依赖 cwd)───
cd "$TEST_ROOT/backend/go"
if [[ "$REL_OUT" == /* && "$REL_OUT" == "$TEST_ROOT/backend/go/dist-runtime" ]]; then
  PASS=$((PASS + 1))
  printf 'ok   - 切换子目录后绝对路径依然有效\n'
else
  FAILED=$((FAILED + 1))
  printf 'FAIL - 切换子目录后路径失效: %s\n' "$REL_OUT"
fi

# ─── 3. 宽目录拒绝 ───
cd "$TEST_ROOT"
expect_reject "拒绝空值" ""
expect_reject "拒绝根目录 /" "/"
expect_reject "拒绝 ." "."
expect_reject "拒绝 .." ".."
expect_reject "拒绝仓库根(绝对路径)" "$TEST_ROOT"
expect_reject "拒绝仓库根(相对形式 ./)" "./"
expect_reject "拒绝 backend 目录" "$TEST_ROOT/backend"
expect_reject "拒绝 backend(相对形式)" "backend"
expect_reject "拒绝 backend/go 目录" "$TEST_ROOT/backend/go"
expect_reject "拒绝 backend/go(相对形式)" "backend/go"

# ─── 4. fake python:UTF-8 环境 + 含空格路径引用 + 参数透传 ───
mkdir -p "$TMP/fake bin"
FAKE_PY="$TMP/fake bin/fake python.sh"
cat > "$FAKE_PY" <<'EOF'
#!/usr/bin/env bash
printf 'PYTHONUTF8=%s\n' "${PYTHONUTF8:-unset}"
printf 'PYTHONIOENCODING=%s\n' "${PYTHONIOENCODING:-unset}"
printf 'PIP_DISABLE_PIP_VERSION_CHECK=%s\n' "${PIP_DISABLE_PIP_VERSION_CHECK:-unset}"
printf 'PYTHONDONTWRITEBYTECODE=%s\n' "${PYTHONDONTWRITEBYTECODE:-unset}"
printf 'ARG1=%s\n' "${1:-}"
EOF
chmod +x "$FAKE_PY"

FAKE_OUT="$(runtime_python "$FAKE_PY" "hello world")"
contains "PYTHONUTF8=1" "$FAKE_OUT" "PYTHONUTF8=1"
contains "PYTHONIOENCODING=utf-8" "$FAKE_OUT" "PYTHONIOENCODING=utf-8"
contains "PIP_DISABLE_PIP_VERSION_CHECK=1" "$FAKE_OUT" "PIP_DISABLE_PIP_VERSION_CHECK=1"
contains "PYTHONDONTWRITEBYTECODE=1" "$FAKE_OUT" "PYTHONDONTWRITEBYTECODE=1"
contains "含空格路径的 fake python 被正确执行并引用" "$FAKE_OUT" "PYTHONUTF8=1"
contains "参数透传(含空格内容)" "$FAKE_OUT" "ARG1=hello world"

# ─── 5. 退出码透传 ───
FAKE_EXIT3="$TMP/fake exit3.sh"
printf '#!/usr/bin/env bash\nexit 3\n' > "$FAKE_EXIT3"
chmod +x "$FAKE_EXIT3"
runtime_python "$FAKE_EXIT3" >/dev/null 2>&1
RC=$?
check "runtime_python 透传退出码" "$RC" "3"

# ─── 6. 环境变量不污染调用方 ───
if [[ -z "${PYTHONUTF8:-}" && -z "${PYTHONIOENCODING:-}" && -z "${PIP_DISABLE_PIP_VERSION_CHECK:-}" && -z "${PYTHONDONTWRITEBYTECODE:-}" ]]; then
  PASS=$((PASS + 1))
  printf 'ok   - runtime_python 不污染调用方环境\n'
else
  FAILED=$((FAILED + 1))
  printf 'FAIL - runtime_python 污染了调用方环境\n'
fi

printf '\n%d passed, %d failed\n' "$PASS" "$FAILED"
if [[ "$FAILED" -gt 0 ]]; then
  exit 1
fi
exit 0
