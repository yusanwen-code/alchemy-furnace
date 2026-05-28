/**
 * 道人服务 - AI Agent 管理 API
 * 提供道人的增删改查及金丹绑定操作（演示模式使用 Mock 数据）
 */
import { mockDelay } from './api'
import { mockAgents, mockAgentPills, mockPills } from './mockData'
import type { Agent, CreateAgentRequest, Pill } from './types'

let agents = [...mockAgents]
let agentPills = { ...mockAgentPills }
let nextAgentId = Math.max(...agents.map(a => a.id)) + 1

/**
 * 获取道人列表
 */
export async function getAgents(): Promise<Agent[]> {
  await mockDelay()
  return [...agents].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
}

/**
 * 获取单个道人详情
 */
export async function getAgent(id: number): Promise<Agent> {
  await mockDelay()
  const agent = agents.find(a => a.id === id)
  if (!agent) throw new Error('道人不存在')
  return { ...agent }
}

/**
 * 创建道人
 */
export async function createAgent(data: CreateAgentRequest): Promise<Agent> {
  await mockDelay(600)
  const agent: Agent = {
    id: nextAgentId++,
    name: data.name,
    personality: data.personality,
    model_name: data.model_name,
    status: 'active',
    created_at: new Date().toISOString(),
  }
  agents.push(agent)
  agentPills[agent.id] = []
  return { ...agent }
}

/**
 * 更新道人
 */
export async function updateAgent(id: number, data: Partial<CreateAgentRequest>): Promise<Agent> {
  await mockDelay()
  const index = agents.findIndex(a => a.id === id)
  if (index === -1) throw new Error('道人不存在')
  agents[index] = {
    ...agents[index],
    ...data,
  }
  return { ...agents[index] }
}

/**
 * 删除道人
 */
export async function deleteAgent(id: number): Promise<void> {
  await mockDelay()
  agents = agents.filter(a => a.id !== id)
  delete agentPills[id]
}

/**
 * 获取道人已服用的金丹列表
 */
export async function getAgentPills(agentId: number): Promise<Pill[]> {
  await mockDelay()
  const pillIds = agentPills[agentId] || []
  return pillIds
    .map(id => mockPills.find(p => p.id === id))
    .filter((p): p is Pill => p !== undefined)
}

/**
 * 道人服用金丹（绑定）
 */
export async function bindPill(agentId: number, pillId: number): Promise<void> {
  await mockDelay()
  if (!agentPills[agentId]) agentPills[agentId] = []
  if (!agentPills[agentId].includes(pillId)) {
    agentPills[agentId].push(pillId)
  }
}

/**
 * 道人解除金丹绑定
 */
export async function unbindPill(agentId: number, pillId: number): Promise<void> {
  await mockDelay()
  if (agentPills[agentId]) {
    agentPills[agentId] = agentPills[agentId].filter(id => id !== pillId)
  }
}
