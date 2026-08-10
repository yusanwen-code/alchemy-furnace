'use client'

import { useEffect } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { ChevronRight, Flame, Plus } from 'lucide-react'
import { usePill } from '@/contexts/PillContext'
import { useAgent } from '@/contexts/AgentContext'
import { useChat } from '@/contexts/ChatContext'
import { FloatCard, CoinIcon, SealDot } from '@/components/alchemy/float-card'
import { DingFlameParticle } from '@/components/alchemy/ding-flame-particle'
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
        {/* 鼎：参考 / 炉子动画 的 float-slow + 径向 mask；hover 点火冒烟；累计缩 20% */}
        {/* 外层 absolute 负责定位；内层 group/ding + relative 是火/烟 absolute 子元素的包含块 */}
        <div className="absolute right-[8%] top-1/2 w-[92%] -translate-y-1/2 opacity-90 sm:right-[8%] md:w-[72%] md:right-[8%] lg:right-[8%] lg:w-[64%]">
          <div className="group/ding relative w-full">
            {/* 鼎体：scale(1.1) 再大 10%（hero 比例已通过 Image 高度约束协调） */}
            <div className="origin-center" style={{ transform: 'scale(1.1)' }}>
              <div
                className="float-slow relative"
                style={{
                  WebkitMaskImage:
                    'radial-gradient(130% 120% at 68% 42%, #000 58%, transparent 90%)',
                  maskImage:
                    'radial-gradient(130% 120% at 68% 42%, #000 58%, transparent 90%)',
                }}
              >
                <div
                  className="relative ml-auto mr-0"
                  style={{
                    height: 'calc(100vh - 5rem)',
                    aspectRatio: '1 / 1',
                    maxWidth: '100%',
                  }}
                >
                <Image
                  src="/ding.png"
                  alt="青铜鼎"
                  width={1024}
                  height={1024}
                  preload
                  className="h-full w-full mix-blend-multiply"
                />

                {/* 火焰（hover 点燃）：3 个窗口各 1 组，坐标由 /tmp/measure_y.py 实测 ding.png 暗区得到 */}
                {/* 容器宽 = 暗区实际宽（每扇窗独立），火苗不溢出铜壁；y 起点 = 暗区顶；h = 暗区高 + 0.2%（约 2px）让 scaleY 起点在暗区下边外侧 */}
                {/* 点燃用 .ding-flame-window 类（CSS @keyframes flame-ignite）：火星一闪 → 跳动生长 → 稳定燃烧，模拟真实火苗 */}
                <div
                  aria-hidden
                  className="pointer-events-none absolute inset-0 opacity-0 transition-opacity duration-500 group-hover/ding:opacity-100"
                >
                  {[
                    // 实测暗区 (measure_y.py): 左 y 50.10-59.18% x 33.30-41.21% (8.01% 宽)
                    { x: 37.26, w: 8.01, top: 50.10, h: 9.38, phase: 0.00 },
                    // 中 y 50.49-59.47% x 46.00-54.59% (8.69% 宽)
                    { x: 50.30, w: 8.69, top: 50.49, h: 9.28, phase: 0.45 },
                    // 右 y 50.20-59.28% x 59.28-67.09% (7.91% 宽)
                    { x: 63.19, w: 7.91, top: 50.20, h: 9.38, phase: 0.90 },
                  ].map((w) => (
                    <div key={w.x} className="contents">
                    {/* 炉口辉光：从暗区底缘向下溢出 13% 到铜壁承沿上, 让暗→铜的硬切变渐变, 减少图层割裂感 */}
                    <div
                      aria-hidden
                      className="ding-flame pointer-events-none absolute"
                      style={{
                        left: `${w.x}%`,
                        top: `${w.top + 4}%`,
                        width: `${w.w * 1.9}%`,
                        height: '14%',
                        transform: 'translateX(-50%)',
                        background:
                          'radial-gradient(ellipse 100% 100% at 50% 0%, rgba(255,160,60,0.85) 0%, rgba(230,100,30,0.6) 20%, rgba(190,55,18,0.35) 45%, rgba(140,40,15,0.15) 70%, transparent 95%)',
                        filter: 'blur(10px)',
                        mixBlendMode: 'screen',
                        animationDelay: `${w.phase}s`,
                      }}
                    />
                    {/* 火焰窗：外层负责定位 + translateX；内层 .ding-flame-window 走 flame-ignite 关键帧动画（火星→跳动→稳定），hover 解除时由 transition 平滑熄灭 */}
                    {/* 容器 h = 暗区高 + 0.2%（≈2px），底边在暗区下边外侧，scaleY(0) 起点正好在暗区下边外 1px，火苗像从炉口下方升起 */}
                    <div
                      className="absolute"
                      style={{
                        left: `${w.x}%`,
                        top: `${w.top}%`,
                        width: `${w.w}%`,
                        height: `${w.h}%`,
                        transform: 'translateX(-50%)',
                      }}
                    >
                      <div
                        className="ding-flame-window h-full w-full overflow-hidden"
                        style={{
                          borderRadius: '55% 55% 8% 8% / 32% 32% 4% 4%',
                        }}
                      >
                      {/* 粒子火苗：fill-glow + tongue + halo 三层合并为一个 canvas 粒子火焰，Hover 时由父级 .ding-flame-window 的 flame-ignite 关键帧整体缩放点燃 */}
                      <DingFlameParticle />
                      </div>
                    </div>
                    </div>
                  ))}
                </div>

                {/* 青烟（hover 袅袅）：3 缕分别从 3 个窗口上方飘升；delay-200 与火苗错开，duration-500 与火苗同步 */}
                <div
                  aria-hidden
                  className="pointer-events-none absolute inset-0 opacity-0 transition-opacity delay-200 duration-500 group-hover/ding:opacity-100"
                >
                  {[
                    { x: 37.55, phase: 0.0 },
                    { x: 50.24, phase: 0.6 },
                    { x: 63.09, phase: 1.2 },
                  ].map((w) => (
                    <div
                      key={w.x}
                      className="absolute"
                      style={{
                        left: `${w.x}%`,
                        top: '34%',
                        width: '8%',
                        height: '18%',
                        transform: 'translateX(-50%)',
                      }}
                    >
                      {[0, 1, 2, 3, 4].map((i) => (
                        <span
                          key={i}
                          className="ding-smoke absolute block rounded-full bg-sage/50"
                          style={{
                            width: 6 + i * 3,
                            height: 6 + i * 3,
                            left: -10 + i * 5,
                            filter: 'blur(6px)',
                            animationDuration: `${3.2 + i * 0.5}s`,
                            animationDelay: `${i * 0.4 + w.phase}s`,
                          }}
                        />
                      ))}
                    </div>
                  ))}
                </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="relative z-10 max-w-3xl">
          <h1 className="font-serif font-black leading-[1.04] tracking-tight text-foreground">
            <span className="block text-[22vw] sm:text-[11rem] md:text-[13rem] lg:text-[15rem]">炼丹</span>
            <span className="block pl-[0.08em] text-[22vw] text-primary sm:text-[11rem] md:text-[13rem] lg:text-[15rem]">炉</span>
          </h1>

          {/* 描述段已删除（原：以火为引...唯心与炉同温） */}

          {/* 数据 banner 已删除（原：藏丹 / 道人 / 论道 / 炉火正温） */}
        </div>

        {/* 底部横排题字 */}
        <p className="relative z-10 mt-14 flex items-center gap-4 font-serif text-sm font-bold tracking-[0.5em] text-sage/80">
          <span className="h-px w-10 bg-gold/70" aria-hidden />
          炉中日月长 · 鼎内乾坤大
        </p>
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
