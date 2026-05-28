# ═══════════════════════════════════════════
# Go API 网关 - 炼丹炉门户
# ═══════════════════════════════════════════

FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git

WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# ─── 生产镜像 ───
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/server .

# 创建上传目录
RUN mkdir -p /app/uploads

EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=15s --timeout=5s --retries=5 \
    CMD curl -f http://localhost:8080/api/v1/system/health || exit 1

CMD ["./server"]
