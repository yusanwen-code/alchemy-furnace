'use client'

import { ChevronRight } from 'lucide-react'
import { FloatCard, CoinIcon } from '@/components/alchemy/float-card'

const recipes = [
  { name: '养气培元丹', grade: '上品', rate: 92, cn: '气', tone: 'gold' as const },
  { name: '凝神静心散', grade: '中品', rate: 78, cn: '神', tone: 'sage' as const },
  { name: '驻颜朱果丸', grade: '上品', rate: 85, cn: '颜', tone: 'cinnabar' as const },
  { name: '破障通玄丹', grade: '极品', rate: 61, cn: '玄', tone: 'gold' as const },
]

export function ElixirRecipes() {
  return (
    <FloatCard delay={0.2} className="h-full">
      <div className="flex h-full flex-col gap-6 p-7">
        <div className="flex items-baseline justify-between">
          <h3 className="font-serif text-xl font-black text-foreground">丹方录</h3>
          <span className="text-xs text-sage">四味 · 待炼</span>
        </div>

        <ul className="flex flex-col gap-2">
          {recipes.map((r) => (
            <li key={r.name}>
              <button
                type="button"
                className="group flex w-full items-center gap-4 rounded-2xl px-2 py-2.5 text-left transition-colors duration-300 hover:bg-secondary/70"
              >
                <CoinIcon tone={r.tone}>
                  <span className="font-serif text-base font-bold">{r.cn}</span>
                </CoinIcon>

                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <p className="truncate font-serif text-base font-bold text-foreground">
                      {r.name}
                    </p>
                    <span className="shrink-0 rounded-full bg-accent px-2 py-0.5 text-[11px] font-medium text-accent-foreground">
                      {r.grade}
                    </span>
                  </div>
                  {/* thin ink-wash success bar */}
                  <div className="mt-2 h-1 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-gold/80"
                      style={{ width: `${r.rate}%` }}
                    />
                  </div>
                </div>

                <div className="flex shrink-0 items-center gap-1 text-muted-foreground">
                  <span className="font-mono text-xs">{r.rate}%</span>
                  <ChevronRight
                    className="size-4 transition-transform duration-300 group-hover:translate-x-0.5"
                    strokeWidth={1.75}
                    aria-hidden
                  />
                </div>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </FloatCard>
  )
}
