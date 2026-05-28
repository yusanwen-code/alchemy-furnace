# ═══════════════════════════════════════════
# React 前端 - 炼丹炉界面
# ═══════════════════════════════════════════

FROM node:20-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY package*.json ./
RUN npm install

# 复制源代码
COPY . .

# 构建生产版本
RUN npm run build

# 将构建结果复制到共享卷（由 docker-compose 挂载到 nginx）
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/dist ./dist
VOLUME /app/dist
CMD ["sh", "-c", "cp -r /app/dist/* /out/ 2>/dev/null || true"]
