import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, expectTypeOf, it, vi } from 'vitest'
import { ActionFeedback, type ActionFeedbackProps } from './action-feedback'

describe('ActionFeedback', () => {
  afterEach(() => cleanup())

  it('announces submission progress', () => {
    render(<ActionFeedback status="submitting" message="正在提交" />)

    expect(screen.getByRole('status')).toHaveTextContent('正在提交')
  })

  it('announces successful completion (new: success variant)', () => {
    render(<ActionFeedback status="success" message="已炼制 1 枚" />)

    expect(screen.getByRole('status')).toHaveTextContent('已炼制 1 枚')
  })

  it('shows the error without a retry button when no onRetry given (new: optional retry)', () => {
    render(<ActionFeedback status="error" message="炼制失败" />)

    expect(screen.getByRole('alert')).toHaveTextContent('炼制失败')
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('keeps the error visible and retries the failed action', async () => {
    const retry = vi.fn()
    render(
      <ActionFeedback
        status="error"
        message="模型未配置"
        retryLabel="Retry"
        onRetry={retry}
      />,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('模型未配置')
    await userEvent.click(screen.getByRole('button', { name: /retry/i }))
    expect(retry).toHaveBeenCalledOnce()
  })

  it('requires localized retry details only when onRetry is provided', () => {
    expectTypeOf<Extract<ActionFeedbackProps, { status: 'error' }>>().toEqualTypeOf<{
      status: 'error'
      message: string
      retryLabel?: string
      onRetry?: () => void
    }>()
    expectTypeOf<Extract<ActionFeedbackProps, { status: 'success' }>>().toEqualTypeOf<{
      status: 'success'
      message: string
    }>()
  })
})
