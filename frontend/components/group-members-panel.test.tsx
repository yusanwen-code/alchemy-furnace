import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { GroupMembersPanel } from '@/components/group-members-panel'
import type { Agent, ChatSession } from '@/services/types'

const doubles = vi.hoisted(() => ({
  fetchAgents: vi.fn(),
  inviteMembers: vi.fn(),
  kickMember: vi.fn(),
  agents: [] as Agent[],
}))

vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('@/contexts/AgentContext', () => ({
  useAgent: () => ({
    state: {
      agents: doubles.agents,
      total: doubles.agents.length,
      currentAgent: null,
      loading: false,
      error: null,
    },
    fetchAgents: doubles.fetchAgents,
  }),
}))

vi.mock('@/contexts/ChatContext', () => ({
  useChat: () => ({
    inviteMembers: doubles.inviteMembers,
    kickMember: doubles.kickMember,
  }),
}))

const groupSession: ChatSession = {
  id: '22222222-2222-4222-8222-222222222222',
  type: 'group',
  agent_id: '',
  title: 'Furnace circle',
  members: [
    { agent_id: 'agent-a', name: 'Alpha', proactivity: 60, avatar: 'https://example.com/alpha.png' },
    { agent_id: 'agent-b', name: 'Beta', proactivity: 70 },
  ],
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
}

describe('GroupMembersPanel avatar handling', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    doubles.fetchAgents.mockResolvedValue(undefined)
    doubles.agents = [
      {
        id: 'agent-c',
        name: 'Candidate',
        model_name: 'model-c',
        status: 'active',
        proactivity: 50,
        created_at: '2026-08-20T00:00:00Z',
        avatar: 'https://example.com/candidate.png',
      },
    ]
  })

  afterEach(() => cleanup())

  it('renders member avatar images for valid URLs', () => {
    render(<GroupMembersPanel session={groupSession} open onClose={() => {}} />)
    expect(screen.getByRole('img', { name: 'Alpha' })).toHaveAttribute('src', 'https://example.com/alpha.png')
  })

  it('falls back to the member initial when the avatar image errors', () => {
    render(<GroupMembersPanel session={groupSession} open onClose={() => {}} />)
    fireEvent.error(screen.getByRole('img', { name: 'Alpha' }))
    expect(screen.queryByRole('img')).toBeNull()
    const anchor = screen.getByRole('button', { name: '查看 Alpha 简介' })
    expect(within(anchor).getByText('A')).toBeInTheDocument()
  })

  it('does not create an img for members without avatar', () => {
    render(<GroupMembersPanel session={groupSession} open onClose={() => {}} />)
    const betaAnchor = screen.getByRole('button', { name: '查看 Beta 简介' })
    expect(within(betaAnchor).queryByRole('img')).toBeNull()
    expect(within(betaAnchor).getByText('B')).toBeInTheDocument()
  })

  it('renders candidate avatars in the invite list and falls back on error', async () => {
    const user = userEvent.setup()
    render(<GroupMembersPanel session={groupSession} open onClose={() => {}} />)
    await user.click(screen.getByRole('button', { name: 'invite' }))
    const img = screen.getByRole('img', { name: 'Candidate' })
    expect(img).toHaveAttribute('src', 'https://example.com/candidate.png')
    fireEvent.error(img)
    expect(screen.queryByRole('img', { name: 'Candidate' })).toBeNull()
    expect(screen.getByText('C')).toBeInTheDocument()
  })
})
