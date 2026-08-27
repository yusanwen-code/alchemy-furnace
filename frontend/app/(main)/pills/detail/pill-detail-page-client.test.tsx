import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

// 适配层只关心:把校验后的查询 UUID 透传给 PillDetailPage。
// 不得 mock @/lib/entity-detail-route —— 适配层的契约就是调用真实解析函数。
const pageSpy = vi.fn()

vi.mock('./pill-detail', () => ({
  default: (props: { pillId?: string }) => {
    pageSpy(props)
    return <div data-testid="pill-detail-probe">{props.pillId ?? 'invalid'}</div>
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

describe('PillDetailPageClient', () => {
  it('passes a valid query UUID into PillDetailPage', () => {
    searchParams = new URLSearchParams('id=11111111-1111-4111-8111-111111111111')
    return import('./pill-detail-page-client').then(({ PillDetailPageClient }) => {
      render(<PillDetailPageClient />)
      expect(screen.getByTestId('pill-detail-probe')).toHaveTextContent(
        '11111111-1111-4111-8111-111111111111',
      )
    })
  })

  it('passes undefined for a missing or malformed id', () => {
    searchParams = new URLSearchParams('id=_')
    return import('./pill-detail-page-client').then(({ PillDetailPageClient }) => {
      render(<PillDetailPageClient />)
      expect(screen.getByTestId('pill-detail-probe')).toHaveTextContent('invalid')
    })
  })
})
