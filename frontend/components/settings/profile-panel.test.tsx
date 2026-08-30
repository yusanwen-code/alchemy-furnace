import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ProfilePanel } from '@/components/settings/profile-panel'
import type { UserProfile } from '@/services/userService'

const td = vi.hoisted(() => ({
  fetchProfile: vi.fn(),
  updateProfile: vi.fn(),
  userState: {
    profile: null as UserProfile | null,
    loading: false,
    error: null as string | null,
  },
}))

vi.mock('@/contexts/UserContext', () => ({
  useUser: () => ({
    profile: td.userState.profile,
    loading: td.userState.loading,
    error: td.userState.error,
    fetchProfile: td.fetchProfile,
    updateProfile: td.updateProfile,
  }),
}))

vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

const profileA: UserProfile = {
  display_name: 'Yao',
  bio: 'Alchemist',
  avatar: '',
  updated_at: '2026-08-20T00:00:00Z',
}

describe('ProfilePanel', () => {
  afterEach(() => cleanup())

  beforeEach(() => {
    vi.resetAllMocks()
    td.userState.profile = null
    td.userState.loading = false
    td.userState.error = null
    td.fetchProfile.mockResolvedValue(undefined)
    td.updateProfile.mockResolvedValue(profileA)
  })

  it('fills the form once the profile arrives', async () => {
    const { rerender } = render(<ProfilePanel />)
    expect(screen.getByLabelText('displayName')).toHaveValue('')

    td.userState.profile = profileA
    rerender(<ProfilePanel />)

    expect(screen.getByLabelText('displayName')).toHaveValue('Yao')
    expect(screen.getByLabelText('bio')).toHaveValue('Alchemist')
  })

  it('keeps in-progress edits while the profile object is unchanged', async () => {
    td.userState.profile = profileA
    const user = userEvent.setup()
    const { rerender } = render(<ProfilePanel />)

    const nameInput = screen.getByLabelText('displayName')
    await user.clear(nameInput)
    await user.type(nameInput, 'Edited')
    expect(nameInput).toHaveValue('Edited')

    // 无关重渲染（同一 profile 引用）不回填表单
    rerender(<ProfilePanel />)
    expect(nameInput).toHaveValue('Edited')
  })

  it('resyncs the form when a refreshed profile object arrives', async () => {
    td.userState.profile = profileA
    const user = userEvent.setup()
    const { rerender } = render(<ProfilePanel />)

    const nameInput = screen.getByLabelText('displayName')
    await user.clear(nameInput)
    await user.type(nameInput, 'Draft')

    td.userState.profile = { ...profileA, display_name: 'From Server' }
    rerender(<ProfilePanel />)

    expect(screen.getByLabelText('displayName')).toHaveValue('From Server')
  })

  it('saves the trimmed display name and bio', async () => {
    td.userState.profile = profileA
    const user = userEvent.setup()
    render(<ProfilePanel />)

    const nameInput = screen.getByLabelText('displayName')
    await user.clear(nameInput)
    await user.type(nameInput, '  New Name  ')
    await user.click(screen.getByRole('button', { name: 'save' }))

    expect(td.updateProfile).toHaveBeenCalledWith({
      display_name: 'New Name',
      bio: 'Alchemist',
      avatar: '',
    })
  })

  it('previews and saves the user avatar', async () => {
    td.userState.profile = { ...profileA, avatar: 'https://example.com/old.png' }
    const user = userEvent.setup()
    render(<ProfilePanel />)

    expect(screen.getByRole('img', { name: 'avatarPreviewAlt' })).toHaveAttribute('src', 'https://example.com/old.png')

    const input = screen.getByLabelText('avatarLabel')
    await user.clear(input)
    await user.type(input, 'https://example.com/new.png')
    await user.click(screen.getByRole('button', { name: 'save' }))

    expect(td.updateProfile).toHaveBeenCalledWith({
      display_name: 'Yao',
      bio: 'Alchemist',
      avatar: 'https://example.com/new.png',
    })
  })

  it('blocks saving an invalid avatar without any API call', async () => {
    td.userState.profile = profileA
    const user = userEvent.setup()
    render(<ProfilePanel />)

    const input = screen.getByLabelText('avatarLabel')
    await user.clear(input)
    await user.type(input, 'javascript:alert(1)')
    await user.click(screen.getByRole('button', { name: 'save' }))

    expect(td.updateProfile).not.toHaveBeenCalled()
    expect(screen.getByText('avatarInvalid')).toBeInTheDocument()
  })

  it('falls back to the initial when the avatar image fails to load', () => {
    td.userState.profile = { ...profileA, avatar: 'https://example.com/old.png' }
    render(<ProfilePanel />)

    fireEvent.error(screen.getByRole('img', { name: 'avatarPreviewAlt' }))

    expect(screen.queryByRole('img', { name: 'avatarPreviewAlt' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('avatarPreviewAlt')).toHaveTextContent('Y')
  })

  it('clears the avatar by submitting an empty value', async () => {
    td.userState.profile = { ...profileA, avatar: 'https://example.com/old.png' }
    const user = userEvent.setup()
    render(<ProfilePanel />)

    const input = screen.getByLabelText('avatarLabel')
    await user.clear(input)
    await user.click(screen.getByRole('button', { name: 'save' }))

    expect(td.updateProfile).toHaveBeenCalledWith({
      display_name: 'Yao',
      bio: 'Alchemist',
      avatar: '',
    })
  })
})
