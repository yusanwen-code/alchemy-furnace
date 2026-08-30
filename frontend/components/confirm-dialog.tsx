'use client'

/**
 * 应用内确认对话框(替代 window.confirm)
 * Wails 桌面 WKWebView 不实现 JS confirm 面板(WKUIDelegate 未接
 * runJavaScriptConfirmPanel),window.confirm 静默返回 false——
 * 用应用内对话框保证 web/桌面行为一致。文案由调用方按其 namespace 传入。
 */
import { useId } from 'react'

interface ConfirmDialogProps {
  /** 标题(可选) */
  title?: string
  /** 说明文案(可选) */
  description?: string
  /** 确认按钮文案 */
  confirmLabel: string
  /** 取消按钮文案 */
  cancelLabel: string
  /** 危险操作:确认按钮用朱砂红主色 */
  destructive?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  title,
  description,
  confirmLabel,
  cancelLabel,
  destructive = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const titleId = useId()
  const descId = useId()
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm"
      onClick={onCancel}
    >
      <div
        role="alertdialog"
        aria-modal="true"
        aria-labelledby={title ? titleId : undefined}
        aria-describedby={description ? descId : undefined}
        onClick={(e) => e.stopPropagation()}
        className="dao-card w-full max-w-sm p-6 animate-in fade-in duration-200"
      >
        {title && (
          <h2 id={titleId} className="mb-2 font-serif text-base font-bold text-gold">
            {title}
          </h2>
        )}
        {description && (
          <p id={descId} className="mb-5 whitespace-pre-wrap text-sm text-muted-foreground">
            {description}
          </p>
        )}
        <div className="flex gap-2">
          <button
            type="button"
            onClick={onCancel}
            className="flex-1 rounded-full bg-muted px-5 py-2.5 font-medium text-muted-foreground transition-all duration-200 hover:bg-secondary active:scale-95"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className={`flex-1 ${destructive ? 'dao-btn-primary' : 'dao-btn-gold'}`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
