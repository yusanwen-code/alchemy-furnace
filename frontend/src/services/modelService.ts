/**
 * 模型服务 - LLM 模型管理 API
 * 对接后端 /api/v1/models（统一响应封装 { code, message, data }）
 * 注意：api_key 仅用于写入（创建/更新），接口永不明文返回，前端不得记录或展示明文密钥
 */
import { get, post, put, del } from './api'
import type { PagedList } from './types'

/** 模型服务商 */
export type ModelProvider = 'openai' | 'deepseek' | 'aliyun' | 'ollama' | 'other'

/** LLM 模型（api_key 仅以掩码形式返回） */
export interface LLMModel {
  id: number
  /** 模型标识（如 gpt-4o） */
  name: string
  /** 显示名 */
  display_name: string
  provider: ModelProvider | string
  base_url?: string
  /** 掩码后的 API Key（如 sk-****wxyz） */
  api_key_masked?: string
  /** 是否已配置 API Key */
  has_api_key: boolean
  temperature: number
  max_tokens: number
  is_enabled: boolean
  /** 是否为默认模型 */
  is_default: boolean
  /** 是否为语言模式合成模型 */
  is_synthesis: boolean
  sort_order: number
  /** 引用该模型的道人数量 */
  referenced_by: number
  created_at: string
  updated_at: string
}

/** 道人表单下拉用的精简模型选项 */
export interface ModelOption {
  name: string
  display_name: string
  provider: string
  is_default: boolean
}

/** 模型列表查询参数 */
export interface ModelListParams {
  enabled?: boolean
  page?: number
  page_size?: number
}

/** 创建模型请求（api_key 为明文，仅用于写入） */
export interface CreateModelRequest {
  name: string
  display_name?: string
  provider: ModelProvider | string
  base_url?: string
  api_key?: string
  temperature?: number
  max_tokens?: number
  is_enabled?: boolean
  is_default?: boolean
  is_synthesis?: boolean
  sort_order?: number
}

/** 更新模型请求（全字段可选；api_key 不传 = 不修改，传空字符串 = 清除密钥） */
export type UpdateModelRequest = Partial<CreateModelRequest>

/** 连接测试结果 */
export interface TestConnectionResult {
  success: boolean
  latency_ms: number
  error: string
}

/**
 * 获取模型列表
 */
export function list(params: ModelListParams = {}): Promise<PagedList<LLMModel>> {
  return get<PagedList<LLMModel>>('/models', {
    page: params.page ?? 1,
    page_size: params.page_size ?? 50,
    enabled: params.enabled,
  })
}

/**
 * 创建模型
 */
export function create(data: CreateModelRequest): Promise<LLMModel> {
  return post<LLMModel>('/models', data)
}

/**
 * 更新模型
 */
export function update(id: number, data: UpdateModelRequest): Promise<LLMModel> {
  return put<LLMModel>(`/models/${id}`, data)
}

/**
 * 删除模型
 * 若模型仍被道人引用，后端返回 409（message 为中文描述）
 */
export function remove(id: number): Promise<void> {
  return del<void>(`/models/${id}`)
}

/**
 * 测试模型连接（以该模型凭证发起一次最小 LLM 调用）
 */
export function testConnection(id: number): Promise<TestConnectionResult> {
  return post<TestConnectionResult>(`/models/${id}/test-connection`)
}

/**
 * 获取已启用模型的精简列表（道人表单下拉用）
 */
export function options(): Promise<ModelOption[]> {
  return get<ModelOption[]>('/models/options')
}
