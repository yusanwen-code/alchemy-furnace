import type { ChatSession } from '@/services/types'

/** 单聊会话按道人分组后的父级 */
export interface SingleSessionGroup {
  agentId: string
  agentName: string
  agentAvatar?: string
  /** 组内最新子会话的 updated_at，父组排序依据 */
  latestUpdatedAt: string
  sessions: ChatSession[]
}

/** 会话种类：group 为围炉论道，其余（含历史缺省类型）均为对谈 */
export function sessionKind(session: ChatSession): 'single' | 'group' {
  return session.type === 'group' ? 'group' : 'single'
}

/**
 * 把对谈会话按 agent_id 分组：
 * - 组内会话按 updated_at 倒序
 * - 父组按组内最新子会话时间倒序
 * - agentName 只取 `agent_name || ''`，绝不把 agent_id 复制为可见名称
 */
export function groupSingleSessions(sessions: ChatSession[]): SingleSessionGroup[] {
  const byAgent = new Map<string, SingleSessionGroup>()
  for (const s of sessions) {
    if (sessionKind(s) !== 'single') continue
    const group = byAgent.get(s.agent_id)
    if (group) {
      group.sessions.push(s)
      if (s.updated_at > group.latestUpdatedAt) group.latestUpdatedAt = s.updated_at
    } else {
      byAgent.set(s.agent_id, {
        agentId: s.agent_id,
        agentName: s.agent_name ?? '',
        agentAvatar: s.agent_avatar,
        latestUpdatedAt: s.updated_at,
        sessions: [s],
      })
    }
  }
  const groups = [...byAgent.values()]
  for (const group of groups) {
    group.sessions.sort((a, b) => b.updated_at.localeCompare(a.updated_at))
  }
  groups.sort((a, b) => b.latestUpdatedAt.localeCompare(a.latestUpdatedAt))
  return groups
}
