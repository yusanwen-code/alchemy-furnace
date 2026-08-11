# 演示模式 (Demo Mode) 部署指南

> 适用于 CI/CD 镜像发布 + 单容器 demo 部署场景。
> demo 模式下后端不依赖 Postgres / 真实模型 API，前端运行期自动展示提示横幅。

---

## 1. 一句话说明

demo 模式 = **同一份二进制镜像 + 一个环境变量**。无外部依赖，开箱即用，进程重启数据重置。

---

## 2. 模式开关

### 环境变量

| 变量 | 取值 | 含义 |
|------|------|------|
| `APP_MODE` | `demo` / `true` / `1` / `yes` | 演示模式，in-memory 存储 |
| `APP_MODE` | `real`（或缺省） | 真实模式，接 Postgres + 真实模型 |

### 后端检测逻辑

`backend/go/internal/configuration/configuration.go`：

```go
// LoadDemoConfig 从环境变量解析演示模式开关
func LoadDemoConfig() {
    v := strings.ToLower(strings.TrimSpace(os.Getenv("APP_MODE")))
    demoConfig.Enabled = v == "true" || v == "1" || v == "yes" || v == "demo"
}
```

### 健康检查端点

`GET /api/v1/system/health` 返回体中 `mode` 字段：

```json
{
  "data": {
    "status": "ok",
    "mode": "demo",        // ← 关键
    "db": "mock",          // demo 时固定为 mock
    "pythonEngine": "ok"
  }
}
```

前端 `frontend/lib/demo-mode.ts` 在浏览器运行期 fetch 此端点，判定后显示/隐藏顶部 DemoBanner。

---

## 3. 部署目标

**默认目标：k3s + GHCR**（`feat/build-pipeline` PR 路线）。

镜像发布（`release.yml`）与运行时模式（`APP_MODE`）解耦，**同一份镜像既能 demo 也能 real**。所有部署侧细节见 `deploy/k8s/`。

### 3.1 镜像列表（GHCR）

| 服务 | 镜像 |
|------|------|
| API 网关 | `ghcr.io/<owner>/alchemy-furnace/go-api:<tag>` |
| 语言引擎 | `ghcr.io/<owner>/alchemy-furnace/python-engine:<tag>` |
| 前端 | `ghcr.io/<owner>/alchemy-furnace/frontend:<tag>` |

### 3.2 demo 部署（k3s，一键 apply）

```bash
# 一次性：创建命名空间 + GHCR 拉取认证 secret
kubectl create namespace alchemy-furnace
kubectl create secret docker-registry ghcr-auth \
  -n alchemy-furnace \
  --docker-server=ghcr.io \
  --docker-username=$GHCR_USER \
  --docker-password=$GHCR_PAT      # PAT 需 read:packages

# 应用全部 manifest
kubectl apply -k deploy/k8s/

# 验证
kubectl -n alchemy-furnace get pods,svc,ingress
```

`deploy/k8s/kustomization.yaml` 默认 `APP_MODE=demo`（通过 ConfigMap 注入），无需额外配置。

### 3.3 real 部署（k3s，加 postgres）

切换模式 = 改 `deploy/k8s/configmap.yaml` 一行 + 加 postgres：

```yaml
data:
  APP_MODE: "real"  # ← 改这个
  DB_HOST: "postgres"
  DB_NAME: "alchemy"
```

postgres 部署建议用 Helm（bitnami/postgresql），不在本仓库范围内。

### 3.4 本地 demo 体验（docker compose 备选）

不想开 k3s 也能体验：仓库提供 `docker-compose.demo.yml`（计划中），拉取 GHCR 镜像后 `docker compose up -d` 即可。

---

## 4. demo 模式数据特性

| 维度 | 行为 |
|------|------|
| 存储位置 | 进程内存（`sync.Map` / `map + sync.RWMutex`） |
| 容器重启 | 数据清空，重新 seed |
| 写入操作 | 接受并立即生效，但仅对当前容器可见 |
| 跨实例 | 各自独立（demo 不应用于多副本部署） |
| 真实数据库 | **不连接**，无 schema 迁移要求 |
| 真实模型 API | **不调用**，所有 LLM 路径走 mock 返回 |

---

## 5. DemoBanner 行为

- 位置：页面顶部，全宽
- 触发：`health.mode === "demo"` 时显示
- 样式：amber 警告色（明亮 / 暗黑模式自适应）
- 交互：右上角 × 可关闭（仅当前会话）
- 文案：通过 i18n `demoBanner.text` 翻译键

代码：`frontend/components/layout/demo-banner.tsx`

---

## 6. CI/CD 中的默认模式

release 镜像的 `Dockerfile` **默认 ENV `APP_MODE=demo`**（计划中，见后文）：

```dockerfile
# backend/go/Dockerfile
ENV APP_MODE=demo
```

这样做的好处：
- 拉取镜像 `docker run ... go-api:v0.1.0` 即可零配置启动
- real 部署在 compose 中用 `environment` 覆盖即可
- GHCR 公开镜像可被任何访客直接 `docker run` 体验

---

## 7. 故障排查

| 现象 | 原因 | 解决 |
|------|------|------|
| `mode: real` 但希望 demo | 环境变量未透传 | 检查 `docker inspect` 的 Env 数组 |
| 横幅不显示 | 前端构建期假设了 `NEXT_PUBLIC_DEMO=true` | 删除构建期 env，运行时探测 |
| demo 模式仍在连 DB | 代码里硬编码了 `dao.GetDB()` 而非 `dao.Get()` | DAO 层需有 demo 替身 |
| 容器重启后数据没了 | **预期行为**，demo 故意设计为无持久化 | — |

---

## 8. 升级路径

demo → real 不需要重新构建镜像：

```bash
# 同一份 go-api:v0.1.0 镜像
docker run -e APP_MODE=real \
           -e DB_HOST=postgres \
           -e DB_PASSWORD=secret \
           ghcr.io/.../go-api:v0.1.0
```

应用代码**零改动**，DAO 层在 init 时根据 `IsDemo()` 选择 `MemoryUser` / `PostgresUser`。

---

## 9. 待办（尚未完成）

- [ ] `backend/go/internal/dao/memory/` 已写但**未接入 wire_gen**，需要：
  1. `dao/memory/daos.go` 定义 `UserDAO` / `ProjectDAO` 等接口实现
  2. `cmd/main/main.go` 启动时根据 `IsDemo()` 选择注入
- [ ] 3 个 Dockerfile 添加 `ENV APP_MODE=demo` 默认值
- [ ] DemoBanner 文案 i18n 补全（zh-CN / en）
- [ ] real 模式启动失败 → 自动回退 demo（可选）
