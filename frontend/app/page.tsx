import Image from 'next/image'
import { FurnaceCard } from '@/components/alchemy/furnace-card'
import { ElixirRecipes } from '@/components/alchemy/elixir-recipes'
import { AlchemyLog } from '@/components/alchemy/alchemy-log'
import { SectionDots } from '@/components/interaction/section-dots'

const sections = [
  { id: 'hero', label: '主殿' },
  { id: 'stats', label: '炉房概览' },
  { id: 'furnace', label: '当前炼制' },
  { id: 'log', label: '炼丹日志' },
]

const stats = [
  { label: '灵气值', value: '3,280', unit: '', trend: '+12%', caption: '天地灵气汇聚一炉' },
  { label: '丹药产量', value: '186', unit: '枚', trend: '+8%', caption: '本月已成之丹' },
  { label: '炼制成功率', value: '84', unit: '%', trend: '+3%', caption: '十炉八九得上品' },
  { label: '药材库存', value: '52', unit: '味', trend: '充盈', caption: '珍稀灵材储于药阁' },
]

export default function Page() {
  return (
    <div className="min-h-screen pb-24">
      <SectionDots sections={sections} />
      <main className="mx-auto max-w-6xl px-4 sm:px-6">
        {/* ── editorial hero: text weighs left, the vessel bleeds off the right ── */}
        <header id="hero" className="relative isolate min-h-[78vh] scroll-mt-24 pt-20 md:pt-28">
          {/* the ding — a heavy object cropped by the frame, mysterious */}
          <div
            aria-hidden
            className="pointer-events-none absolute -right-6 top-4 -z-10 w-[68%] max-w-[760px] opacity-90 md:-right-16 md:top-0 md:w-[62%]"
          >
            <div
              className="float-slow"
              style={{
                WebkitMaskImage:
                  'radial-gradient(130% 120% at 68% 42%, #000 58%, transparent 90%)',
                maskImage:
                  'radial-gradient(130% 120% at 68% 42%, #000 58%, transparent 90%)',
              }}
            >
              <Image
                src="/ding.png"
                alt=""
                width={1024}
                height={1024}
                priority
                className="h-auto w-full mix-blend-multiply"
              />
            </div>
          </div>

          <div className="max-w-2xl">
            <div className="flex items-center gap-3 text-sage">
              <span className="h-px w-8 bg-gold" aria-hidden />
              <span className="size-1.5 rounded-full bg-primary" aria-hidden />
              <span className="text-sm tracking-[0.3em]">丹房 · 今日宜炼丹</span>
            </div>

            <h1 className="mt-8 font-serif text-[24vw] font-black leading-[0.82] tracking-tight text-foreground sm:text-[16rem] md:text-[13rem] lg:text-[15rem]">
              炼丹
              <span className="text-primary">炉</span>
            </h1>

            <p className="mt-10 max-w-md text-pretty text-lg leading-relaxed text-muted-foreground">
              以火为引，以药为基。观灵气流转、察丹药自成——
              <span className="text-foreground">一方静室，万物悬浮，唯心与炉同温。</span>
            </p>

            <div className="mt-10 flex flex-wrap items-center gap-x-8 gap-y-3 border-t border-border/70 pt-6 font-mono text-xs tracking-wide text-sage">
              <span>第 七 转</span>
              <span className="text-border">/</span>
              <span>文火慢炼</span>
              <span className="text-border">/</span>
              <span>炉温 八百度</span>
              <span className="text-border">/</span>
              <span className="text-primary">灵气充盈</span>
            </div>
          </div>
        </header>

        {/* ── stats as a single quiet ledger, hairline-divided — not four toy cards ── */}
        <section
          id="stats"
          aria-label="炉房概览"
          className="mt-8 grid scroll-mt-24 grid-cols-2 overflow-hidden rounded-[20px] border border-border/70 bg-card/50 backdrop-blur-sm md:grid-cols-4"
        >
          {stats.map((s, i) => (
            <div
              key={s.label}
              className={[
                'group relative p-7 transition-colors duration-500 hover:bg-card',
                i % 2 === 0 ? 'border-r border-border/70' : '',
                i < 2 ? 'border-b border-border/70 md:border-b-0' : '',
                i === 2 ? 'md:border-r' : '',
                i === 1 ? 'md:border-r' : '',
              ].join(' ')}
            >
              <div className="flex items-center justify-between">
                <span className="flex items-center gap-2 text-xs tracking-widest text-sage">
                  <span className="size-1 rounded-full bg-primary/70" aria-hidden />
                  {s.label}
                </span>
                <span className="font-mono text-[11px] text-gold">{s.trend}</span>
              </div>
              <div className="mt-6 flex items-baseline gap-1">
                <span className="font-serif text-5xl font-black leading-none tracking-tight text-foreground">
                  {s.value}
                </span>
                {s.unit ? <span className="text-base text-muted-foreground">{s.unit}</span> : null}
              </div>
              <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{s.caption}</p>
              {/* a thread of cinnabar underlines on hover */}
              <span
                aria-hidden
                className="absolute inset-x-7 bottom-0 h-px origin-left scale-x-0 bg-primary/60 transition-transform duration-500 group-hover:scale-x-100"
              />
            </div>
          ))}
        </section>

        {/* main furnace + recipes */}
        <section
          id="furnace"
          className="mt-6 grid scroll-mt-24 grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)]"
        >
          <FurnaceCard />
          <ElixirRecipes />
        </section>

        {/* log + closing seal */}
        <section
          id="log"
          className="mt-6 grid scroll-mt-24 grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.7fr)]"
        >
          <AlchemyLog />
          <ClosingSeal />
        </section>
      </main>
    </div>
  )
}

function ClosingSeal() {
  return (
    <div className="relative flex h-full flex-col justify-between overflow-hidden rounded-[20px] border border-border/70 bg-card/60 p-10 shadow-[0_25px_50px_-12px_rgba(60,40,20,0.08)]">
      <div
        aria-hidden
        className="pointer-events-none absolute -right-10 -top-10 size-52 rounded-full opacity-60 ink-fade"
        style={{
          background: 'radial-gradient(circle, rgba(201,169,110,0.16), transparent 65%)',
        }}
      />
      <div className="relative flex items-start justify-between gap-6">
        <div>
          <p className="font-serif text-3xl font-black text-foreground">丹成 · 归元</p>
          <p className="mt-4 max-w-sm text-pretty leading-relaxed text-muted-foreground">
            「炉中日月长，鼎内乾坤大。」火候到时，自有灵光一点，归于本源。
          </p>
        </div>
        {/* the seal — used once, like a stamp */}
        <div className="grid size-20 shrink-0 place-items-center rounded-2xl bg-primary text-primary-foreground shadow-[0_20px_40px_-12px_rgba(181,74,63,0.45)]">
          <span className="font-serif text-3xl font-black leading-none">丹</span>
        </div>
      </div>
      <button
        type="button"
        className="relative mt-10 inline-flex w-fit items-center gap-2 rounded-full bg-foreground px-7 py-3 text-sm font-medium text-background transition-transform duration-300 hover:scale-[1.03]"
      >
        开炉取丹
      </button>
    </div>
  )
}
