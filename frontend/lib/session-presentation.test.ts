import { describe, expect, it } from 'vitest'

import { groupSingleSessions, sessionKind } from '@/lib/session-presentation'
import type { ChatSession } from '@/services/types'

function session(
  id: string,
  type: ChatSession['type'],
  agentId: string,
  agentName: string | undefined,
  updatedAt: string,
): ChatSession {
  return {
    id,
    type,
    agent_id: agentId,
    agent_name: agentName,
    title: `title ${id}`,
    created_at: updatedAt,
    updated_at: updatedAt,
  }
}

const groupSession: ChatSession = {
  id: 'group-1',
  type: 'group',
  agent_id: '',
  title: 'Furnace circle',
  members: [{ agent_id: 'agent-a', name: 'Alpha', proactivity: 60 }],
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-25T00:00:00Z',
}

describe('session presentation', () => {
  it('classifies sessions by kind', () => {
    expect(sessionKind(session('a', undefined, 'x', 'A', '2026-08-20T00:00:00Z'))).toBe('single')
    expect(sessionKind(session('b', 'single', 'y', 'B', '2026-08-20T00:00:00Z'))).toBe('single')
    expect(sessionKind(groupSession)).toBe('group')
  })

  it('groups legacy and single sessions by agent and sorts newest first', () => {
    const groups = groupSingleSessions([
      session('old-a', undefined, 'agent-a', 'Alpha', '2026-08-20T00:00:00Z'),
      session('new-b', 'single', 'agent-b', 'Beta', '2026-08-23T00:00:00Z'),
      session('new-a', 'single', 'agent-a', 'Alpha', '2026-08-22T00:00:00Z'),
      groupSession,
    ])
    expect(groups.map(group => group.agentId)).toEqual(['agent-b', 'agent-a'])
    expect(groups[1].sessions.map(item => item.id)).toEqual(['new-a', 'old-a'])
  })

  it('never turns agent ids into visible names', () => {
    const groups = groupSingleSessions([session('x', 'single', 'secret-uuid', undefined, '2026-08-20T00:00:00Z')])
    expect(groups[0].agentName).toBe('')
  })
})
