# 炼丹炉 · 演示模式部署文档 (007-demo-mode)

本文档描述如何在干净的 Linux 服务器上一键部署**演示模式** (Demo Mode)。
演示模式下:

- Go 网关与 Python 语言引擎均启用 `APP_MODE=demo`
- 所有数据走内存 mock,**不依赖 PostgreSQL**
- 启动即内置 9 种金丹 / 9 位道人 / 9 个供应商 / 9 个模型 / 9 条会话与消息
- 前端顶部显示演示模式横幅,可手动收起

---

## 前置条件

| 组件 | 最低版本 | 安装命令 (Debian/Ubuntu) |
|------|---------|--------------------------|
| OS | Debian 12 / Ubuntu 22.04 | - |
| Git | 2.x | `apt-get install git` |
| Go | 1.22+ | `apt-get install golang-go` |
| Python | 3.10+ | `apt-get install python3 python3-venv` |
| Node.js | 18.12+ | 见下 |
| pnpm | 8+ | `npm install -g pnpm` |
| Nginx | 1.18+ | `apt-get install nginx` |
| curl / systemd | 任意 | 通常已预装 |

**Node.js 安装示例**:

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt-get install -y nodejs
npm install -g pnpm
```

---

## 快速部署

```bash
git clone https://github.com/yusanwen-code/alchemy-furnace.git
cd alchemy-furnace
git checkout 007-demo-mode
sudo bash scripts/deploy-demo.sh
```

脚本会完成:

1. 依赖检查 (git/go/python3/pnpm/nginx/systemd)
2. 将仓库同步到 `/opt/alchemy-furnace`
3. 构建 Go 网关二进制到 `/opt/alchemy-furnace/backend/go/dist/server`
4. 创建 Python 虚拟环境并安装依赖
5. 安装前端依赖并构建静态产物到 `frontend/out`
6. 写入 systemd 服务 `alchemy-go` 与 `alchemy-python`
7. 写入 nginx 配置 `alchemy-demo` 并启动
8. 健康检查确认 `mode=demo` 与 `db=mock`

---

## systemd 服务说明

服务文件由脚本自动生成:

- `/etc/systemd/system/alchemy-go.service`
- `/etc/systemd/system/alchemy-python.service`

常用命令:

```bash
# 查看状态
systemctl status alchemy-go alchemy-python

# 查看日志
journalctl -u alchemy-go -f
journalctl -u alchemy-python -f

# 重启
systemctl restart alchemy-go alchemy-python

# 停止
systemctl stop alchemy-go alchemy-python
```

### 环境变量

两个服务均默认注入:

```env
APP_MODE=demo
```

Go 服务额外注入:

```env
PYTHON_ENGINE_BASE_URL=http://127.0.0.1:8000
GIN_MODE=release
```

---

## nginx 配置

服务由 `/etc/nginx/sites-available/alchemy-demo` 管理,主要逻辑:

- `/` 直接服务 `frontend/out` 中的静态产物 (Next.js `output: 'export'` 生成)
- `/api/` 反向代理到本机 Go 网关 `127.0.0.1:8080`
- SSE 流式对话路径禁用缓冲 (`proxy_buffering off`)

验证配置:

```bash
nginx -t
systemctl reload nginx
```

---

## 验证

脚本末尾会调用 `/api/v1/system/health` 并检查 `mode` 字段。部署成功后应看到:

```json
{
  "data": {
    "status": "ok",
    "db": "mock",
    "python_engine": "ok",
    "mode": "demo"
  }
}
```

手动验证:

```bash
# 1. 健康检查
curl -s http://127.0.0.1/api/v1/system/health | python3 -m json.tool

# 2. 种子数据数量 (应为 9)
curl -s 'http://127.0.0.1/api/v1/pills?page=1&page_size=20' | \
  python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['total'])"

curl -s 'http://127.0.0.1/api/v1/agents?page=1&page_size=20' | \
  python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['total'])"

# 3. 演示横幅文本 (从构建出的 HTML 中检查)
grep -o "演示模式" /opt/alchemy-furnace/frontend/out/index.html | head -1
```

---

## 回退到真实模式

演示模式与真实模式不能同时运行 (共享 8080/8000 端口)。如需切换:

1. 编辑服务文件,移除 `Environment=APP_MODE=demo`:

```bash
sed -i '/APP_MODE=demo/d' /etc/systemd/system/alchemy-go.service
sed -i '/APP_MODE=demo/d' /etc/systemd/system/alchemy-python.service
```

2. 准备 PostgreSQL 数据库,配置 `config/main.toml` 与 `.env`。

3. 重载并重启服务:

```bash
systemctl daemon-reload
systemctl restart alchemy-go alchemy-python
```

4. 健康检查应返回 `mode: "real"` 与 `db: "connected"`。

---

## 升级演示站

1. 拉取最新代码:

```bash
cd /opt/alchemy-furnace
git pull origin 007-demo-mode
```

2. 重新执行部署脚本 (幂等):

```bash
sudo bash scripts/deploy-demo.sh
```

脚本会自动重建 Go 二进制、重新安装 Python 依赖、重新构建前端并重启服务。

---

## 故障排查

### 健康检查失败

```bash
journalctl -u alchemy-go -n 50 --no-pager
journalctl -u alchemy-python -n 50 --no-pager
nginx -t
systemctl status nginx
```

### Python 模块未找到

确保 systemd 服务中的 `WorkingDirectory` 指向 `/opt/alchemy-furnace/backend/python`,
且 `ExecStart` 使用 `.venv/bin/python -m uvicorn app.main:app`。

### 前端不显示横幅

浏览器访问 `/api/v1/system/health`,确认响应包含 `"mode": "demo"`;
确认 nginx 将 `/api/` 正确代理到 Go 网关。

---

## 端口说明

| 端口 | 用途 | 监听范围 |
|------|------|----------|
| 80 | Nginx HTTP | 0.0.0.0 |
| 8080 | Go 网关 | 127.0.0.1 (内部) |
| 8000 | Python 语言引擎 | 127.0.0.1 (内部) |

外部用户只需访问 `http://服务器IP/`。
