#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════
# 炼丹炉 · verify-release-assets.sh 回归测试(无框架 bash,红→绿)
#
# 覆盖根因:
#   1. 缺资产(5 个平台二进制 / checksums.txt / release-manifest.json)
#   2. 空文件
#   3. 重复架构(同架构出现第二份副本)
#   4. SHA256 不匹配(checksums.txt 或 manifest 与实际文件不符)
#   5. manifest 版本不符 / 非白名单文件 / 参数缺失
#   6. --binaries-only 阶段(生成 checksums/manifest 前的 5 包验证)
#
# 用法: bash scripts/tests/verify-release-assets-test.sh
# ═══════════════════════════════════════════════════════════

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERIFY="$SCRIPT_DIR/../verify-release-assets.sh"

if [[ ! -f "$VERIFY" ]]; then
  printf 'FAIL: 缺少 %s(red: verify-release-assets.sh 尚未实现)\n' "$VERIFY"
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

expect_fail() { # expect_fail <描述> <fixture-dir> <版本> <错误片段...>
  local desc="$1" dir="$2" version="$3"
  shift 3
  local out rc ok
  out="$(bash "$VERIFY" "$dir" "$version" 2>&1)"
  rc=$?
  if [[ $rc -eq 0 ]]; then
    FAILED=$((FAILED + 1))
    printf 'FAIL - %s(应失败却通过)\n' "$desc"
    return
  fi
  ok=1
  for needle in "$@"; do
    if [[ "$out" != *"$needle"* ]]; then
      ok=0
      printf '     缺少错误片段: %s\n' "$needle"
    fi
  done
  if [[ $ok -eq 1 ]]; then
    PASS=$((PASS + 1))
    printf 'ok   - %s\n' "$desc"
  else
    FAILED=$((FAILED + 1))
    printf 'FAIL - %s\n      实际输出: %s\n' "$desc" "$out"
  fi
}

hash_of() { # hash_of <file>
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

ASSET_NAMES=(
  AlchemyFurnace-mac-arm64.dmg
  AlchemyFurnace-mac-arm64.zip
  AlchemyFurnace-mac-x64.dmg
  AlchemyFurnace-mac-x64.zip
  AlchemyFurnace-Setup.exe
)

# 构造一个完整合法 fixture:5 个非空二进制 + checksums.txt + release-manifest.json
build_fixture() { # build_fixture <dir> <version>
  local dir="$1" version="$2" n
  for n in "${ASSET_NAMES[@]}"; do
    printf 'fixture-%s\n' "$n" > "$dir/$n"
  done
  : > "$dir/checksums.txt"
  for n in "${ASSET_NAMES[@]}"; do
    printf '%s  %s\n' "$(hash_of "$dir/$n")" "$n" >> "$dir/checksums.txt"
  done
  python3 - "$dir" "$version" <<'PY'
import json, sys, hashlib, os
d, version = sys.argv[1], sys.argv[2]
assets = {}
for n in sorted(os.listdir(d)):
    if n in ("checksums.txt", "release-manifest.json"):
        continue
    p = os.path.join(d, n)
    assets[n] = {
        "size": os.path.getsize(p),
        "sha256": hashlib.sha256(open(p, "rb").read()).hexdigest(),
    }
manifest = {
    "version": version,
    "tag": "v" + version,
    "created_at": "2026-08-28T00:00:00Z",
    "build": {"repository": "example/repo", "run_id": "1", "run_number": "1", "sha": "abc123"},
    "assets": assets,
}
json.dump(manifest, open(os.path.join(d, "release-manifest.json"), "w"), indent=2)
PY
}

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# ─── 1. 全通过 ───
GOOD="$TMP/good"
mkdir -p "$GOOD"
build_fixture "$GOOD" "0.2.0"
out="$(bash "$VERIFY" "$GOOD" "0.2.0" 2>&1)"
check "全通过:exit 0" "$?" "0"
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
contains "全通过:输出 PASS" "$out" "PASS"
out="$(bash "$VERIFY" "$GOOD" "v0.2.0" 2>&1)"
check "全通过:版本带 v 前缀可接受" "$?" "0"

# ─── 2. 缺资产 ───
MISS="$TMP/missing"
mkdir -p "$MISS"
build_fixture "$MISS" "0.2.0"
rm "$MISS/AlchemyFurnace-Setup.exe"
expect_fail "缺资产:Setup.exe" "$MISS" "0.2.0" "AlchemyFurnace-Setup.exe"

MISS2="$TMP/missing-checksums"
mkdir -p "$MISS2"
build_fixture "$MISS2" "0.2.0"
rm "$MISS2/checksums.txt"
expect_fail "缺 checksums.txt" "$MISS2" "0.2.0" "checksums.txt"

# ─── 3. 空文件 ───
EMPTY="$TMP/empty"
mkdir -p "$EMPTY"
build_fixture "$EMPTY" "0.2.0"
: > "$EMPTY/AlchemyFurnace-mac-x64.zip"
expect_fail "空文件:mac-x64.zip" "$EMPTY" "0.2.0" "AlchemyFurnace-mac-x64.zip" "空"

# ─── 4. 重复架构 ───
DUP="$TMP/dup"
mkdir -p "$DUP"
build_fixture "$DUP" "0.2.0"
printf 'extra copy\n' > "$DUP/AlchemyFurnace-mac-arm64.1.zip"
expect_fail "重复架构:arm64 第二份" "$DUP" "0.2.0" "重复"

# ─── 5. SHA256 不匹配 ───
HASHBAD="$TMP/hashbad"
mkdir -p "$HASHBAD"
build_fixture "$HASHBAD" "0.2.0"
printf 'corrupted\n' >> "$HASHBAD/AlchemyFurnace-mac-arm64.dmg"
expect_fail "SHA256 不符:mac-arm64.dmg" "$HASHBAD" "0.2.0" "AlchemyFurnace-mac-arm64.dmg" "SHA256"

# ─── 6. manifest 版本不符 ───
VERBAD="$TMP/verbad"
mkdir -p "$VERBAD"
build_fixture "$VERBAD" "0.2.0"
python3 - "$VERBAD" <<'PY'
import json, sys
p = sys.argv[1] + "/release-manifest.json"
m = json.load(open(p, encoding="utf-8"))
m["version"] = "9.9.9"
json.dump(m, open(p, "w"), indent=2)
PY
expect_fail "manifest 版本不符" "$VERBAD" "0.2.0" "版本不符"

# ─── 7. 非白名单文件 ───
EXTRA="$TMP/extra"
mkdir -p "$EXTRA"
build_fixture "$EXTRA" "0.2.0"
printf 'x\n' > "$EXTRA/AlchemyFurnace-linux-x64.deb"
expect_fail "非白名单文件" "$EXTRA" "0.2.0" "AlchemyFurnace-linux-x64.deb"

# ─── 8. --binaries-only:生成元数据前只验 5 包 ───
BIN="$TMP/binonly"
mkdir -p "$BIN"
for n in "${ASSET_NAMES[@]}"; do
  printf 'fixture-%s\n' "$n" > "$BIN/$n"
done
out="$(bash "$VERIFY" "$BIN" "0.2.0" --binaries-only 2>&1)"
check "binaries-only:无 checksums/manifest 通过" "$?" "0"
contains "binaries-only:输出 PASS" "$out" "PASS"
BIN2="$TMP/binonly-missing"
mkdir -p "$BIN2"
for n in "${ASSET_NAMES[@]}"; do
  printf 'fixture-%s\n' "$n" > "$BIN2/$n"
done
rm "$BIN2/AlchemyFurnace-mac-arm64.dmg"
out="$(bash "$VERIFY" "$BIN2" "0.2.0" --binaries-only 2>&1)"
rc=$?
if [[ $rc -ne 0 && "$out" == *"AlchemyFurnace-mac-arm64.dmg"* ]]; then
  PASS=$((PASS + 1))
  printf 'ok   - binaries-only 阶段缺资产仍失败\n'
else
  FAILED=$((FAILED + 1))
  printf 'FAIL - binaries-only 阶段缺资产仍失败(rc=%s)\n      输出: %s\n' "$rc" "$out"
fi

# ─── 9. 参数校验 ───
expect_fail "缺版本参数" "$GOOD" "" "版本"
expect_fail "目录不存在" "$TMP/nope" "0.2.0" "目录"

printf '\n%d passed, %d failed\n' "$PASS" "$FAILED"
if [[ "$FAILED" -gt 0 ]]; then
  exit 1
fi
exit 0
