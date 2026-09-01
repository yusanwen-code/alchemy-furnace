'use client'

/**
 * 金丹阁统一外框与页头（2026-08-31 金丹阁布局统一）
 * 丹方 /recipes、金丹库存 /pills、融合炉 /fusion 及两类详情页共用同一内容外框,
 * 避免三页各自实现 max-width 与留白导致的“有的居中有的拉长”观感。
 * 仅统一外部网格、标题与对齐线;不强迫三种业务使用同一种内容组件。
 */
import type { ReactNode } from 'react'

export const PILL_WORKSPACE_FRAME =
  'mx-auto w-full min-w-0 max-w-6xl px-4 sm:px-6'

export function PillWorkspacePage({ children }: { children: ReactNode }) {
  return (
    <div data-pill-workspace-page className={`${PILL_WORKSPACE_FRAME} pt-8 pb-24`}>
      {children}
    </div>
  )
}

export function PillWorkspaceHeader({
  icon,
  title,
  subtitle,
  actions,
}: {
  icon: ReactNode
  title: string
  subtitle: string
  actions?: ReactNode
}) {
  return (
    <header className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex min-w-0 items-start gap-3">
        <span aria-hidden className="mt-1 shrink-0">
          {icon}
        </span>
        <div className="min-w-0">
          <h1 className="page-title break-words">{title}</h1>
          <p className="page-subtitle break-words">{subtitle}</p>
        </div>
      </div>
      {actions && <div className="flex shrink-0 flex-wrap gap-2">{actions}</div>}
    </header>
  )
}
