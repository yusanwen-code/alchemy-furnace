#!/usr/bin/env bash
# =============================================================================
# 炼丹炉 · 演示模式一键部署脚本 (007-demo-mode)
# =============================================================================
# 在干净的 Debian/Ubuntu 机器上一键部署完整产品演示页。
# 不需要 PostgreSQL;Go/Python 均启用 APP_MODE=demo,数据走内存 mock。
#
# 用法:
#   sudo bash scripts/deploy-demo.sh
#
# 前置依赖(脚本会检查并提示):
#   git, go >=1.22, python3 >=3.10, python3-venv, pnpm, nginx, curl, systemctl
#
# 部署产物:
#   - /opt/alchemy-furnace/dist/server         Go 网关二进制
#   - /opt/alchemy-furnace/backend/python/.venv Python 运行时
#   - /opt/alchemy-furnace/frontend/out        静态前端产物
#   - /etc/systemd/system/alchemy-go.service
#   - /etc/systemd/system/alchemy-python.service
#   - /etc/nginx/sites-available/alchemy-demo
# =============================================================================

set -euo pipefail

# ---------- 配置 ----------
REPO_DIR="/opt/alchemy-furnace"
GO_PORT="${GO_PORT:-8080}"
PY_PORT="${PY_PORT:-8000}"
HTTP_PORT="${HTTP_PORT:-80}"
GO_BINARY="$REPO_DIR/backend/go/dist/server"
PY_VENV="$REPO_DIR/backend/python/.venv"

# ---------- 颜色 ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

say() { echo -e "${GREEN}[炼丹炉]${NC} $*"; }
warn() { echo -e "${YELLOW}[警告]${NC} $*"; }
fail() { echo -e "${RED}[错误]${NC} $*"; exit 1; }

# ---------- 依赖检查 ----------
say "检查前置依赖..."

need_cmd() {
  local cmd=$1 pkg=${2:-$1}
  if ! command -v "$cmd" &>/dev/null; then
    fail "缺少依赖: $cmd\n安装提示: apt-get update && apt-get install -y $pkg"
  fi
}

need_cmd git git
need_cmd go golang-go
need_cmd python3 python3
need_cmd nginx nginx
need_cmd curl curl
need_cmd systemctl systemd

# python3-venv 不是二进制,检查 venv 模块
if ! python3 -m venv --help &>/dev/null; then
  fail "缺少 python3-venv 模块\n安装提示: apt-get install -y python3-venv"
fi

# pnpm
if ! command -v pnpm &>/dev/null; then
  if command -v npm &>/dev/null; then
    warn "未找到 pnpm,尝试通过 npm 全局安装..."
    npm install -g pnpm
  else
    fail "缺少 pnpm\n安装提示: npm install -g pnpm (需先安装 Node.js >=18.12)"
  fi
fi

# ---------- 准备目录 ----------
say "准备部署目录 $REPO_DIR..."
mkdir -p "$REPO_DIR"

# 如果当前目录就是 repo,复制过去;否则克隆
if [ -f "$PWD/CLAUDE.md" ] && [ -d "$PWD/.git" ]; then
  rsync -a --delete --exclude='node_modules' --exclude='.venv' --exclude='.next' --exclude='out' \
    "$PWD/" "$REPO_DIR/"
else
  if [ ! -d "$REPO_DIR/.git" ]; then
    git clone https://github.com/yusanwen-code/alchemy-furnace.git "$REPO_DIR"
  fi
  cd "$REPO_DIR"
  git fetch origin
  git checkout 007-demo-mode || git checkout master
  git pull
fi

cd "$REPO_DIR"

# ---------- 构建 Go 网关 ----------
say "构建 Go 网关..."
export PATH="$HOME/.nvm/versions/node/v22.12.0/bin:$PATH" 2>/dev/null || true
(
  cd backend/go
  mkdir -p dist
  go build -o "$GO_BINARY" cmd/main/main.go
)
say "Go 网关: $GO_BINARY"

# ---------- 构建 Python 语言引擎 ----------
say "准备 Python 虚拟环境..."
(
  cd backend/python
  if [ ! -d "$PY_VENV" ]; then
    python3 -m venv .venv
  fi
  .venv/bin/pip install -U pip
  if [ -f requirements.txt ]; then
    .venv/bin/pip install -r requirements.txt
  else
    warn "未找到 requirements.txt,跳过 pip 依赖安装"
  fi
)
say "Python 引擎: $PY_VENV"

# ---------- 构建前端 ----------
say "构建前端静态产物..."
(
  cd frontend
  pnpm install --frozen-lockfile
  pnpm build
)
say "前端产物: $REPO_DIR/frontend/out"

# ---------- systemd: Go 服务 ----------
say "安装 systemd 服务..."

cat > /etc/systemd/system/alchemy-go.service <<EOF
[Unit]
Description=Alchemy Furnace Go Gateway (Demo)
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$REPO_DIR/backend/go
Environment=APP_MODE=demo
Environment=PYTHON_ENGINE_BASE_URL=http://127.0.0.1:$PY_PORT
Environment=GIN_MODE=release
ExecStart=$GO_BINARY serve
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

# ---------- systemd: Python 服务 ----------
cat > /etc/systemd/system/alchemy-python.service <<EOF
[Unit]
Description=Alchemy Furnace Python Engine (Demo)
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$REPO_DIR/backend/python
Environment=APP_MODE=demo
ExecStart=$PY_VENV/bin/python -m uvicorn app.main:app --host 127.0.0.1 --port $PY_PORT
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable alchemy-go alchemy-python

# ---------- nginx ----------
say "配置 nginx..."

cat > /etc/nginx/sites-available/alchemy-demo <<EOF
server {
    listen $HTTP_PORT;
    server_name _;

    root $REPO_DIR/frontend/out;
    index index.html;

    # 静态前端:Next.js output:export 产物
    location / {
        try_files \$uri \$uri/ \$uri.html /index.html;
    }

    # API / SSE 代理到 Go 网关
    location /api/ {
        proxy_pass http://127.0.0.1:$GO_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # SSE 流式对话需要禁用缓冲
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
    }

    # 静态资源长期缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
EOF

if [ -d /etc/nginx/sites-enabled ]; then
  rm -f /etc/nginx/sites-enabled/default
  ln -sf /etc/nginx/sites-available/alchemy-demo /etc/nginx/sites-enabled/alchemy-demo
fi

nginx -t || fail "nginx 配置测试失败"
systemctl restart nginx
systemctl enable nginx

# ---------- 启动服务 ----------
say "启动服务..."
systemctl restart alchemy-python
sleep 2
systemctl restart alchemy-go
sleep 3

# ---------- 健康检查 ----------
say "等待健康检查..."
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:$GO_PORT/api/v1/system/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

HEALTH=$(curl -s "http://127.0.0.1:$GO_PORT/api/v1/system/health" 2>/dev/null || echo '{}')
MODE=$(echo "$HEALTH" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('mode','unknown'))" 2>/dev/null || echo 'unknown')
DB=$(echo "$HEALTH" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('db','unknown'))" 2>/dev/null || echo 'unknown')

case "$MODE" in
  demo)
    say "部署成功! 模式: demo | db: $DB"
    ;;
  *)
    fail "健康检查失败或模式异常 (mode=$MODE, db=$DB)\n原始响应: $HEALTH"
    ;;
esac

say "访问地址: http://$(hostname -I | awk '{print $1}' | head -1):$HTTP_PORT"
say "管理命令:"
say "  systemctl status alchemy-go alchemy-python"
say "  journalctl -u alchemy-go -f"
say "  journalctl -u alchemy-python -f"
say ""
say "如需切回真实模式,编辑 /etc/systemd/system/alchemy-{go,python}.service"
say "移除 Environment=APP_MODE=demo,执行 systemctl daemon-reload && systemctl restart alchemy-go alchemy-python"
