#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════
# 炼丹炉 · Release 资产完整性门禁
#
# 用法: scripts/verify-release-assets.sh <dist-directory> <version> [--binaries-only]
#
# 契约: 只接受以下 7 个文件,其余一律失败
#   AlchemyFurnace-mac-arm64.dmg / AlchemyFurnace-mac-arm64.zip
#   AlchemyFurnace-mac-x64.dmg    / AlchemyFurnace-mac-x64.zip
#   AlchemyFurnace-Setup.exe
#   checksums.txt  release-manifest.json
#
# 失败条件: 缺失资产、空文件、重复架构(同架构第二份)、SHA256 不匹配、
#           manifest 版本不符 / 资产 hash 不符、非白名单文件、参数缺失。
#
# --binaries-only: 只验证 5 个二进制(聚合 job 生成 checksums/manifest 前使用)。
# <version> 允许带或不带 v 前缀,与 release-manifest.json 的 version 归一化后比较。
# 退出码: 0 = 通过, 1 = 失败(错误信息到 stderr)。
# ═══════════════════════════════════════════════════════════

set -u

ASSETS=(
  AlchemyFurnace-mac-arm64.dmg
  AlchemyFurnace-mac-arm64.zip
  AlchemyFurnace-mac-x64.dmg
  AlchemyFurnace-mac-x64.zip
  AlchemyFurnace-Setup.exe
)
META_FILES=(checksums.txt release-manifest.json)

DIST_DIR="${1:-}"
VERSION="${2:-}"
BINARIES_ONLY=0
if [[ "${3:-}" == "--binaries-only" ]]; then
  BINARIES_ONLY=1
fi

ERRORS=()
fail() { ERRORS+=("$*"); }

contains_element() { # contains_element <needle> <items...>
  local needle="$1" e
  shift
  for e in "$@"; do
    [[ "$e" == "$needle" ]] && return 0
  done
  return 1
}

hash_of() { # hash_of <file> → stdout: 64-hex(小写)
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

# ─── 参数校验 ───
if [[ -z "$DIST_DIR" ]]; then
  echo "缺少 dist 目录参数: $0 <dist-directory> <version> [--binaries-only]" >&2
  exit 1
fi
if [[ -z "$VERSION" ]]; then
  echo "缺少版本参数: $0 <dist-directory> <version> [--binaries-only]" >&2
  exit 1
fi
NORM_VERSION="${VERSION#v}"
if [[ -z "$NORM_VERSION" || ! "$NORM_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]]; then
  echo "版本参数不合法: $VERSION" >&2
  exit 1
fi
if [[ ! -d "$DIST_DIR" ]]; then
  echo "不是目录: $DIST_DIR" >&2
  exit 1
fi

# ─── 目录扫描: 分类每个顶层条目 ───
SEEN=()
while IFS= read -r entry; do
  [[ -z "$entry" ]] && continue
  base="$(basename "$entry")"
  if contains_element "$base" "${ASSETS[@]}" "${META_FILES[@]}"; then
    SEEN+=("$base")
  elif [[ "$base" == AlchemyFurnace-mac-arm64* || "$base" == AlchemyFurnace-mac-x64* ]]; then
    # 前缀命中同架构但非白名单名 → 同架构出现第二份副本
    fail "重复架构资产: $base(每个架构只允许恰好一份 .dmg 与一份 .zip)"
  else
    fail "非白名单文件: $base"
  fi
done < <(find "$DIST_DIR" -mindepth 1 -maxdepth 1)

# ─── 5 个平台二进制: 缺失 + 空文件 ───
for a in "${ASSETS[@]}"; do
  if ! contains_element "$a" "${SEEN[@]}"; then
    fail "缺失资产: $a"
  elif [[ ! -s "$DIST_DIR/$a" ]]; then
    fail "空文件: $a"
  fi
done

# ─── 元数据文件(全量模式才要求) ───
if [[ "$BINARIES_ONLY" -eq 0 ]]; then
  for m in "${META_FILES[@]}"; do
    if ! contains_element "$m" "${SEEN[@]}"; then
      fail "缺失资产: $m"
    elif [[ ! -s "$DIST_DIR/$m" ]]; then
      fail "空文件: $m"
    fi
  done

  # ─── checksums.txt: 必须恰好覆盖 5 个二进制且 SHA256 匹配 ───
  CS_NAMES=()
  CS_HASHES=()
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    line="${line%$'\r'}" # 兼容 CRLF
    if [[ "$line" =~ ^[0-9a-fA-F]{64}[[:space:]]+\*?[[:space:]]*(.+)$ ]]; then
      CS_HASHES+=("${line:0:64}")
      CS_NAMES+=("${BASH_REMATCH[1]}")
    else
      fail "checksums.txt 行格式不合法: $line"
    fi
  done < "$DIST_DIR/checksums.txt"

  for a in "${ASSETS[@]}"; do
    count=0
    cs_hash=""
    for i in "${!CS_NAMES[@]}"; do
      if [[ "${CS_NAMES[$i]}" == "$a" ]]; then
        count=$((count + 1))
        cs_hash="${CS_HASHES[$i]}"
      fi
    done
    if [[ $count -eq 0 ]]; then
      fail "checksums.txt 缺少资产: $a"
    elif [[ $count -gt 1 ]]; then
      fail "checksums.txt 重复资产条目: $a"
    else
      actual="$(hash_of "$DIST_DIR/$a")"
      cs_hash="$(printf '%s' "$cs_hash" | tr 'A-F' 'a-f')"
      if [[ "$cs_hash" != "$actual" ]]; then
        fail "SHA256 不匹配: $a(checksums.txt 期望 $cs_hash, 实际 $actual)"
      fi
    fi
  done
  for i in "${!CS_NAMES[@]}"; do
    if ! contains_element "${CS_NAMES[$i]}" "${ASSETS[@]}"; then
      fail "checksums.txt 包含非资产条目: ${CS_NAMES[$i]}"
    fi
  done

  # ─── release-manifest.json: 版本 + 每资产 sha256 与实际一致 ───
  if ! command -v python3 >/dev/null 2>&1; then
    fail "缺少 python3(无法校验 release-manifest.json)"
  else
    manifest_out="$(python3 - "$DIST_DIR/release-manifest.json" "$NORM_VERSION" <<'PY'
import json, sys

try:
    with open(sys.argv[1], encoding="utf-8") as f:
        m = json.load(f)
except Exception as e:
    print(f"JSON 解析失败: {e}")
    sys.exit(1)

want = sys.argv[2]
if not isinstance(m, dict) or not isinstance(m.get("version"), str):
    print("manifest 缺少 version 字段")
    sys.exit(1)
if m["version"].lstrip("v") != want:
    print(f"manifest 版本不符: 期望 {want}, 实际 {m['version']!r}")
    sys.exit(1)

assets = m.get("assets")
if not isinstance(assets, dict):
    print("manifest 缺少 assets 对象")
    sys.exit(1)
for name, meta in assets.items():
    if not isinstance(meta, dict) or not isinstance(meta.get("sha256"), str):
        print(f"manifest 资产条目不合法: {name}")
        sys.exit(1)
    print(f"{name} {meta['sha256']}")
PY
)"
    rc=$?
    if [[ $rc -ne 0 ]]; then
      fail "release-manifest.json 无效: $manifest_out"
    else
      while IFS= read -r line; do
        name="${line%% *}"
        m_hash="$(printf '%s' "${line#* }" | tr 'A-F' 'a-f')"
        if ! contains_element "$name" "${ASSETS[@]}"; then
          fail "manifest 包含未知资产: $name"
          continue
        fi
        actual="$(hash_of "$DIST_DIR/$name")"
        if [[ "$m_hash" != "$actual" ]]; then
          fail "manifest SHA256 不匹配: $name(期望 $m_hash, 实际 $actual)"
        fi
      done <<< "$manifest_out"
      for a in "${ASSETS[@]}"; do
        if [[ "$manifest_out" != *"$a "* ]]; then
          fail "manifest 缺少资产: $a"
        fi
      done
    fi
  fi
fi

# ─── 汇总 ───
if [[ ${#ERRORS[@]} -gt 0 ]]; then
  for e in "${ERRORS[@]}"; do
    echo "校验失败: $e" >&2
  done
  echo "校验失败: 共 ${#ERRORS[@]} 个错误" >&2
  exit 1
fi

if [[ "$BINARIES_ONLY" -eq 1 ]]; then
  echo "PASS: 5 个平台资产验证通过(version=$NORM_VERSION, 阶段=binaries-only)"
else
  echo "PASS: 5 个平台资产 + checksums.txt + release-manifest.json 验证通过(version=$NORM_VERSION)"
fi
exit 0
