'use client'

import Image from 'next/image'
import { Flame } from 'lucide-react'
import { SealDot } from '@/components/alchemy/float-card'

const PROGRESS = 68

export function FurnaceCard() {
  return (
    <div className="relative flex h-full flex-col overflow-hidden rounded-[20px] border border-border/70 bg-card/60 shadow-[0_25px_50px_-12px_rgba(60,40,20,0.08)]">
      {/* a cropped shard of the vessel, dissolving into the paper on the right */}
      <div
        aria-hidden
        className="pointer-events-none absolute -right-10 top-0 h-full w-1/2 opacity-70"
      >
        <div className="ink-fade h-full w-full">
          <Image
            src="/ding.png"
            alt=""
            width={1024}
            height={1024}
            className="h-full w-full object-cover object-left mix-blend-multiply"
          />
        </div>
      </div>

      <div className="relative flex h-full flex-col gap-8 p-8 md:p-10">
        <div className="flex items-center justify-between">
          <span className="flex items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
            <Flame className="size-3.5" strokeWidth={2} aria-hidden />
            文火慢炼
          </span>
          <span className="font-mono text-xs text-muted-foreground">第 七 转</span>
        </div>

        <div className="max-w-md">
          <p className="text-sm tracking-widest text-sage">当前炼制</p>
          <h2 className="mt-2 text-balance font-serif text-5xl font-black leading-[0.95] text-foreground md:text-6xl">
            九转
            <br />
            还魂丹
          </h2>
          <p className="mt-5 text-pretty leading-relaxed text-muted-foreground">
            取天山雪莲、千年灵芝与朱砂三味，以三昧真火温养。丹成之日，灵气自生，可续将断之魂魄。
          </p>
        </div>

        {/* progress stated boldly, as a number — not a spinning ring toy */}
        <div className="mt-auto">
          <div className="flex items-end justify-between gap-4">
            <div className="flex items-baseline gap-2">
              <span className="font-serif text-7xl font-black leading-none tracking-tight text-primary">
                {PROGRESS}
              </span>
              <span className="font-serif text-2xl font-bold text-primary/70">%</span>
              <span className="ml-1 pb-1 text-sm text-muted-foreground">丹成</span>
            </div>
            <dl className="hidden gap-8 sm:flex">
              {[
                { k: '炉温', v: '八百', u: '度' },
                { k: '灵火', v: '真火', u: '' },
                { k: '预计', v: '两', u: '时辰' },
              ].map((it) => (
                <div key={it.k}>
                  <dt className="flex items-center gap-1.5 text-xs text-sage">
                    <SealDot />
                    {it.k}
                  </dt>
                  <dd className="mt-1 font-serif text-lg font-bold text-foreground">
                    {it.v}
                    {it.u ? (
                      <span className="ml-0.5 text-sm text-muted-foreground">{it.u}</span>
                    ) : null}
                  </dd>
                </div>
              ))}
            </dl>
          </div>

          {/* a single ink-wash meter */}
          <div className="mt-5 h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-gradient-to-r from-gold to-primary"
              style={{ width: `${PROGRESS}%` }}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
