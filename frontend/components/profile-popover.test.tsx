import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { ProfilePopover } from '@/components/profile-popover'
import type { Agent } from '@/services/types'
import type { UserProfile } from '@/services/userService'

vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

const agentWithAvatar: Agent = {
  id: 'agent-1',
  name: '太上老君',
  avatar: 'https://example.com/laojun.png',
  personality: '沉稳如山',
  model_name: 'gpt-4o',
  status: 'active',
  proactivity: 60,
  created_at: '2026-08-20T00:00:00Z',
}

const userProfile: UserProfile = {
  display_name: '炼丹者',
  bio: '',
  avatar: 'https://example.com/user.png',
  updated_at: '2026-08-20T00:00:00Z',
}

/** 渲染 agent 弹窗:anchor 挂在 body 上,popover 经 portal 也渲染到 body */
function renderAgentPopover(agent: Agent | null) {
  const anchor = document.createElement('button')
  anchor.setAttribute('aria-label', 'avatar-anchor')
  document.body.appendChild(anchor)
  const anchorRef = { current: anchor as HTMLElement }
  render(<ProfilePopover kind="agent" anchorRef={anchorRef} open onClose={() => {}} agent={agent} />)
  return anchor
}

function renderUserPopover(profile: UserProfile | null) {
  const anchor = document.createElement('button')
  anchor.setAttribute('aria-label', 'avatar-anchor')
  document.body.appendChild(anchor)
  const anchorRef = { current: anchor as HTMLElement }
  render(
    <ProfilePopover
      kind="user"
      anchorRef={anchorRef}
      open
      onClose={() => {}}
      userProfile={profile}
    />,
  )
  return anchor
}

afterEach(() => {
  cleanup()
  document.body.innerHTML = ''
})

describe('ProfilePopover avatar handling', () => {
  it('renders the agent avatar image for a valid URL', () => {
    renderAgentPopover(agentWithAvatar)
    expect(screen.getByRole('img', { name: '太上老君' })).toHaveAttribute('src', 'https://example.com/laojun.png')
  })

  it('falls back to the initial on image error without changing avatar size or popover position', () => {
    renderAgentPopover(agentWithAvatar)
    fireEvent.error(screen.getByRole('img', { name: '太上老君' }))
    expect(screen.queryByRole('img')).toBeNull()
    const initial = screen.getByText('太')
    expect(initial.className).toContain('h-12 w-12')
    expect(initial.className).toContain('rounded-full')
    expect(screen.getByRole('dialog')).toHaveStyle({ position: 'fixed' })
  })

  it('does not create an img when the agent has no avatar', () => {
    renderAgentPopover({ ...agentWithAvatar, avatar: '' })
    expect(screen.queryByRole('img')).toBeNull()
    expect(screen.getByText('太')).toBeInTheDocument()
  })

  it('renders the user avatar image and falls back to the initial on error', () => {
    renderUserPopover(userProfile)
    const img = screen.getByRole('img', { name: '炼丹者' })
    expect(img).toHaveAttribute('src', 'https://example.com/user.png')
    fireEvent.error(img)
    expect(screen.queryByRole('img')).toBeNull()
    expect(screen.getByText('炼')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toHaveStyle({ position: 'fixed' })
  })

  it('does not create an img when the user has no avatar', () => {
    renderUserPopover({ ...userProfile, avatar: '' })
    expect(screen.queryByRole('img')).toBeNull()
    expect(screen.getByText('炼')).toBeInTheDocument()
  })
})
