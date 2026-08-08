'use client'

import { useEffect, useState } from 'react'
import { cn } from '@/lib/utils'

export interface Section {
  id: string
  label: string
}

/**
 * 黑神话式左侧竖排圆点导航（参考 reference/交互.png）
 * IntersectionObserver 追踪当前区块，点击平滑滚动定位；窄屏隐藏
 */
export function SectionDots({ sections }: { sections: Section[] }) {
  const [activeId, setActiveId] = useState(sections[0]?.id ?? '')

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) setActiveId(entry.target.id)
        }
      },
      // 视口中间一条窄带为激活区
      { rootMargin: '-45% 0px -45% 0px', threshold: 0 },
    )
    sections.forEach(({ id }) => {
      const el = document.getElementById(id)
      if (el) observer.observe(el)
    })
    return () => observer.disconnect()
  }, [sections])

  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return (
    <nav
      aria-label="页面区块导航"
      className="fixed left-6 top-1/2 z-40 hidden -translate-y-1/2 flex-col items-center gap-4 lg:flex"
    >
      {sections.map(({ id, label }) => {
        const active = id === activeId
        return (
          <button
            key={id}
            type="button"
            aria-label={label}
            aria-current={active ? 'true' : undefined}
            onClick={() => scrollTo(id)}
            className="group relative grid size-5 place-items-center"
          >
            <span
              className={cn(
                'rounded-full transition-all duration-300',
                active
                  ? 'size-2.5 bg-primary'
                  : 'size-1.5 bg-sage/40 group-hover:size-2 group-hover:bg-sage',
              )}
            />
            {/* hover 提示 */}
            <span className="pointer-events-none absolute left-7 whitespace-nowrap rounded-full bg-card/95 px-2.5 py-1 font-serif text-xs font-bold text-foreground opacity-0 shadow-sm ring-1 ring-border/70 transition-opacity duration-300 group-hover:opacity-100">
              {label}
            </span>
          </button>
        )
      })}
    </nav>
  )
}
