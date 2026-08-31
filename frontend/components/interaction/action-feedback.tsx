export type ActionFeedbackStatus = 'submitting' | 'success' | 'error'

export type ActionFeedbackProps =
  | {
      status: 'submitting'
      message: string
    }
  | {
      status: 'success'
      message: string
    }
  | {
      status: 'error'
      message: string
      /** 缺省时不渲染重试按钮（如 409 版本冲突需刷新，而非重试提交） */
      onRetry?: () => void
      retryLabel?: string
    }

export function ActionFeedback(props: ActionFeedbackProps) {
  if (props.status === 'submitting') {
    return (
      <p role="status" aria-live="polite" className="break-words text-sm text-sage">
        {props.message}
      </p>
    )
  }

  if (props.status === 'success') {
    return (
      <p role="status" aria-live="polite" className="break-words text-sm text-sage">
        {props.message}
      </p>
    )
  }

  return (
    <div role="alert" className="flex min-w-0 flex-wrap items-center gap-2 text-sm text-primary">
      <span className="min-w-0 flex-1 break-words">{props.message}</span>
      {props.onRetry && (
        <button type="button" onClick={props.onRetry} className="dao-btn-secondary shrink-0 text-xs">
          {props.retryLabel}
        </button>
      )}
    </div>
  )
}
