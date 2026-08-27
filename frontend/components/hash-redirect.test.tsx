import { render } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { HashRedirect } from '@/components/hash-redirect'

const replace = vi.hoisted(() => vi.fn())

vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace }),
}))

function renderWithHash(hash: string) {
  window.location.hash = hash
  render(<HashRedirect />)
}

describe('HashRedirect legacy links', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.location.hash = ''
  })

  afterEach(() => {
    window.location.hash = ''
  })

  it('normalizes a legacy #/chat/<uuid> to the canonical query URL', () => {
    renderWithHash('#/chat/11111111-1111-4111-8111-111111111111')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith(
      '/chat?session=11111111-1111-4111-8111-111111111111',
    )
  })

  it('sends the static-export placeholder #/chat/_ back to the lobby', () => {
    renderWithHash('#/chat/_')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith('/chat')
  })

  it('sends a malformed #/chat/<id> back to the lobby without throwing', () => {
    renderWithHash('#/chat/not-a-uuid')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith('/chat')
  })

  it('normalizes a legacy #/agents/<uuid> to the canonical detail query URL', () => {
    renderWithHash('#/agents/11111111-1111-4111-8111-111111111111')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith('/agents/detail?id=11111111-1111-4111-8111-111111111111')
  })

  it('normalizes a legacy #/pills/<uuid> to the canonical detail query URL', () => {
    renderWithHash('#/pills/33333333-3333-4333-8333-333333333333')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith('/pills/detail?id=33333333-3333-4333-8333-333333333333')
  })

  it('sends the static-export placeholder #/agents/_ back to the agent list', () => {
    renderWithHash('#/agents/_')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith('/agents')
  })

  it('sends the static-export placeholder #/pills/_ back to the pill list', () => {
    renderWithHash('#/pills/_')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith('/pills')
  })

  it('sends a malformed #/agents/<id> back to the agent list without throwing', () => {
    renderWithHash('#/agents/not-a-uuid')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith('/agents')
  })

  it('sends a malformed #/pills/<id> back to the pill list without throwing', () => {
    renderWithHash('#/pills/not-a-uuid')
    expect(replace).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith('/pills')
  })
})
