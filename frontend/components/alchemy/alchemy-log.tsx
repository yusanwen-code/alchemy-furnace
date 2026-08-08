'use client'

import { FloatCard, SealDot } from '@/components/alchemy/float-card'

const logs = [
  { time: '子时', text: '投朱砂三钱，炉火转赤，灵气微动。', tag: '投料' },
  { time: '丑时', text: '雪莲入鼎，白雾漫起，温养其性。', tag: '温养' },
  { time: '寅时', text: '真火渐盛，丹胚初凝，色如琥珀。', tag: '凝丹' },
  { time: '卯时', text: '灵芝化液，与诸味相融，香透丹房。', tag: '融合' },
]

export function AlchemyLog() {
  return (
    <FloatCard delay={0.6} className="h-full">
      <div className="flex h-full flex-col gap-6 p-7">
        <div className="flex items-baseline justify-between">
          <h3 className="font-serif text-xl font-black text-foreground">炼丹日志</h3>
          <span className="text-xs text-sage">今日 · 四则</span>
        </div>

        <ol className="relative flex flex-col gap-6 pl-4">
          {/* vertical ink thread */}
          <span
            aria-hidden
            className="absolute bottom-2 left-[3px] top-2 w-px bg-gradient-to-b from-gold/50 via-border to-transparent"
          />
          {logs.map((l) => (
            <li key={l.time} className="relative">
              <span className="absolute -left-4 top-1.5" aria-hidden>
                <SealDot />
              </span>
              <div className="flex items-center gap-2">
                <p className="font-serif text-sm font-bold text-foreground">{l.time}</p>
                <span className="rounded-full bg-secondary px-2 py-0.5 text-[11px] text-sage">
                  {l.tag}
                </span>
              </div>
              <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{l.text}</p>
            </li>
          ))}
        </ol>
      </div>
    </FloatCard>
  )
}
