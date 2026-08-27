import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

// 适配层只关心:把校验后的查询 UUID 透传给 AgentDetailPage。
// 不得 mock @/lib/entity-detail-route —— 适配层的契约就是调用真实解析函数。
const pageSpy = vi.fn()

vi.mock('./agent-detail', () => ({
  default: (props: { agentId?: string }) => {
    pageSpy(props)
    return <div data-testid="agent-detail-probe">{props.agentId ?? 'invalid'}</div>
  },
}))

vi.mock('next/navigation', () => ({
  useSearchParams: () => searchParams,
}))

let searchParams: URLSearchParams

afterEach(() => {
  cleanup()
  pageSpy.mockClear()
})

describe('AgentDetailPageClient', () => {
  it('passes a valid query UUID into AgentDetailPage', () => {
    searchParams = new URLSearchParams('id=11111111-1111-4111-8111-111111111111')
    return import('./agent-detail-page-client').then(({ AgentDetailPageClient }) => {
      render(<AgentDetailPageClient />)
      expect(screen.getByTestId('agent-detail-probe')).toHaveTextContent(
        '11111111-1111-4111-8111-111111111111',
      )
    })
  })

  it('passes undefined for a missing or malformed id', () => {
    searchParams = new URLSearchParams('id=_')
    return import('./agent-detail-page-client').then(({ AgentDetailPageClient }) => {
      render(<AgentDetailPageClient />)
      expect(screen.getByTestId('agent-detail-probe')).toHaveTextContent('invalid')
    })
  })
})
