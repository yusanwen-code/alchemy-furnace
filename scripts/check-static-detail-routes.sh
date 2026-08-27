#!/usr/bin/env bash
set -euo pipefail

# 生产静态产物契约检查：实体详情必须导出为查询参数路由（agents/detail.html、pills/detail.html），
# 且不得残留 `_` 占位动态路由（agents/_.html、pills/_.html）。
# 与 backend/go/internal/webui/webui.go 的映射逻辑互为守护：任一侧回归都会让这里挂掉。
# 用法：先 `cd frontend && pnpm build`（或 make test-frontend），再运行本脚本。

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/frontend/out"

if [ ! -d "$OUT" ]; then
  echo "❌ 未找到前端产物目录: frontend/out（请先执行 pnpm build）" >&2
  exit 1
fi

fail=0

for f in agents/detail.html pills/detail.html; do
  if [ ! -f "$OUT/$f" ]; then
    echo "❌ 缺少生产产物: frontend/out/$f" >&2
    fail=1
  fi
done

for f in agents/_.html pills/_.html; do
  if [ -e "$OUT/$f" ]; then
    echo "❌ 不应存在占位产物(动态路由已废弃): frontend/out/$f" >&2
    fail=1
  fi
done

# Next 16 会把 agentDetailHref/pillDetailHref 编译为 `/${kind}/detail?id=${...}` 模板
# (运行时插值,产物里无字面量 /agents/detail);rg 未必在 PATH 且默认遵守 .gitignore,
# 故用 grep -R 直接搜产物目录(BRE 里 ? 为字面量)。
if ! grep -R -q 'detail?id=' "$OUT" 2>/dev/null; then
  echo "❌ 产物中未找到详情查询路由引用 (detail?id=)" >&2
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  echo "静态实体详情路由契约检查失败" >&2
  exit 1
fi

echo "✅ 静态实体详情路由契约通过 (frontend/out)"
