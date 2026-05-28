# RAG 引擎文档

## 概述

Python RAG 引擎是「炼丹炉」的核心，负责文档解析、文本切分、向量化、存储和检索。以道教炼丹术语命名各模块。

## 模块架构

```
app/
├── api/
│   ├── documents.py    # 丹方解析 API
│   ├── vectors.py      # 金丹入库 API
│   ├── chat.py         # 论道 API
│   └── media.py        # 音视频处理 API
├── core/
│   ├── document/
│   │   └── parser.py   # 丹方解析器
│   ├── embedding/
│   │   └── embedder.py # 炼丹术（向量化）
│   ├── vectorstore/
│   │   └── qdrant_store.py  # 丹房（向量库）
│   ├── splitter/
│   │   └── text_splitter.py # 切丹术（文本切分）
│   └── retrieval/
│       └── retriever.py     # 寻丹术（检索器）
└── services/
    ├── document_service.py  # 丹方处理服务
    ├── vector_service.py    # 金丹入库服务
    ├── chat_service.py      # 论道对话服务
    └── media_service.py     # 音视频处理服务
```

## 文档解析器 (parser.py)

### 支持的文件类型

| 文件类型 | 扩展名 | 解析器类 | 依赖 |
|---------|--------|---------|------|
| Word | .docx | `DocxParser` | python-docx |
| Excel | .xlsx | `ExcelParser` | openpyxl |
| Markdown | .md | `MarkdownParser` | 原生 |
| 文本 | .txt | `TextParser` | 原生 |
| PDF | .pdf | `PDFParser` | pdfplumber |

### 使用方式

```python
from core.document.parser import ParserFactory

# 自动根据文件类型选择解析器
parser = ParserFactory.get_parser("docx")
text = parser.parse("/path/to/file.docx")

# 返回解析后的纯文本
print(text)  # "文档内容..."
```

### 扩展新解析器

```python
from core.document.parser import BaseParser

class NewParser(BaseParser):
    def parse(self, file_path: str) -> str:
        # 实现解析逻辑
        return "解析后的文本"
    
    def supported_extensions(self) -> list:
        return [".new"]

# 注册到工厂
ParserFactory.register_parser("new", NewParser)
```

## 文本切分器 (text_splitter.py)

### 切分策略

| 策略 | 说明 | 适用场景 |
|------|------|---------|
| `fixed` | 固定长度（500 字符，重叠 50） | 通用场景 |
| `paragraph` | 按段落（空行分隔） | 段落清晰的文档 |
| `semantic` | 按语义边界（标题层级） | 结构化文档 |

### 使用方式

```python
from core.splitter.text_splitter import TextSplitter

splitter = TextSplitter(chunk_size=500, chunk_overlap=50)

# 固定长度切分
chunks = splitter.split(text, strategy="fixed")

# 按段落切分
chunks = splitter.split(text, strategy="paragraph")

# 返回格式
# [{"content": "文本块", "metadata": {"index": 0, "strategy": "fixed"}}]
```

## 向量化 (embedder.py)

### 配置

```python
# 通过环境变量配置
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://api.openai.com/v1
EMBEDDING_MODEL=text-embedding-3-small  # 1536 维
```

### 使用方式

```python
from core.embedding.embedder import Embedder

embedder = Embedder()

# 单文本向量化
vector = embedder.embed(["文本"])  # [[0.1, 0.2, ...]]

# 批量向量化
vectors = embedder.embed(["文本1", "文本2", "文本3"])
```

## 向量存储 (qdrant_store.py)

### 集合结构

- **Collection**: `elixir_pills`
- **Vector Size**: 1536 (OpenAI) / 可配置
- **Distance**: Cosine
- **Payload**:
  - `pill_id`: 金丹 ID (integer)
  - `recipe_id`: 丹方 ID (integer)
  - `content`: 文本内容 (string)
  - `chunk_index`: 块索引 (integer)
  - `metadata`: 额外元数据 (object)

### 使用方式

```python
from core.vectorstore.qdrant_store import QdrantStore

store = QdrantStore()

# 初始化集合（启动时自动执行）
store.init_collection()

# 向量化入库（炼丹）
store.upsert(
    pill_id=1,
    recipe_id=1,
    chunks=[
        {"content": "文本块1", "metadata": {"index": 0}},
        {"content": "文本块2", "metadata": {"index": 1}}
    ]
)

# 搜索（寻丹）
results = store.search(
    pill_ids=[1, 2],
    query_vector=[0.1, 0.2, ...],
    top_k=5
)

# 删除金丹的所有向量
store.delete_by_pill(pill_id=1)
```

## 检索器 (retriever.py)

### 检索流程

```
用户查询 → Embedding → Qdrant 搜索 (filter by pill_ids) → 返回 Top-K 结果
```

### 使用方式

```python
from core.retrieval.retriever import Retriever

retriever = Retriever()

# 检索
results = retriever.retrieve(
    pill_ids=[1, 2],      # 在指定金丹中搜索
    query="何为道？",      # 查询内容
    top_k=5               # 返回数量
)

# 返回结果
# [{"content": "相关文本", "score": 0.95, "metadata": {...}}]
```

## 对话服务 (chat_service.py)

### RAG Prompt 模板

```
【系统指令】
你是{agent_name}，{personality}

【已服用金丹】
你已将以下金丹炼化于丹田之中，论道时可引用其中内容：

{context}

【论道规则】
1. 回答时请基于已服用金丹中的知识
2. 如引用金丹内容，请自然融入回答
3. 若金丹中无相关内容，请基于自身知识回答
4. 保持{agent_name}的言谈风格

【引用格式】
回答结束后，可附加引用来源。
```

### 使用方式

```python
from services.chat_service import ChatService

chat_service = ChatService()

# 非流式对话
response = await chat_service.chat(
    messages=[{"role": "user", "content": "何为道？"}],
    pill_ids=[1, 2],
    model="gpt-4o"
)

# 流式对话 (SSE)
async for chunk in chat_service.chat_stream(
    messages=[{"role": "user", "content": "何为道？"}],
    pill_ids=[1, 2],
    model="gpt-4o"
):
    print(chunk)  # "道", "可", "道", ...
```

## 音视频处理 (media_service.py)

### 音频转录

```python
from services.media_service import MediaService

service = MediaService()

# 音频转文字
text = await service.transcribe("/path/to/audio.mp3")
```

### 视频提取字幕

```python
# 提取视频音频 → 转文字
text = await service.extract_subtitles("/path/to/video.mp4")
```

### 流程

```
视频文件 → FFmpeg 提取音频 → Whisper 转文字 → 返回文本
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PYTHON_PORT` | 8000 | 服务端口 |
| `QDRANT_HOST` | qdrant | Qdrant 地址 |
| `QDRANT_PORT` | 6333 | Qdrant 端口 |
| `QDRANT_COLLECTION` | elixir_pills | 集合名称 |
| `OPENAI_API_KEY` | - | OpenAI API Key |
| `OPENAI_BASE_URL` | https://api.openai.com/v1 | API 基础地址 |
| `EMBEDDING_MODEL` | text-embedding-3-small | Embedding 模型 |
| `CHUNK_SIZE` | 500 | 切分块大小 |
| `CHUNK_OVERLAP` | 50 | 切分重叠大小 |
| `TOP_K` | 5 | 检索返回数量 |
