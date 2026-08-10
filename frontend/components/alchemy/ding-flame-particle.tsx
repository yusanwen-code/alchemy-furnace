'use client'

import { useEffect, useMemo, useRef, useState } from 'react'
import { FireFlame } from '@9am/fire-flame-react'
import { FireFlameOption, Vector } from '@9am/fire-flame'

export function DingFlameParticle() {
  const wrapRef = useRef<HTMLDivElement>(null)
  const flameRef = useRef<{ setOption: (o: FireFlameOption) => void }>(null)
  const [size, setSize] = useState({ w: 0, h: 0 })

  useEffect(() => {
    if (!wrapRef.current) return
    const ro = new ResizeObserver((entries) => {
      const e = entries[0]
      if (!e) return
      const { width, height } = e.contentRect
      setSize({ w: Math.round(width), h: Math.round(height) })
    })
    ro.observe(wrapRef.current)
    // Initial measurement: ResizeObserver fires on observe, but if it
    // doesn't (e.g. element has 0 size at attach time), we still need a
    // first frame. Use getBoundingClientRect as a fallback.
    const r = wrapRef.current.getBoundingClientRect()
    if (r.width > 0 && r.height > 0) {
      setSize({ w: Math.round(r.width), h: Math.round(r.height) })
    }
    return () => ro.disconnect()
  }, [])

  const option = useMemo<FireFlameOption>(
    () => ({
      painter: 'canvas',
      // Always provide a non-zero size so the library can initialize its
      // canvas even before the first ResizeObserver tick. Real measurements
      // overwrite this on the first observation.
      w: size.w || 80,
      h: size.h || 40,
      // Particle spawn point: bottom-center of the canvas so wind (-0.4 in y)
      // carries the trail UPWARD into the viewport instead of off the top.
      // (Default x/y = 0,0 puts everything in the top-left and the trail
      // escapes the canvas before being visible.)
      x: (size.w || 80) / 2,
      y: (size.h || 40) * 0.92,
      mousemove: false,
      // Spec called for `{ x: 0, y: -0.4 }` but the library's `updateParticles`
      // calls `wind.add(...)` / `wind.multiply(...)` at runtime, so it must be
      // a Vector instance, not a plain Point.
      wind: new Vector({ x: 0, y: -0.4 }),
      friction: 0.98,
      particleNum: 18,
      particleDistance: 8,
      particleFPS: 12,
      innerColor: '#ffd97a',
      outerColor: '#b8331f',
      fps: 60,
    }),
    [size.w, size.h]
  )

  return (
    <div
      ref={wrapRef}
      style={{ position: 'absolute', inset: 0, overflow: 'hidden' }}
    >
      <FireFlame ref={flameRef} option={option} />
    </div>
  )
}

