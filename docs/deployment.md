# 部署指南

## 环境要求

- Docker 20.10+
- Docker Compose 2.0+
- 至少 4GB 可用内存
- 至少 10GB 磁盘空间

## 快速部署

### 1. 克隆项目

```bash
git clone https://github.com/yourusername/alchemy-furnace.git
cd alchemy-furnace
```

### 2. 配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件
vim .env
```

至少配置以下项：

```env
# LLM API Key（必填）
OPENAI_API_KEY=sk-your-api-key-here

# 如果使用非 OpenAI 的兼容接口，修改 base URL
# OPENAI_BASE_URL=https://api.deepseek.com/v1
```

### 3. 一键启动

```bash
make deploy
```

或手动执行：

```bash
docker-compose build
docker-compose up -d
```

### 4. 访问服务

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端界面 | http://localhost | 主要用户界面 |
| Go API | http://localhost:8080 | API 网关 |
| Python RAG | http://localhost:8000 | RAG 引擎 + Swagger 文档 |
| Qdrant | http://localhost:6333 | 向量数据库管理界面 |

## 开发环境部署

### 方式一：Docker Compose（推荐）

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f go-api
docker-compose logs -f python-rag
```

### 方式二：本地开发（热重载）

需要本地安装：Node.js 20+、Go 1.21+、Python 3.11+、PostgreSQL 14、Qdrant

**终端 1 - 前端**：
```bash
cd frontend
npm install
npm run dev
```

**终端 2 - Go 后端**：
```bash
cd backend/go
go mod tidy
go run cmd/server/main.go
```

**终端 3 - Python RAG**：
```bash
cd backend/python
pip install -r requirements.txt
uvicorn app.main:app --reload
```

**终端 4 - 基础设施**：
```bash
# 启动 PostgreSQL 和 Qdrant
docker-compose up -d postgres qdrant
```

## 生产环境部署

### 1. 使用 Nginx 反向代理

已配置在 `infra/nginx/nginx.conf`，Docker Compose 会自动启动 Nginx 容器。

如需使用外部 Nginx，修改配置：

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    # 前端静态文件
    location / {
        root /path/to/alchemy-furnace/frontend/dist;
        try_files $uri $uri/ /index.html;
    }
    
    # API 代理
    location /api/v1/ {
        proxy_pass http://localhost:8080/api/v1/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### 2. HTTPS 配置

使用 Let's Encrypt：

```bash
# 安装 certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d your-domain.com
```

### 3. 系统服务

创建 systemd 服务文件 `/etc/systemd/system/alchemy-furnace.service`：

```ini
[Unit]
Description=Alchemy Furnace RAG System
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/path/to/alchemy-furnace
ExecStart=/usr/bin/docker-compose up -d
ExecStop=/usr/bin/docker-compose down

[Install]
WantedBy=multi-user.target
```

启用服务：

```bash
sudo systemctl enable alchemy-furnace
sudo systemctl start alchemy-furnace
```

## 更新部署

```bash
# 拉取最新代码
git pull origin main

# 重建并重启
docker-compose down
docker-compose pull
docker-compose build --no-cache
docker-compose up -d

# 查看状态
docker-compose ps
```

## 数据备份

### 备份 PostgreSQL

```bash
# 备份
docker exec alchemy-postgres pg_dump -U alchemy alchemy_db > backup.sql

# 恢复
cat backup.sql | docker exec -i alchemy-postgres psql -U alchemy alchemy_db
```

### 备份 Qdrant

```bash
# 备份快照
curl -X POST 'http://localhost:6333/collections/elixir_pills/snapshots'

# 下载快照
curl -O 'http://localhost:6333/collections/elixir_pills/snapshots/<snapshot_name>'
```

### 备份上传文件

```bash
# 备份 uploads 目录
tar -czf uploads-backup.tar.gz /var/lib/docker/volumes/alchemy-furnace_uploads_data/_data
```

## 故障排查

### 服务无法启动

```bash
# 查看日志
docker-compose logs

# 检查端口占用
lsof -i :8080
lsof -i :8000
lsof -i :5432
lsof -i :6333
```

### 数据库连接失败

```bash
# 检查 PostgreSQL 状态
docker-compose ps postgres

# 进入 PostgreSQL 容器
docker exec -it alchemy-postgres psql -U alchemy -d alchemy_db

# 检查连接配置
cat .env | grep DB_
```

### 向量搜索无结果

```bash
# 检查 Qdrant 状态
curl http://localhost:6333/healthz

# 查看集合信息
curl http://localhost:6333/collections/elixir_pills

# 检查向量数量
curl -X POST http://localhost:6333/collections/elixir_pills/points/count
```

### LLM 调用失败

```bash
# 检查 Python RAG 日志
docker-compose logs python-rag

# 测试 API 连通性
curl http://localhost:8000/health

# 检查 API Key 配置
cat .env | grep OPENAI
```

## 性能优化

### 1. 数据库优化

```sql
-- 添加索引（已自动迁移创建）
CREATE INDEX idx_recipes_pill_id ON elixir_recipes(pill_id);
CREATE INDEX idx_agent_pills_agent_id ON agent_pills(agent_id);
CREATE INDEX idx_chat_messages_session_id ON chat_messages(session_id);
```

### 2. 向量检索优化

- 增加 Qdrant 内存限制
- 调整 `TOP_K` 参数
- 使用 HNSW 索引（Qdrant 默认）

### 3. 文件上传优化

- 使用对象存储（S3/OSS）替代本地存储
- 配置 CDN 加速静态资源
