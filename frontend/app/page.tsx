'use client'

import { useEffect } from 'react'
import Link from 'next/link'
import { ChevronRight, Flame, Plus } from 'lucide-react'
import { usePill } from '@/contexts/PillContext'
import { useAgent } from '@/contexts/AgentContext'
import { useChat } from '@/contexts/ChatContext'
import { DingHero } from '@/components/alchemy/ding-hero'
import { FloatCard, CoinIcon, SealDot } from '@/components/alchemy/float-card'
import { formatDateTime } from '@/utils/format'

export default function Page() {
  const { state: pillState, fetchPills } = usePill()
  const { state: agentState, fetchAgents } = useAgent()
  const { state: chatState, fetchSessions } = useChat()

  useEffect(() => {
    fetchPills({})
    fetchAgents()
    fetchSessions()
  }, [fetchPills, fetchAgents, fetchSessions])

  const pills = pillState.pills
  const agents = agentState.agents
  const sessions = chatState.sessions

  const stats = [
    { label: '阁中藏丹', value: pills.length, unit: '枚', caption: '金丹阁所藏语言模式' },
    { label: '府中道人', value: agents.length, unit: '位', caption: '道人府在册 Agent' },
    { label: '论道场次', value: sessions.length, unit: '场', caption: '已结与未结之缘' },
    { label: '内置丹方', value: pills.filter((p) => p.is_builtin).length, unit: '味', caption: '系统预置的金丹' },
  ]

  const spotlight = pills[0]
  const recentPills = pills.slice(0, 4)
  const recentSessions = sessions.slice(0, 4)

  return (
    <div className="pb-24">
      {/* ── hero：通栏铺满，标题压着鼎交叠 ── */}
      <header id="hero" className="relative isolate flex min-h-[calc(100vh-4rem)] flex-col justify-center overflow-hidden px-5 pt-16 sm:px-8 md:pt-0 lg:px-14">
        {/* 鼎：更大更靠左，让标题尾部压在其上 */}
        <div className="absolute -right-[12%] top-1/2 w-[92%] -translate-y-1/2 opacity-90 sm:-right-[8%] md:w-[72%] lg:-right-[4%] lg:w-[64%]">
          <DingHero />
        </div>

        {/* 竖排古朴题字 */}
        <div
          aria-hidden
          className="absolute right-6 top-1/2 hidden -translate-y-1/2 select-none lg:block"
          style={{ writingMode: 'vertical-rl' }}
        >
          <span className="font-serif text-xl font-bold tracking-[0.6em] text-sage/70">
            炉中日月长 · 鼎内乾坤大
          </span>
        </div>

        <div className="relative z-10 max-w-3xl">
          <div className="flex items-center gap-3 text-sage">
            <span className="h-px w-8 bg-gold" aria-hidden />
            <span className="size-1.5 rounded-full bg-primary" aria-hidden />
            <span className="text-sm tracking-[0.3em]">丹房 · 今日宜炼丹</span>
          </div>

          <h1 className="mt-8 whitespace-nowrap font-serif text-[24vw] font-black leading-[0.82] tracking-tight text-foreground sm:text-[16rem] md:text-[13rem] lg:text-[17rem]">
            炼丹
            <span className="text-primary">炉</span>
          </h1>

          <p className="mt-10 max-w-md text-pretty text-lg leading-relaxed text-muted-foreground">
            以火为引，以药为基。观灵气流转、察丹药自成——
            <span className="text-foreground">一方静室，万物悬浮，唯心与炉同温。</span>
          </p>

          <div className="mt-10 flex flex-wrap items-center gap-x-8 gap-y-3 border-t border-border/70 pt-6 font-mono text-xs tracking-wide text-sage">
            <span>藏丹 {pills.length} 枚</span>
            <span className="text-border">/</span>
            <span>道人 {agents.length} 位</span>
            <span className="text-border">/</span>
            <span>论道 {sessions.length} 场</span>
            <span className="text-border">/</span>
            <span className="text-primary">炉火正温</span>
          </div>
        </div>
      </header>

      <main className="px-5 sm:px-8 lg:px-14">
        {/* ── 炉房概览：真实数据账簿 ── */}
        <section
          id="stats"
          aria-label="炉房概览"
          className="mt-8 grid grid-cols-2 overflow-hidden rounded-[20px] border border-border/70 bg-card/50 backdrop-blur-sm md:grid-cols-4"
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
              <span className="flex items-center gap-2 text-xs tracking-widest text-sage">
                <span className="size-1 rounded-full bg-primary/70" aria-hidden />
                {s.label}
              </span>
              <div className="mt-6 flex items-baseline gap-1">
                <span className="font-serif text-5xl font-black leading-none tracking-tight text-foreground">
                  {s.value}
                </span>
                <span className="text-base text-muted-foreground">{s.unit}</span>
              </div>
              <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{s.caption}</p>
              <span
                aria-hidden
                className="absolute inset-x-7 bottom-0 h-px origin-left scale-x-0 bg-primary/60 transition-transform duration-500 group-hover:scale-x-100"
              />
            </div>
          ))}
        </section>

        {/* ── 金丹阁：新成之丹 ── */}
        <section id="pills" className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)]">
          {/* 最新金丹 spotlight */}
          <div className="relative flex h-full flex-col overflow-hidden rounded-[20px] border border-border/70 bg-card/60 shadow-[0_25px_50px_-12px_rgba(60,40,20,0.08)]">
            <div className="relative flex h-full flex-col gap-8 p-8 md:p-10">
              <div className="flex items-center justify-between">
                <span className="flex items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
                  <Flame className="size-3.5" strokeWidth={2} aria-hidden />
                  新成之丹
                </span>
                <Link href="/pills" className="font-mono text-xs text-muted-foreground transition-colors hover:text-primary">
                  入阁观丹 →
                </Link>
              </div>

              {spotlight ? (
                <div className="max-w-md">
                  <p className="text-sm tracking-widest text-sage">
                    {spotlight.is_builtin ? '内置丹方' : '自行炼制'} · {formatDateTime(spotlight.created_at)}
                  </p>
                  <h2 className="mt-2 text-balance font-serif text-5xl font-black leading-[0.95] text-foreground md:text-6xl">
                    {spotlight.name}
                  </h2>
                  <p className="mt-5 line-clamp-3 text-pretty leading-relaxed text-muted-foreground">
                    {spotlight.description || '此丹未留丹解，入阁可观其详。'}
                  </p>
                  {spotlight.tags.length > 0 && (
                    <div className="mt-5 flex flex-wrap gap-2">
                      {spotlight.tags.map((t) => (
                        <span key={t} className="rounded-full bg-secondary px-2.5 py-0.5 text-xs text-sage">
                          {t}
                        </span>
                      ))}
                    </div>
                  )}
                  <Link
                    href={`/pills/${spotlight.id}`}
                    className="mt-8 inline-flex w-fit items-center gap-2 rounded-full bg-foreground px-7 py-3 text-sm font-medium text-background transition-transform duration-300 hover:scale-[1.03]"
                  >
                    观其丹性
                  </Link>
                </div>
              ) : (
                <div className="flex flex-1 flex-col items-start justify-center gap-4">
                  <p className="font-serif text-3xl font-black text-foreground">阁中尚无一丹</p>
                  <p className="max-w-sm leading-relaxed text-muted-foreground">
                    丹阁空空，炉火却正温。炼第一枚语言模式金丹，塑道人之性情。
                  </p>
                  <Link
                    href="/pills"
                    className="inline-flex items-center gap-2 rounded-full bg-primary px-7 py-3 text-sm font-medium text-primary-foreground transition-transform duration-300 hover:scale-[1.03]"
                  >
                    <Plus className="size-4" aria-hidden />
                    炼制新金丹
                  </Link>
                </div>
              )}
            </div>
          </div>

          {/* 丹方录：最新四枚 */}
          <FloatCard delay={0.2} className="h-full">
            <div className="flex h-full flex-col gap-6 p-7">
              <div className="flex items-baseline justify-between">
                <h3 className="font-serif text-xl font-black text-foreground">丹方录</h3>
                <span className="text-xs text-sage">{pills.length} 味 · 藏丹</span>
              </div>

              <ul className="flex flex-col gap-2">
                {recentPills.map((p, i) => (
                  <li key={p.id}>
                    <Link
                      href={`/pills/${p.id}`}
                      className="group flex w-full items-center gap-4 rounded-2xl px-2 py-2.5 text-left transition-colors duration-300 hover:bg-secondary/70"
                    >
                      <CoinIcon tone={(['gold', 'sage', 'cinnabar'] as const)[i % 3]}>
                        <span className="font-serif text-base font-bold">{p.name.slice(0, 1)}</span>
                      </CoinIcon>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <p className="truncate font-serif text-base font-bold text-foreground">{p.name}</p>
                          {p.is_builtin && (
                            <span className="shrink-0 rounded-full bg-accent px-2 py-0.5 text-[11px] font-medium text-accent-foreground">
                              内置
                            </span>
                          )}
                        </div>
                        <p className="mt-1 truncate text-sm text-muted-foreground">
                          {p.description || '未留丹解'}
                        </p>
                      </div>
                      <ChevronRight
                        className="size-4 shrink-0 text-muted-foreground transition-transform duration-300 group-hover:translate-x-0.5"
                        strokeWidth={1.75}
                        aria-hidden
                      />
                    </Link>
                  </li>
                ))}
                {recentPills.length === 0 && (
                  <li className="py-6 text-center text-sm text-muted-foreground">暂无藏丹</li>
                )}
              </ul>
            </div>
          </FloatCard>
        </section>

        {/* ── 论道旧录 + 收尾 ── */}
        <section id="log" className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.7fr)]">
          <FloatCard delay={0.4} className="h-full">
            <div className="flex h-full flex-col gap-6 p-7">
              <div className="flex items-baseline justify-between">
                <h3 className="font-serif text-xl font-black text-foreground">论道旧录</h3>
                <span className="text-xs text-sage">{sessions.length} 场 · 在录</span>
              </div>

              <ol className="relative flex flex-col gap-6 pl-4">
                <span
                  aria-hidden
                  className="absolute bottom-2 left-[3px] top-2 w-px bg-gradient-to-b from-gold/50 via-border to-transparent"
                />
                {recentSessions.map((s) => (
                  <li key={s.id} className="relative">
                    <span className="absolute -left-4 top-1.5" aria-hidden>
                      <SealDot />
                    </span>
                    <Link href={`/chat/${s.id}`} className="group block">
                      <div className="flex items-center gap-2">
                        <p className="font-serif text-sm font-bold text-foreground transition-colors group-hover:text-primary">
                          {s.title || '未命名论道'}
                        </p>
                        <span className="rounded-full bg-secondary px-2 py-0.5 text-[11px] text-sage">
                          {s.agent?.name || `道人 #${s.agent_id}`}
                        </span>
                      </div>
                      <p className="mt-1 text-sm leading-relaxed text-muted-foreground">
                        {formatDateTime(s.updated_at || s.created_at)}
                      </p>
                    </Link>
                  </li>
                ))}
                {recentSessions.length === 0 && (
                  <li className="py-2 text-sm text-muted-foreground">尚无论道旧录</li>
                )}
              </ol>
            </div>
          </FloatCard>

          <div className="relative flex h-full flex-col justify-between overflow-hidden rounded-[20px] border border-border/70 bg-card/60 p-10 shadow-[0_25px_50px_-12px_rgba(60,40,20,0.08)]">
            <div
              aria-hidden
              className="ink-fade pointer-events-none absolute -right-10 -top-10 size-52 rounded-full opacity-60"
              style={{
                background: 'radial-gradient(circle, rgba(201,169,110,0.16), transparent 65%)',
              }}
            />
            <div className="relative flex items-start justify-between gap-6">
              <div>
                <p className="font-serif text-3xl font-black text-foreground">丹成 · 开炉</p>
                <p className="mt-4 max-w-sm text-pretty leading-relaxed text-muted-foreground">
                  「炉中日月长，鼎内乾坤大。」携丹入炉，与道人论道参玄。
                </p>
              </div>
              <div className="grid size-20 shrink-0 place-items-center rounded-2xl bg-primary text-primary-foreground shadow-[0_20px_40px_-12px_rgba(181,74,63,0.45)]">
                <span className="font-serif text-3xl font-black leading-none">丹</span>
              </div>
            </div>
            <Link
              href="/chat"
              className="relative mt-10 inline-flex w-fit items-center gap-2 rounded-full bg-foreground px-7 py-3 text-sm font-medium text-background transition-transform duration-300 hover:scale-[1.03]"
            >
              开炉论道
            </Link>
          </div>
        </section>
      </main>
    </div>
  )
}
