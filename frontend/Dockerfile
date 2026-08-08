# ═══════════════════════════════════════════
# Next.js 前端 - 炼丹炉界面
# 构建产物 out/（next build 静态导出）复制到共享卷，由 nginx 服务
# ═══════════════════════════════════════════

FROM node:20-alpine AS builder

WORKDIR /app

# 启用 pnpm
RUN corepack enable

# 复制依赖文件
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# 复制源代码
COPY . .

# 构建生产版本（静态导出 out/）
# 注意：next/font 需要访问 Google Fonts，受限网络下需配置代理：
#   docker build --build-arg HTTPS_PROXY=http://host:port .
ARG NEXT_PUBLIC_API_BASE_URL
ARG NEXT_PUBLIC_WS_BASE_URL
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL \
    NEXT_PUBLIC_WS_BASE_URL=$NEXT_PUBLIC_WS_BASE_URL
RUN pnpm build

# 将构建结果复制到共享卷（由 docker-compose 挂载到 nginx）
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/out ./dist
VOLUME /app/dist
CMD ["sh", "-c", "cp -r /app/dist/* /out/ 2>/dev/null || true"]
