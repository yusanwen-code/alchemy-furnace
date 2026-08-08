'use client'

import Image from 'next/image'

/**
 * 主殿 hero 的青铜鼎
 * - 自然漂浮：多关键帧漂移（位移 + 微旋转），非机械上下
 * - 悬停生火：鼎下朱砂/金色火舌闪烁，青烟袅袅上升
 * - reduced-motion 时全部静止
 */
export function DingHero() {
  return (
    <div className="group/ding relative cursor-pointer">
      {/* 鼎体 */}
      <div
        className="ding-drift relative"
        style={{
          WebkitMaskImage: 'radial-gradient(130% 120% at 62% 42%, #000 55%, transparent 88%)',
          maskImage: 'radial-gradient(130% 120% at 62% 42%, #000 55%, transparent 88%)',
        }}
      >
        <Image
          src="/ding.png"
          alt="青铜鼎"
          width={1024}
          height={1024}
          priority
          className="h-auto w-full mix-blend-multiply"
        />
      </div>

      {/* ── 火焰（悬停点燃）：尺寸/位移用内联样式，避免 Tailwind 动态类被裁剪 ── */}
      <div
        aria-hidden
        className="pointer-events-none absolute left-1/2 opacity-0 transition-opacity duration-700 group-hover/ding:opacity-100"
        style={{ bottom: '2%', transform: 'translateX(-50%)', width: 260, height: 120 }}
      >
        {/* 炉底光晕 */}
        <div
          className="absolute rounded-full"
          style={{
            bottom: -24,
            left: '50%',
            transform: 'translateX(-50%)',
            width: 260,
            height: 90,
            filter: 'blur(28px)',
            background:
              'radial-gradient(ellipse at center, rgba(181,74,63,0.4), rgba(201,169,110,0.2) 55%, transparent 75%)',
          }}
        />
        {/* 火舌三层 */}
        {[
          { w: 110, h: 92, x: -55, d: '0s', dur: 0.9 },
          { w: 78, h: 108, x: -108, d: '0.15s', dur: 1.1 },
          { w: 78, h: 88, x: 28, d: '0.3s', dur: 0.8 },
        ].map((f, i) => (
          <div
            key={i}
            className="absolute"
            style={{
              bottom: 0,
              left: '50%',
              width: f.w,
              height: f.h,
              marginLeft: f.x,
              borderRadius: '50%',
              filter: 'blur(10px)',
              background:
                'radial-gradient(ellipse 50% 70% at 50% 80%, rgba(201,169,110,0.9), rgba(181,74,63,0.6) 55%, transparent 78%)',
              animation: `flame-flicker ${f.dur}s ease-in-out ${f.d} infinite alternate`,
              transformOrigin: '50% 100%',
            }}
          />
        ))}
      </div>

      {/* ── 青烟（悬停袅袅） ── */}
      <div
        aria-hidden
        className="pointer-events-none absolute opacity-0 transition-opacity delay-200 duration-1000 group-hover/ding:opacity-100"
        style={{ left: '50%', top: '10%', width: 0, height: 0 }}
      >
        {[0, 1, 2, 3, 4].map((i) => (
          <span
            key={i}
            className="smoke-wisp absolute block rounded-full bg-sage/50"
            style={{
              width: 14 + i * 4,
              height: 14 + i * 4,
              left: -30 + i * 14,
              filter: 'blur(6px)',
              animation: `smoke-rise ${3.2 + i * 0.5}s ease-out ${i * 0.7}s infinite`,
            }}
          />
        ))}
      </div>
    </div>
  )
}
