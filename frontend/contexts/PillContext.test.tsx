import { act, cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { PillProvider, usePill } from '@/contexts/PillContext'
import { ApiError } from '@/services/api'
import type { Pill } from '@/services/types'

const td = vi.hoisted(() => ({
  getPill: vi.fn(),
}))

vi.mock('@/services/pillService', () => ({
  getPill: td.getPill,
}))

const PILL_A_ID = '88888888-8888-4888-8888-888888888888'
const PILL_B_ID = '99999999-9999-4999-8999-999999999999'

const pillA: Pill = {
  id: PILL_A_ID,
  name: '丹心妙语',
  description: '温润如茶',
  skill_schema: {},
  tags: [],
  author: '太上老君',
  version: '1.0.0',
  is_builtin: false,
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
}

const pillB: Pill = { ...pillA, id: PILL_B_ID, name: '浩然正气' }

const renderLog: string[] = []

/** 展示详情加载状态机 + 当前金丹,便于断言竞态守卫 */
function Probe() {
  const { state } = usePill()
  const { detailLoad, currentPill } = state
  renderLog.push(`load=${detailLoad.id}:${detailLoad.status}`)
  return (
    <div>
      <span data-testid="detail-id">{detailLoad.id ?? 'none'}</span>
      <span data-testid="detail-status">{detailLoad.status}</span>
      <span data-testid="detail-error">{detailLoad.error ?? 'none'}</span>
      <span data-testid="current-pill">{currentPill ? currentPill.name : 'none'}</span>
    </div>
  )
}

function Actions() {
  const { fetchPill } = usePill()
  return (
    <div>
      <button data-testid="fetch-a" onClick={() => void fetchPill(PILL_A_ID)} />
      <button data-testid="fetch-b" onClick={() => void fetchPill(PILL_B_ID)} />
    </div>
  )
}

describe('PillContext.fetchPill', () => {
  afterEach(() => cleanup())

  beforeEach(() => {
    vi.resetAllMocks()
    renderLog.length = 0
  })

  it('成功:置 ready 并填充 currentPill', async () => {
    td.getPill.mockResolvedValue(pillA)
    render(
      <PillProvider>
        <Probe />
        <Actions />
      </PillProvider>,
    )

    act(() => {
      screen.getByTestId('fetch-a').click()
    })
    expect(screen.getByTestId('detail-status')).toHaveTextContent('loading')

    expect(await screen.findByText('丹心妙语')).toBeInTheDocument()
    expect(screen.getByTestId('detail-id')).toHaveTextContent(PILL_A_ID)
    expect(screen.getByTestId('detail-status')).toHaveTextContent('ready')
    expect(screen.getByTestId('detail-error')).toHaveTextContent('none')
  })

  it('仅 API 明确 404 才判定不存在:not-found 且不残留旧金丹', async () => {
    td.getPill.mockRejectedValue(new ApiError('not found', 404))
    render(
      <PillProvider>
        <Probe />
        <Actions />
      </PillProvider>,
    )

    act(() => {
      screen.getByTestId('fetch-a').click()
    })
    await act(async () => {})

    expect(screen.getByTestId('detail-id')).toHaveTextContent(PILL_A_ID)
    expect(screen.getByTestId('detail-status')).toHaveTextContent('not-found')
    expect(screen.getByTestId('detail-error')).toHaveTextContent('none')
    expect(screen.getByTestId('current-pill')).toHaveTextContent('none')
  })

  it('其他错误:置 error 并携带消息,不误判不存在', async () => {
    td.getPill.mockRejectedValue(new ApiError('服务器内部错误', 500))
    render(
      <PillProvider>
        <Probe />
        <Actions />
      </PillProvider>,
    )

    act(() => {
      screen.getByTestId('fetch-a').click()
    })
    expect(await screen.findByText('服务器内部错误')).toBeInTheDocument()
    expect(screen.getByTestId('detail-status')).toHaveTextContent('error')
    expect(screen.getByTestId('current-pill')).toHaveTextContent('none')
  })

  it('竞态:先发的旧请求晚到时不覆盖新页(状态与金丹都不被旧结果污染)', async () => {
    let resolveA!: (p: Pill) => void
    let resolveB!: (p: Pill) => void
    td.getPill.mockImplementation((id: string) =>
      id === PILL_A_ID
        ? new Promise<Pill>((res) => { resolveA = res })
        : new Promise<Pill>((res) => { resolveB = res }),
    )
    render(
      <PillProvider>
        <Probe />
        <Actions />
      </PillProvider>,
    )

    // 先请求 A 再快速切到 B,让 B 先落地
    act(() => {
      screen.getByTestId('fetch-a').click()
      screen.getByTestId('fetch-b').click()
    })
    await act(async () => {
      resolveB(pillB)
    })
    expect(screen.getByTestId('detail-id')).toHaveTextContent(PILL_B_ID)
    expect(screen.getByTestId('detail-status')).toHaveTextContent('ready')
    expect(screen.getByTestId('current-pill')).toHaveTextContent('浩然正气')

    // A 的旧结果晚到:守卫拒绝,页面仍停留在 B
    await act(async () => {
      resolveA(pillA)
    })
    expect(screen.getByTestId('detail-id')).toHaveTextContent(PILL_B_ID)
    expect(screen.getByTestId('detail-status')).toHaveTextContent('ready')
    expect(screen.getByTestId('current-pill')).toHaveTextContent('浩然正气')
    expect(screen.queryByText('丹心妙语')).toBeNull()
  })

  it('竞态:旧请求的 404/错误晚到同样被守卫拒绝', async () => {
    let resolveA!: (p: Pill) => void
    let rejectB!: (e: Error) => void
    td.getPill.mockImplementation((id: string) =>
      id === PILL_A_ID
        ? new Promise<Pill>((res) => { resolveA = res })
        : new Promise<Pill>((_, rej) => { rejectB = rej }),
    )
    render(
      <PillProvider>
        <Probe />
        <Actions />
      </PillProvider>,
    )

    act(() => {
      screen.getByTestId('fetch-a').click()
      screen.getByTestId('fetch-b').click()
    })
    await act(async () => {
      rejectB(new ApiError('not found', 404))
    })
    expect(screen.getByTestId('detail-status')).toHaveTextContent('not-found')

    // A 的成功结果晚到:不得把已判定 not-found 的页面改回 ready
    await act(async () => {
      resolveA(pillA)
    })
    expect(screen.getByTestId('detail-status')).toHaveTextContent('not-found')
    expect(screen.getByTestId('current-pill')).toHaveTextContent('none')
  })
})
