# API 接口文档

## 基础信息

- **Base URL**: `/api/v1`
- **请求格式**: JSON
- **响应格式**: JSON
- **WebSocket**: `/api/v1/chat/ws/:session_id`

### 统一响应格式

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

错误响应:

```json
{
  "code": 1001,
  "message": "错误描述",
  "data": null
}
```

## 金丹管理 (Pills)

### 获取金丹列表

```
GET /api/v1/pills?page=1&page_size=10
```

响应:
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "total": 10,
    "items": [
      {
        "id": 1,
        "name": "太上老君丹",
        "description": "包含道德经核心要义",
        "status": "refined",
        "vector_count": 156,
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

### 创建金丹

```
POST /api/v1/pills
```

请求体:
```json
{
  "name": "新金丹",
  "description": "金丹描述"
}
```

### 获取金丹详情

```
GET /api/v1/pills/:id
```

### 更新金丹

```
PUT /api/v1/pills/:id
```

请求体:
```json
{
  "name": "更新名称",
  "description": "更新描述"
}
```

### 删除金丹

```
DELETE /api/v1/pills/:id
```

> 级联删除该金丹下的所有丹方和向量数据

## 丹方管理 (Recipes)

### 上传丹方

```
POST /api/v1/recipes/upload
```

请求格式: `multipart/form-data`

| 字段 | 类型 | 说明 |
|------|------|------|
| files[] | File | 文件列表，支持多文件 |
| pill_id | integer | 所属金丹 ID |

响应:
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "uploaded": [
      {
        "id": 1,
        "filename": "document.docx",
        "file_type": "docx",
        "file_size": 10240,
        "extract_status": "pending"
      }
    ],
    "failed": []
  }
}
```

### 获取金丹下的丹方列表

```
GET /api/v1/recipes/pill/:pill_id?page=1&page_size=20
```

### 删除丹方

```
DELETE /api/v1/recipes/:id
```

### 重新提取

```
POST /api/v1/recipes/:id/re-extract
```

## 道人管理 (Agents)

### 获取道人列表

```
GET /api/v1/agents?page=1&page_size=10
```

### 创建道人

```
POST /api/v1/agents
```

请求体:
```json
{
  "name": "太上老君",
  "avatar": "https://example.com/avatar.png",
  "personality": "你是太上老君，道教的始祖，言谈充满智慧...",
  "model_name": "gpt-4o"
}
```

### 获取道人详情

```
GET /api/v1/agents/:id
```

### 更新道人

```
PUT /api/v1/agents/:id
```

### 删除道人

```
DELETE /api/v1/agents/:id
```

### 服用金丹（绑定知识库）

```
POST /api/v1/agents/:id/pills
```

请求体:
```json
{
  "pill_id": 1
}
```

### 解除金丹绑定

```
DELETE /api/v1/agents/:id/pills/:pill_id
```

### 获取已服用金丹列表

```
GET /api/v1/agents/:id/pills
```

## 对话管理 (Chat)

### 创建会话

```
POST /api/v1/chat/sessions
```

请求体:
```json
{
  "agent_id": 1,
  "title": "论道"
}
```

### 获取会话列表

```
GET /api/v1/chat/sessions?page=1&page_size=20
```

### 获取会话消息

```
GET /api/v1/chat/sessions/:id/messages?page=1&page_size=50
```

响应:
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "total": 10,
    "items": [
      {
        "id": 1,
        "session_id": 1,
        "role": "user",
        "content": "何为道？",
        "sources": null,
        "created_at": "2025-01-01T00:00:00Z"
      },
      {
        "id": 2,
        "session_id": 1,
        "role": "assistant",
        "content": "道可道，非常道...",
        "sources": [
          {
            "content": "道可道，非常道；名可名，非常名。",
            "metadata": {"filename": "道德经.md", "chunk_index": 0}
          }
        ],
        "created_at": "2025-01-01T00:00:01Z"
      }
    ]
  }
}
```

### 流式对话 (WebSocket)

```
WS /api/v1/chat/ws/:session_id
```

连接后发送 JSON 消息:
```json
{
  "content": "何为道？"
}
```

服务端流式返回:
```json
{"type": "chunk", "content": "道"}
{"type": "chunk", "content": "可"}
{"type": "chunk", "content": "道"}
...
{"type": "done", "sources": [...]}
```

错误返回:
```json
{"type": "error", "message": "错误描述"}
```

## 系统接口 (System)

### 健康检查

```
GET /api/v1/system/health
```

响应:
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "services": {
      "database": "connected",
      "qdrant": "connected",
      "python_rag": "connected"
    }
  }
}
```

### 获取系统配置

```
GET /api/v1/system/config
```

响应:
```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "models": [
      {"id": "gpt-4o", "name": "GPT-4o"},
      {"id": "gpt-4o-mini", "name": "GPT-4o Mini"},
      {"id": "deepseek-chat", "name": "DeepSeek"}
    ],
    "file_types": ["doc", "docx", "xls", "xlsx", "md", "txt", "pdf", "mp3", "wav", "mp4"],
    "max_file_size": 104857600
  }
}
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 未找到资源 |
| 1003 | 权限不足 |
| 2001 | 文件上传失败 |
| 2002 | 不支持的文件类型 |
| 2003 | 文件过大 |
| 3001 | RAG 服务调用失败 |
| 3002 | 向量操作失败 |
| 4001 | LLM 调用失败 |
| 5001 | 数据库错误 |
| 5002 | WebSocket 错误 |
| 9999 | 系统内部错误 |
