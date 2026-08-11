'use client'

/**
 * 演示模式横幅(007-demo-mode)
 *
 * 客户端组件:挂载时探测 /api/v1/system/health 的 mode 字段,为 'demo' 时
 * 顶部显示可折叠提示条。生产 output:'export' 下无法在构建期得知运行模式,
 * 故延迟到浏览器运行期探测;非演示模式或已收起时不渲染任何内容。
 */

import { useEffect, useState } from 'react'
import { useTranslations } from 'next-intl'
import { FlaskConical, X } from 'lucide-react'
import { getMode } from '@/lib/demo-mode'
import { cn } from '@/lib/utils'

export function DemoBanner() {
  const t = useTranslations('demoBanner')
  const [isDemo, setIsDemo] = useState(false)
  const [dismissed, setDismissed] = useState(false)

  useEffect(() => {
    getMode().then((m) => setIsDemo(m === 'demo'))
  }, [])

  if (!isDemo || dismissed) return null

  return (
    <div
      className={cn(
        'flex items-center justify-center gap-2 px-4 py-1.5 text-sm',
        'bg-amber-50 text-amber-800 border-b border-amber-200',
        'dark:bg-amber-950/40 dark:text-amber-200 dark:border-amber-800',
      )}
    >
      <FlaskConical className="h-3.5 w-3.5 shrink-0" />
      <span className="truncate">{t('text')}</span>
      <button
        type="button"
        onClick={() => setDismissed(true)}
        aria-label={t('dismiss')}
        className="ml-1 shrink-0 rounded p-0.5 hover:bg-amber-200/60 dark:hover:bg-amber-800/50"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}
