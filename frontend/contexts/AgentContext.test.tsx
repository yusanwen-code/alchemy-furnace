import { act, cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgentProvider, useAgent } from '@/contexts/AgentContext'
import { ApiError } from '@/services/api'
import type { AgentDetail } from '@/services/types'

const td = vi.hoisted(() => ({
  getAgent: vi.fn(),
}))

vi.mock('@/services/agentService', () => ({
  getAgent: td.getAgent,
}))

const AGENT_ID = '11111111-1111-4111-8111-111111111111'
const OTHER_ID = '22222222-2222-4222-8222-222222222222'

const agentA: AgentDetail = {
  id: AGENT_ID,
  name: '太上老君',
  avatar: '',
  personality: '沉稳如山',
  model_name: 'gpt-4o',
  status: 'active',
  proactivity: 60,
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
  language_pattern: {
    is_valid: false,
    system_prompt: '',
    emergence_rules: [],
    inner_tensions: [],
  },
}

const agentB: AgentDetail = { ...agentA, id: OTHER_ID, name: '元始天尊' }

const renderLog: string[] = []

/** 展示详情加载状态机 + 当前道人,便于断言竞态守卫 */
function Probe() {
  const { state } = useAgent()
  const { detailLoad, currentAgent } = state
  renderLog.push(`load=${detailLoad.id}:${detailLoad.status}`)
  return (
    <div>
      <span data-testid="detail-id">{detailLoad.id ?? 'none'}</span>
      <span data-testid="detail-status">{detailLoad.status}</span>
      <span data-testid="detail-error">{detailLoad.error ?? 'none'}</span>
      <span data-testid="current-agent">{currentAgent ? currentAgent.name : 'none'}</span>
    </div>
  )
}

function Actions() {
  const { fetchAgent } = useAgent()
  return (
    <div>
      <button data-testid="fetch-a" onClick={() => void fetchAgent(AGENT_ID)} />
      <button data-testid="fetch-b" onClick={() => void fetchAgent(OTHER_ID)} />
    </div>
  )
}

describe('AgentContext.fetchAgent', () => {
  afterEach(() => cleanup())

  beforeEach(() => {
    vi.resetAllMocks()
    renderLog.length = 0
  })

  it('成功:置 ready 并填充 currentAgent', async () => {
    td.getAgent.mockResolvedValue(agentA)
    render(
      <AgentProvider>
        <Probe />
        <Actions />
      </AgentProvider>,
    )

    act(() => {
      screen.getByTestId('fetch-a').click()
    })
    expect(screen.getByTestId('detail-status')).toHaveTextContent('loading')

    expect(await screen.findByText('太上老君')).toBeInTheDocument()
    expect(screen.getByTestId('detail-id')).toHaveTextContent(AGENT_ID)
    expect(screen.getByTestId('detail-status')).toHaveTextContent('ready')
    expect(screen.getByTestId('detail-error')).toHaveTextContent('none')
  })

  it('仅 API 明确 404 才判定不存在:not-found 且不残留旧道人', async () => {
    td.getAgent.mockRejectedValue(new ApiError('not found', 404))
    render(
      <AgentProvider>
        <Probe />
        <Actions />
      </AgentProvider>,
    )

    act(() => {
      screen.getByTestId('fetch-a').click()
    })
    await act(async () => {})

    expect(screen.getByTestId('detail-id')).toHaveTextContent(AGENT_ID)
    expect(screen.getByTestId('detail-status')).toHaveTextContent('not-found')
    expect(screen.getByTestId('detail-error')).toHaveTextContent('none')
    expect(screen.getByTestId('current-agent')).toHaveTextContent('none')
  })

  it('其他错误:置 error 并携带消息,不误判不存在', async () => {
    td.getAgent.mockRejectedValue(new ApiError('服务器内部错误', 500))
    render(
      <AgentProvider>
        <Probe />
        <Actions />
      </AgentProvider>,
    )

    act(() => {
      screen.getByTestId('fetch-a').click()
    })
    expect(await screen.findByText('服务器内部错误')).toBeInTheDocument()
    expect(screen.getByTestId('detail-status')).toHaveTextContent('error')
    expect(screen.getByTestId('current-agent')).toHaveTextContent('none')
  })

  it('竞态:先发的旧请求晚到时不覆盖新页(状态与道人都不被旧结果污染)', async () => {
    let resolveA!: (a: AgentDetail) => void
    let resolveB!: (a: AgentDetail) => void
    td.getAgent.mockImplementation((id: string) =>
      id === AGENT_ID
        ? new Promise<AgentDetail>((res) => { resolveA = res })
        : new Promise<AgentDetail>((res) => { resolveB = res }),
    )
    render(
      <AgentProvider>
        <Probe />
        <Actions />
      </AgentProvider>,
    )

    // 先请求 A 再快速切到 B,让 B 先落地
    act(() => {
      screen.getByTestId('fetch-a').click()
      screen.getByTestId('fetch-b').click()
    })
    await act(async () => {
      resolveB(agentB)
    })
    expect(screen.getByTestId('detail-id')).toHaveTextContent(OTHER_ID)
    expect(screen.getByTestId('detail-status')).toHaveTextContent('ready')
    expect(screen.getByTestId('current-agent')).toHaveTextContent('元始天尊')

    // A 的旧结果晚到:守卫拒绝,页面仍停留在 B
    await act(async () => {
      resolveA(agentA)
    })
    expect(screen.getByTestId('detail-id')).toHaveTextContent(OTHER_ID)
    expect(screen.getByTestId('detail-status')).toHaveTextContent('ready')
    expect(screen.getByTestId('current-agent')).toHaveTextContent('元始天尊')
    expect(screen.queryByText('太上老君')).toBeNull()
  })

  it('竞态:旧请求的 404/错误晚到同样被守卫拒绝', async () => {
    let resolveA!: (a: AgentDetail) => void
    let rejectB!: (e: Error) => void
    td.getAgent.mockImplementation((id: string) =>
      id === AGENT_ID
        ? new Promise<AgentDetail>((res) => { resolveA = res })
        : new Promise<AgentDetail>((_, rej) => { rejectB = rej }),
    )
    render(
      <AgentProvider>
        <Probe />
        <Actions />
      </AgentProvider>,
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
      resolveA(agentA)
    })
    expect(screen.getByTestId('detail-status')).toHaveTextContent('not-found')
    expect(screen.getByTestId('current-agent')).toHaveTextContent('none')
  })
})
