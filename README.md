# 炼丹炉 · Alchemy Furnace

> 面向桌面端的语言模式工程工作台：把可复用的表达方式炼成金丹，让道人在对谈与围炉论道中稳定呈现自己的风格。

[English README](README_EN.md) · [项目仓库](https://github.com/yusanwen-code/alchemy-furnace)

## 项目定位

炼丹炉是一个 **Wails 桌面应用**。它在本机运行 Go 网关、Python 语言引擎和 Next.js 静态界面，数据默认保存在用户配置目录下的 SQLite 文件中；模型调用使用你在设置中配置的 OpenAI 兼容接口。

桌面安装包是正式产品形态。Web、`serve` 命令和 Docker Compose 仍保留用于开发、诊断和 CI，不承诺作为独立 Web 产品或移动 H5 产品交付。

## 当前功能

- **金丹阁**：管理语言模式技能包；页面只保留「全部金丹」与「融合金丹」两个入口。支持手工创建、编辑、试丹，以及通过「女娲智能蒸馏」从材料生成候选草稿后确认入库。
- **道人府**：管理 AI 道人列表、道号、简介、头像、基础性格、模型和服丹关系；金丹以权重与顺序影响合成结果。
- **论道**：在「对谈」和「围炉论道」两个 Tab 中聊天。单聊会话按道人归档；围炉论道支持设置主题名称，便于快速定位历史讨论。
- **论道旧录**：首页的最近会话和完整会话目录使用真实的道人名称展示，不向用户展示道人 UUID。
- **设置**：配置个人简介与头像、模型提供商、默认模型及桌面应用选项。
- **金丹化性**：Go 负责业务编排与缓存，Python 引擎负责结构化合成和 OpenAI 兼容模型调用；相同来源指纹会复用有效的语言模式缓存。

## 核心概念

| 概念 | 说明 |
| --- | --- |
| 金丹（Elixir Pill） | 描述一种语言模式、技能或表达倾向的结构化技能包（`skill_schema`）。 |
| 道人（Dao Agent） | 具备基础性格、道号、模型和服丹记录的 AI 对话角色。 |
| 服丹（Binding） | 将金丹绑定到道人，并设置 `weight`（0–10）与 `sort_order`。 |
| 合成（Synthesis） | 将基础性格与已服金丹合并，生成可供对话使用的系统提示词与规则。 |
| 融合金丹（Fusion） | 把多枚金丹提炼为新的金丹，保留来源和版本信息。 |

## 架构

```text
Wails 桌面壳（Next.js 静态 UI）
              │ HTTP / WebSocket
              ▼
       Go API 网关与业务层
          │              │
          │              └── SQLite（桌面默认单文件）
          ▼
     Python 语言引擎
  合成 / 蒸馏 / LLM 对话
```

桌面启动时会准备本地数据目录、迁移数据库并拉起随包提供的 Python runtime。自部署或开发模式可切换 PostgreSQL/MySQL，并通过 Docker Compose 编排服务。

## 安装与使用

从 [GitHub Releases](https://github.com/yusanwen-code/alchemy-furnace/releases) 下载对应平台安装包：

- macOS Apple Silicon（`darwin-arm64`）
- macOS Intel（`darwin-amd64`）
- Windows x64（`windows-amd64`）

启动后依次完成：

1. 在「设置」中配置模型提供商、API Key、Base URL 和默认模型；
2. 在「金丹阁」创建或蒸馏金丹；
3. 在「道人府」创建道人并服用金丹；
4. 进入「论道」开始对谈或围炉论道。

API Key 会加密保存在本地数据库中。请勿把 `.env`、导出的配置或日志提交到仓库。

## 本地开发

### 环境要求

- Go（见 `backend/go/go.mod`）
- Python 3.11+ 与 `pip`
- Node.js 20+、`pnpm`
- Docker Desktop（仅在使用 PostgreSQL 或完整 Compose 开发栈时需要）

### 快速开始

```bash
make init                 # 创建 .env（仅开发/自部署配置）
make dev                  # 启动前端、Go、Python，并检查数据库
```

也可以分别启动服务：

```bash
make dev-front
make dev-go
make dev-python
```

### 测试、格式化与桌面打包

```bash
make test
make format
make desktop-package PLATFORM=darwin-arm64 VERSION=v0.1.0
# PLATFORM 可选：darwin-arm64、darwin-amd64、windows-amd64
```

`make build/up/down` 面向 Docker Compose 开发环境；它们不是桌面用户的安装方式。

## 目录结构

```text
backend/go/       Go API 网关、Wails 入口、数据访问与业务服务
backend/python/   FastAPI 语言引擎、蒸馏与 LLM 调用
frontend/         Next.js + React + Tailwind 桌面界面
scripts/          开发、runtime 构建、桌面打包与产物校验
docs/             架构、部署、运维与设计文档
specs/            功能规格与数据契约
```

## 进一步阅读

- [系统架构](docs/architecture.md)
- [桌面发布流程](docs/deployment/desktop-release.md)
- [前端说明](docs/frontend.md)
- [英文 README](README_EN.md)

## 许可证

本项目使用仓库中的 [自定义许可证](LICENSE)。默认允许个人、学习和非商业使用；商业使用需事先获得项目作者书面授权。

© yusanwen-code · Alchemy Furnace
