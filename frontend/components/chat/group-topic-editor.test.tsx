import { act, cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { GroupTopicEditor } from '@/components/chat/group-topic-editor'
import type { ChatSession } from '@/services/types'

vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

const groupSession: ChatSession = {
  id: '22222222-2222-4222-8222-222222222222',
  type: 'group',
  agent_id: '',
  title: '旧主题',
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

describe('GroupTopicEditor', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it('trims and saves a valid topic', async () => {
    const onRename = vi.fn().mockResolvedValue({ ...groupSession, title: '新主题' })
    const user = userEvent.setup()
    render(<GroupTopicEditor sessionId="group-1" title="旧主题" onRename={onRename} />)
    await user.click(screen.getByRole('button', { name: 'rename' }))
    await user.clear(screen.getByLabelText('renameLabel'))
    await user.type(screen.getByLabelText('renameLabel'), '  新主题  ')
    await user.click(screen.getByRole('button', { name: 'saveRename' }))
    expect(onRename).toHaveBeenCalledWith('group-1', '新主题')
    expect(screen.queryByLabelText('renameLabel')).not.toBeInTheDocument()
  })

  it('keeps the draft open after a failed rename', async () => {
    const onRename = vi.fn().mockResolvedValue(null)
    const user = userEvent.setup()
    render(<GroupTopicEditor sessionId="group-1" title="旧主题" onRename={onRename} />)
    await user.click(screen.getByRole('button', { name: 'rename' }))
    await user.clear(screen.getByLabelText('renameLabel'))
    await user.type(screen.getByLabelText('renameLabel'), '仍要保存的主题')
    await user.click(screen.getByRole('button', { name: 'saveRename' }))
    expect(screen.getByLabelText('renameLabel')).toHaveValue('仍要保存的主题')
    expect(screen.getByRole('alert')).toHaveTextContent('renameError')
  })

  it('blocks a blank topic with zero API calls', async () => {
    const onRename = vi.fn()
    const user = userEvent.setup()
    render(<GroupTopicEditor sessionId="group-1" title="旧主题" onRename={onRename} />)
    await user.click(screen.getByRole('button', { name: 'rename' }))
    await user.clear(screen.getByLabelText('renameLabel'))
    await user.type(screen.getByLabelText('renameLabel'), '    ')
    await user.click(screen.getByRole('button', { name: 'saveRename' }))
    expect(screen.getByRole('alert')).toHaveTextContent('emptyError')
    expect(onRename).not.toHaveBeenCalled()
  })

  it('blocks a topic longer than 200 characters with zero API calls', async () => {
    const onRename = vi.fn()
    const user = userEvent.setup()
    render(<GroupTopicEditor sessionId="group-1" title="旧主题" onRename={onRename} />)
    await user.click(screen.getByRole('button', { name: 'rename' }))
    await user.clear(screen.getByLabelText('renameLabel'))
    await user.type(screen.getByLabelText('renameLabel'), '丹'.repeat(201))
    await user.click(screen.getByRole('button', { name: 'saveRename' }))
    expect(screen.getByRole('alert')).toHaveTextContent('tooLongError')
    expect(onRename).not.toHaveBeenCalled()
  })

  it('cancels with Escape and discards the draft', async () => {
    const onRename = vi.fn()
    const user = userEvent.setup()
    render(<GroupTopicEditor sessionId="group-1" title="旧主题" onRename={onRename} />)
    await user.click(screen.getByRole('button', { name: 'rename' }))
    await user.clear(screen.getByLabelText('renameLabel'))
    await user.type(screen.getByLabelText('renameLabel'), '改了一半')
    await user.keyboard('{Escape}')
    expect(screen.queryByLabelText('renameLabel')).not.toBeInTheDocument()
    expect(screen.getByText('旧主题')).toBeInTheDocument()
    expect(onRename).not.toHaveBeenCalled()
  })

  it('blocks duplicate saves while a rename is in flight', async () => {
    const pending = deferred<ChatSession>()
    const onRename = vi.fn().mockReturnValue(pending.promise)
    const user = userEvent.setup()
    render(<GroupTopicEditor sessionId="group-1" title="旧主题" onRename={onRename} />)
    await user.click(screen.getByRole('button', { name: 'rename' }))
    await user.clear(screen.getByLabelText('renameLabel'))
    await user.type(screen.getByLabelText('renameLabel'), '新主题')
    const save = screen.getByRole('button', { name: 'saveRename' })
    await user.click(save)
    await user.click(save)
    expect(onRename).toHaveBeenCalledTimes(1)
    await act(async () => {
      pending.resolve({ ...groupSession, title: '新主题' })
      await pending.promise
    })
    expect(screen.queryByLabelText('renameLabel')).not.toBeInTheDocument()
  })
})
