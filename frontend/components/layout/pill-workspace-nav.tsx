'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { isNavItemActive, pillWorkspaceItems } from './nav-config'
import { cn } from '@/lib/utils'

/** 保留原有静态路由与详情地址，在金丹阁各页提供唯一且常驻的分区导航。 */
export function PillWorkspaceNav() {
  const pathname = usePathname()
  const t = useTranslations('nav')
  const isActive = (path: string) => isNavItemActive({ path }, pathname)
  if (!pillWorkspaceItems.some(item => isActive(item.path))) return null
  // 库存及两类详情沿用窄版阅读容器，导航需与页面内容共用左右边界。
  const compact = isActive('/pills') || isActive('/recipes/detail')

  return (
    <nav
      aria-label={t('pillWorkspaceLabel')}
      className={cn('mx-auto px-4 pt-6 sm:px-6', compact ? 'max-w-4xl' : 'max-w-6xl')}
    >
      <div className="flex gap-1 overflow-x-auto border-b border-border">
        {pillWorkspaceItems.map(item => {
          const active = isActive(item.path)
          const Icon = item.icon
          return (
            <Link
              key={item.path}
              href={item.path}
              aria-current={active ? 'page' : undefined}
              title={t(item.descKey)}
              className={cn(
                'inline-flex shrink-0 items-center gap-2 border-b-2 px-4 py-3 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-2',
                active ? 'border-primary font-medium text-primary' : 'border-transparent text-muted-foreground hover:text-foreground',
              )}
            >
              <Icon className="size-4" aria-hidden />
              {t(item.titleKey)}
            </Link>
          )
        })}
      </div>
    </nav>
  )
}
