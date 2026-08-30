'use client'

import Link from 'next/link'
import { useTranslations } from 'next-intl'
import type { ChatSession } from '@/services/types'
import { chatSessionHref } from '@/lib/chat-route'
import { formatDateTime } from '@/utils/format'
import { SealDot } from '@/components/alchemy/float-card'

/**
 * 丹房「论道旧录」：只渲染最近四条会话。
 * - 单聊徽标 = `session.agent_name || unknownAgent`，绝不读取 `session.agent`
 * - 群聊徽标 = `groupIdentity`（成员数来自 `members.length`）
 * - UUID 只用于 React key 和 href，不得进入任何用户可见文本。
 */
export function RecentSessionList({ sessions }: { sessions: ChatSession[] }) {
  const t = useTranslations('home.sessions')
  const recent = sessions.slice(0, 4)

  return (
    <ol className="relative flex flex-col gap-6 pl-4">
      <span
        aria-hidden
        className="absolute bottom-2 left-[3px] top-2 w-px bg-gradient-to-b from-gold/50 via-border to-transparent"
      />
      {recent.length === 0 ? (
        <li className="py-2 text-sm text-muted-foreground">
          {t('empty')}
        </li>
      ) : (
        recent.map((s) => (
          <li key={s.id} className="relative">
            <span className="absolute -left-4 top-1.5" aria-hidden>
              <SealDot />
            </span>
            <Link href={chatSessionHref(s.id)} className="group block">
              <div className="flex items-center gap-2">
                <p className="font-serif text-sm font-bold text-foreground transition-colors group-hover:text-primary">
                  {s.title || t('fallbackTitle')}
                </p>
                <span className="rounded-full bg-secondary px-2 py-0.5 text-[11px] text-sage">
                  {s.type === 'group'
                    ? t('groupIdentity', { count: s.members?.length ?? 0 })
                    : (s.agent_name || t('unknownAgent'))}
                </span>
              </div>
              <p className="mt-1 text-sm leading-relaxed text-muted-foreground">
                {formatDateTime(s.updated_at || s.created_at)}
              </p>
            </Link>
          </li>
        ))
      )}
    </ol>
  )
}
