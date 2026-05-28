# 炼丹炉 &middot; Alchemy Furnace

<p align="center">
  <img src="https://img.shields.io/badge/React-18-61DAFB?logo=react" />
  <img src="https://img.shields.io/badge/Go-1.21-00ADD8?logo=go" />
  <img src="https://img.shields.io/badge/Python-3.11-3776AB?logo=python" />
  <img src="https://img.shields.io/badge/Qdrant-Vector_DB-FD6096?logo=qdrant" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker" />
  <img src="https://img.shields.io/badge/License-Custom-orange.svg" />
</p>

<p align="center">
  <i>"道生一，一生二，二生三，三生万物——万物皆可炼为金丹"</i>
</p>

<p align="center">
  <a href="#简介">简介</a> &middot;
  <a href="#特性">特性</a> &middot;
  <a href="#架构">架构</a> &middot;
  <a href="#快速开始">快速开始</a> &middot;
  <a href="#概念">概念</a> &middot;
  <a href="#API">API</a> &middot;
  <a href="#文档">文档</a>
</p>

---

## 简介

**炼丹炉**（Alchemy Furnace）是一款以道教炼丹文化为设计灵感的 RAG（检索增强生成）对话系统。在这里，你的文档化作**丹方**，知识库化作**金丹**，AI Agent 化作**道人**——道人服用金丹后，便能运用其中蕴含的知识与你论道对答。

> 古人云："丹成而龙虎伏，道备而鬼神惊。"今人以 AI 为火，以数据为药，炼就知识之金丹，养出智慧之道人。

## 关于作者

本项目「炼丹炉 (Alchemy Furnace)」由 **[yusanwen-code](https://github.com/yusanwen-code)** 开发维护。

- **GitHub**: [https://github.com/yusanwen-code](https://github.com/yusanwen-code)
- **仓库地址**: [https://github.com/yusanwen-code/alchemy-furnace](https://github.com/yusanwen-code/alchemy-furnace)
- **开源方式**: 私人开源，商业闭源。个人使用、学习研究可自由使用；商业使用需单独授权，且必须保留项目名称和开发者信息。

## 特性

### 核心能力

- **丹方萃取** - 支持 Word、Excel、Markdown、PDF、纯文本、音频、视频等十余种文件格式，自动提取内容
- **炼丹入炉** - 智能文本切分（固定长度 / 段落 / 语义三种策略），向量化入库
- **道人养成** - 创建个性化 AI Agent，配置性格、系统提示词、选用大模型
- **服用金丹** - 一个道人可服用多颗金丹，汇聚众家之长
- **论道对答** - 基于 RAG 的流式对话，自动引用金丹中的知识，显示来源

### 技术亮点

- **响应式设计** - 一套 React 代码同时兼容桌面 Web 端和手机 H5 端
- **微服务架构** - Go 负责 API 网关，Python 负责 RAG 引擎，职责分明
- **流式对话** - WebSocket 实时传输，打字机效果呈现 AI 回复
- **异步炼丹** - 文件上传后后台自动处理，无需等待
- **道教美学** - 朱砂红、金箔黄、宣纸米白，CSS 动画炼丹特效，沉浸式体验

## 架构

```
                    紫府（前端 React）
                         │
                    南天门（Nginx）
                    /          \
              左青龙        右白虎
           （Go API）    （Python RAG）
              │              │
         玉简库（PG）    丹房（Qdrant）
```

| 服务 | 技术 | 职责 | 端口 |
|------|------|------|------|
| **紫府** | React 18 + TypeScript + Tailwind CSS | 用户界面 | 80 (Nginx) |
| **左青龙** | Go 1.21 + Gin + GORM | API 网关、业务逻辑 | 8080 |
| **右白虎** | Python 3.11 + FastAPI + LangChain | RAG 引擎、LLM 调用 | 8000 |
| **玉简库** | PostgreSQL 14 | 业务数据持久化 | 5432 |
| **丹房** | Qdrant | 向量存储与检索 | 6333 |
| **南天门** | Nginx | 反向代理、静态文件 | 80 |

## 快速开始

###  prerequisites

- [Docker](https://docs.docker.com/get-docker/) 20.10+
- [Docker Compose](https://docs.docker.com/compose/install/) 2.0+
- 一个 OpenAI 兼容的 API Key（支持 OpenAI、DeepSeek、通义千问等）

### 一键部署

```bash
# 1. 克隆仙府
git clone https://github.com/yusanwen-code/alchemy-furnace.git
cd alchemy-furnace

# 2. 配置天机
cp .env.example .env
# 编辑 .env，填入你的 OPENAI_API_KEY

# 3. 点火烧炉
make deploy

# 4. 开炉取丹
# 打开浏览器访问 http://localhost
```

### 使用 Make 命令

```bash
make help       # 查看所有可用命令
make init       # 初始化环境
make build      # 构建镜像
make up         # 启动服务
make down       # 停止服务
make logs       # 查看日志
make ps         # 查看状态
make clean      # 清理所有数据
make dev-front  # 前端开发模式
make dev-go     # Go 后端开发模式
make dev-python # Python RAG 开发模式
```

### 首次使用指南

1. **创建金丹** - 进入「金丹阁」，点击「炼制新丹」，输入名称和描述
2. **上传丹方** - 进入金丹详情页，拖拽或选择文件上传（支持 docx/xlsx/md/pdf/音频/视频等）
3. **孕育道人** - 进入「道人府」，点击「收徒」，配置道人的名称、性格和选用的大模型
4. **服用金丹** - 进入道人详情页，选择已炼制好的金丹让道人服用
5. **开炉论道** - 进入「炼丹室」，选择道人，开始对话！

## 概念

### 金丹（Elixir Pill）

> "金丹者，混元一气之所结也。"

金丹即**知识库**。你可以创建多个金丹，每个金丹包含一组相关的丹方（文档）。金丹有状态：炼制中（refining）→ 炼成（refined）。

### 丹方（Elixir Recipe）

> "丹方者，采药之规程也。"

丹方即**文档文件**。支持十余种格式：

| 格式 | 扩展名 | 说明 |
|------|--------|------|
| Word | .doc, .docx | 自动提取正文 |
| Excel | .xls, .xlsx | 逐 Sheet 提取 |
| Markdown | .md | 原生支持 |
| 文本 | .txt | 原生支持 |
| PDF | .pdf | 逐页提取 |
| 音频 | .mp3, .wav, .m4a | Whisper 转录 |
| 视频 | .mp4, .avi, .mov | 提取字幕 |

### 道人（Dao Agent）

> "道者，万物之所系也。"

道人即 **AI Agent**。每个道人有独立的性格、系统提示词和选用的大模型。一个道人可以服用多颗金丹，从而拥有多个领域的知识。

### 服用金丹（Bind Pill）

> "服食金丹，脱胎换骨，通达古今。"

将金丹与道人绑定，道人在对话时便能引用该金丹中的知识。一个道人可服用多颗金丹，知识库越丰富，论道越精彩。

### 炼丹（RAG Pipeline）

> "采药 → 配比 → 入炉 → 温养 → 开炉取丹"

对应 RAG 流程：

| 炼丹步骤 | RAG 步骤 | 说明 |
|---------|---------|------|
| 采药 | 文档提取 | 解析文件获取原始文本 |
| 配比 | 文本切分 | 按策略切分为 chunks |
| 入炉 | 向量化 | Embedding 转换为向量 |
| 温养 | 入库 | 存入 Qdrant 向量库 |
| 开炉取丹 | 检索 | 对话时相似度搜索 |

## 截图

<p align="center">
  <i>🖼️ 金丹阁 - 知识库管理</i>
</p>

<p align="center">
  <i>🖼️ 道人府 - AI Agent 管理</i>
</p>

<p align="center">
  <i>🖼️ 炼丹室 - 流式对话</i>
</p>

## API

详见 [docs/api.md](docs/api.md)

### 快速示例

```bash
# 创建金丹
curl -X POST http://localhost:8080/api/v1/pills \
  -H "Content-Type: application/json" \
  -d '{"name": "道德经", "description": "老子五千言"}'

# 上传丹方
curl -X POST http://localhost:8080/api/v1/recipes/upload \
  -F "files[]=@dao_de_jing.md" \
  -F "pill_id=1"

# 创建道人
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "太上老君",
    "personality": "你是太上老君，道家的始祖。言谈充满无上智慧，善用《道德经》的哲理来阐释万物。语气超然物外，又平易近人。",
    "model_name": "gpt-4o"
  }'

# 服用金丹
curl -X POST http://localhost:8080/api/v1/agents/1/pills \
  -H "Content-Type: application/json" \
  -d '{"pill_id": 1}'

# 创建会话并对话（WebSocket）
# 详见 API 文档
```

## 文档

| 文档 | 说明 |
|------|------|
| [docs/architecture.md](docs/architecture.md) | 系统架构详解 |
| [docs/api.md](docs/api.md) | API 接口文档 |
| [docs/frontend.md](docs/frontend.md) | 前端开发文档 |
| [docs/rag-engine.md](docs/rag-engine.md) | RAG 引擎文档 |
| [docs/deployment.md](docs/deployment.md) | 部署指南 |

## 项目结构

```
alchemy-furnace/
├── README.md                    # 项目说明
├── docker-compose.yml           # Docker 编排
├── .env.example                 # 环境变量模板
├── Makefile                     # 常用命令
├── docs/                        # 维护文档
│   ├── architecture.md          # 架构文档
│   ├── api.md                   # API 文档
│   ├── frontend.md              # 前端文档
│   ├── rag-engine.md            # RAG 引擎文档
│   └── deployment.md            # 部署指南
├── backend/
│   ├── go/                      # Go API 网关
│   │   ├── cmd/server/
│   │   ├── internal/
│   │   ├── pkg/
│   │   ├── go.mod
│   │   └── Dockerfile
│   └── python/                  # Python RAG 引擎
│       ├── app/
│       ├── core/
│       ├── models/
│       ├── services/
│       ├── requirements.txt
│       └── Dockerfile
└── frontend/                    # React 前端
    ├── src/
    ├── public/
    ├── package.json
    └── Dockerfile
```

## 技术栈

### 前端
- React 18 + TypeScript
- Tailwind CSS + shadcn/ui
- Vite
- HashRouter（兼容静态部署）
- WebSocket 客户端
- React Context API 状态管理

### 后端
- Go 1.21 + Gin
- GORM + PostgreSQL
- WebSocket
- RESTful API
- Zap 日志

### RAG 引擎
- Python 3.11 + FastAPI
- LangChain + OpenAI
- Qdrant 向量数据库
- Whisper（音频转录）
- FFmpeg（视频处理）

## 开发计划

- [x] 金丹管理（知识库 CRUD）
- [x] 丹方管理（文件上传、解析）
- [x] 道人管理（Agent CRUD）
- [x] 服用金丹（知识库绑定）
- [x] 流式对话（WebSocket）
- [x] 音频转录（Whisper）
- [x] 视频字幕提取
- [x] RAG 检索增强
- [ ] 多轮对话上下文优化
- [ ] 知识图谱支持
- [ ] 多用户权限管理
- [ ] 炼丹过程可视化
- [ ] 金丹分享与 marketplace

## 许可证

本项目采用自定义许可证，详情参见 [LICENSE](LICENSE) 文件。

**简要说明：**

- **个人使用、学习研究**：可自由使用、修改、分发
- **商业使用**：必须保留项目名称「炼丹炉 (Alchemy Furnace)」和开发者信息「yusanwen-code」，且需联系作者获取商业授权

---

<p align="center">
  <i>道法自然 · 炼丹铸智</i>
</p>

<p align="center">
  Made with ☯️ by <a href="https://github.com/yusanwen-code">yusanwen-code</a>
</p>
