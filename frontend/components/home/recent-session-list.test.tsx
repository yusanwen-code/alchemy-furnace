import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { RecentSessionList } from '@/components/home/recent-session-list'
import type { ChatSession } from '@/services/types'

vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

const single: ChatSession = {
  id: '11111111-1111-4111-8111-111111111111',
  type: 'single',
  agent_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  agent_name: '太上老君',
  title: '炼丹之问',
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-21T00:00:00Z',
}

const group: ChatSession = {
  id: '22222222-2222-4222-8222-222222222222',
  type: 'group',
  agent_id: '',
  title: '围炉夜话',
  members: [
    { agent_id: 'member-a', name: 'A', proactivity: 1 },
    { agent_id: 'member-b', name: 'B', proactivity: 2 },
  ],
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-22T00:00:00Z',
}

describe('recent session list', () => {
  afterEach(() => cleanup())

  it('shows authoritative single-agent names and never renders UUIDs', () => {
    render(<RecentSessionList sessions={[single]} />)

    expect(screen.getByText('太上老君')).toBeInTheDocument()
    expect(screen.getByText('炼丹之问')).toBeInTheDocument()
    expect(document.body.textContent).not.toContain('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa')
  })

  it('falls back to unknownAgent when identity is missing, without rendering the UUID', () => {
    render(<RecentSessionList sessions={[{ ...single, agent_name: undefined }]} />)

    expect(screen.getByText('unknownAgent')).toBeInTheDocument()
    expect(document.body.textContent).not.toContain('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa')
  })

  it('shows group identity with member count for group sessions', () => {
    render(<RecentSessionList sessions={[group]} />)

    expect(screen.getByText('groupIdentity')).toBeInTheDocument()
    expect(document.body.textContent).not.toContain('22222222-2222-4222-8222-222222222222')
    expect(screen.queryByText('太上老君')).not.toBeInTheDocument()
  })

  it('renders only the four most recent sessions', () => {
    const five: ChatSession[] = Array.from({ length: 5 }, (_, index) => ({
      ...single,
      id: `99999999-9999-4999-8999-99999999990${index}`,
      title: `discourse ${index}`,
      agent_id: `aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa${index}`,
      agent_name: `Daoist ${index}`,
    }))

    render(<RecentSessionList sessions={five} />)

    expect(screen.getByText('discourse 0')).toBeInTheDocument()
    expect(screen.queryByText('discourse 4')).not.toBeInTheDocument()
  })
})
