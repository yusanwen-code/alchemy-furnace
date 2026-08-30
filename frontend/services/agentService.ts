/**
 * 道人服务 - AI Agent 管理 API
 * 对接后端 /api/v1/agents，含服用金丹（绑定/权重/顺序）操作
 */
import { get, post, put, patch, del } from './api'
import type {
  Agent,
  AgentDetail,
  AgentMemory,
  CreateAgentRequest,
  CreateMemoryRequest,
  MemoryKind,
  UpdateAgentRequest,
  UpdateMemoryRequest,
  PagedList,
  AgentListParams,
  ReplacePillsItem,
} from './types'

/**
 * 获取道人列表
 */
export function listAgents(params: AgentListParams = {}): Promise<PagedList<Agent>> {
  return get<PagedList<Agent>>('/agents', {
    page: params.page ?? 1,
    page_size: params.page_size ?? 100,
    status: params.status,
  })
}

/**
 * 获取道人详情（含已服用金丹 agent_pills 与语言模式缓存）
 */
export function getAgent(id: string): Promise<AgentDetail> {
  return get<AgentDetail>(`/agents/${id}`)
}

/**
 * 创建道人
 */
export function createAgent(data: CreateAgentRequest): Promise<Agent> {
  return post<Agent>('/agents', data)
}

/**
 * 更新道人
 * @param memoryEnabled 本地记忆开关;传 undefined 时不携带该字段
 */
export function updateAgent(
  id: string,
  data: UpdateAgentRequest,
  memoryEnabled?: boolean
): Promise<Agent> {
  const payload =
    memoryEnabled === undefined ? data : { ...data, memory_enabled: memoryEnabled }
  return put<Agent>(`/agents/${id}`, payload)
}

/**
 * 删除道人
 */
export function deleteAgent(id: string): Promise<void> {
  return del<void>(`/agents/${id}`)
}

/**
 * 道人服用金丹（绑定），weight 0-10，sort_order >= 0
 */
export function bindPill(agentId: string, pillId: string, weight = 1, sortOrder = 0): Promise<void> {
  return post<void>(`/agents/${agentId}/pills`, {
    pill_id: pillId,
    weight,
    sort_order: sortOrder,
  })
}

/**
 * 更新服用记录（权重/顺序）
 */
export function updateAgentPill(
  agentId: string,
  pillId: string,
  weight: number,
  sortOrder: number
): Promise<void> {
  return put<void>(`/agents/${agentId}/pills/${pillId}`, {
    weight,
    sort_order: sortOrder,
  })
}

/**
 * 道人解除金丹绑定
 */
export function unbindPill(agentId: string, pillId: string): Promise<void> {
  return del<void>(`/agents/${agentId}/pills/${pillId}`)
}

/**
 * 完整替换道人服丹编排（原子）：以传入数组为最终编排
 * 空数组 = 清空全部服用关系；返回写后服务端确认的道人详情
 */
export function replacePills(agentId: string, pills: ReplacePillsItem[]): Promise<AgentDetail> {
  return put<AgentDetail>(`/agents/${agentId}/pills`, { pills })
}

// ========== 本地记忆 ==========

/**
 * 获取道人本地记忆列表(默认仅 active;kind 可选按类型筛选)
 * GET /agents/:uuid/memories
 */
export function fetchAgentMemories(
  agentId: string,
  kind?: MemoryKind,
  active = true
): Promise<AgentMemory[]> {
  return get<AgentMemory[]>(`/agents/${agentId}/memories`, { kind, active })
}

/**
 * 新建道人记忆
 * POST /agents/:uuid/memories
 */
export function createAgentMemory(
  agentId: string,
  input: CreateMemoryRequest
): Promise<AgentMemory> {
  return post<AgentMemory>(`/agents/${agentId}/memories`, input)
}

/**
 * 更新道人记忆(仅传变更字段)
 * PATCH /agents/:uuid/memories/:memory_uuid
 */
export function updateAgentMemory(
  agentId: string,
  memoryUuid: string,
  input: UpdateMemoryRequest
): Promise<AgentMemory> {
  return patch<AgentMemory>(`/agents/${agentId}/memories/${memoryUuid}`, input)
}

/**
 * 删除单条道人记忆(物理删除)
 * DELETE /agents/:uuid/memories/:memory_uuid
 */
export function deleteAgentMemory(agentId: string, memoryUuid: string): Promise<void> {
  return del<void>(`/agents/${agentId}/memories/${memoryUuid}`)
}

/**
 * 清空道人全部记忆(物理删除)
 * DELETE /agents/:uuid/memories
 */
export function clearAgentMemories(agentId: string): Promise<void> {
  return del<void>(`/agents/${agentId}/memories`)
}
