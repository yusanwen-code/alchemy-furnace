import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { MentionSuggest } from '@/components/mention-suggest'
import type { GroupMember } from '@/services/types'

vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

const candidates: GroupMember[] = [
  { agent_id: 'agent-a', name: 'Alpha', proactivity: 60, avatar: 'https://example.com/alpha.png' },
  { agent_id: 'agent-b', name: 'Beta', proactivity: 70 },
]

function renderSuggest() {
  return render(
    <MentionSuggest
      candidates={candidates}
      activeIndex={0}
      onPick={() => {}}
      onHover={() => {}}
      position={{ bottom: 40, left: 24 }}
    />,
  )
}

describe('MentionSuggest avatar handling', () => {
  beforeEach(() => {
    Element.prototype.scrollIntoView = vi.fn()
  })

  afterEach(() => cleanup())

  it('renders candidate avatar images at the fixed anchor position', () => {
    renderSuggest()
    expect(screen.getByRole('img', { name: 'Alpha' })).toHaveAttribute('src', 'https://example.com/alpha.png')
    expect(screen.getByRole('listbox')).toHaveStyle({ position: 'fixed', bottom: '40px', left: '24px' })
  })

  it('falls back to the candidate initial on image error without changing row size or position', () => {
    renderSuggest()
    fireEvent.error(screen.getByRole('img', { name: 'Alpha' }))
    expect(screen.queryByRole('img')).toBeNull()
    const initial = screen.getByText('A')
    expect(initial.className).toContain('h-8 w-8')
    expect(screen.getByRole('listbox')).toHaveStyle({ position: 'fixed', bottom: '40px', left: '24px' })
  })

  it('does not create an img for candidates without avatar', () => {
    renderSuggest()
    const betaOption = screen.getByRole('option', { name: /Beta/ })
    expect(within(betaOption).queryByRole('img')).toBeNull()
    expect(within(betaOption).getByText('B')).toBeInTheDocument()
  })
})
