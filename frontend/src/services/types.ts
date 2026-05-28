/**
 * 数据类型定义
 * 对应后端数据库模型的前端 TypeScript 类型
 */

/** 金丹状态 */
export type PillStatus = 'refining' | 'refined' | 'failed'

/** 金丹（知识库） */
export interface Pill {
  id: number
  name: string
  description?: string
  status: PillStatus
  vector_count: number
  created_at: string
  updated_at: string
}

/** 提取状态 */
export type ExtractStatus = 'pending' | 'extracting' | 'completed' | 'failed'

/** 丹方（文档文件） */
export interface Recipe {
  id: number
  pill_id: number
  filename: string
  file_type: string
  file_size: number
  file_path?: string
  extract_status: ExtractStatus
  extract_result?: string
  chunk_count: number
  created_at: string
}

/** Agent 状态 */
export type AgentStatus = 'active' | 'inactive'

/** 道人（AI Agent） */
export interface Agent {
  id: number
  name: string
  avatar?: string
  personality?: string
  model_name: string
  status: AgentStatus
  created_at: string
}

/** 服用记录（Agent 绑定金丹） */
export interface AgentPill {
  id: number
  agent_id: number
  pill_id: number
  created_at: string
}

/** 对话会话 */
export interface ChatSession {
  id: number
  agent_id: number
  title?: string
  created_at: string
  updated_at: string
}

/** RAG 引用来源 */
export interface Source {
  content: string
  score: number
  metadata: {
    filename?: string
    page?: number
    chunk_index?: number
    [key: string]: unknown
  }
}

/** 对话消息 */
export interface ChatMessage {
  id: number
  session_id: number
  role: 'user' | 'assistant' | 'system'
  content: string
  sources?: Source[]
  created_at: string
}

/** 模型配置 */
export interface ModelConfig {
  id: string
  name: string
  description: string
  provider: string
}

/** 系统配置 */
export interface SystemConfig {
  api_key?: string
  base_url?: string
  default_model?: string
  models: ModelConfig[]
}

/** 创建金丹请求 */
export interface CreatePillRequest {
  name: string
  description?: string
}

/** 创建道人请求 */
export interface CreateAgentRequest {
  name: string
  personality?: string
  model_name: string
}

/** 创建会话请求 */
export interface CreateSessionRequest {
  agent_id: number
  title?: string
}

/** WebSocket 消息 */
export interface WSMessage {
  type: 'message' | 'chunk' | 'done' | 'error'
  content?: string
  sources?: Source[]
  error?: string
}
