'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

/** 旧 HashRouter 链接 → App Router 映射 */
const HASH_MAP: Record<string, string> = {
  '#/': '/',
  '#/pills': '/pills',
  '#/agents': '/agents',
  '#/chat': '/chat',
  '#/settings': '/settings',
}

/**
 * 旧 hash 链接兼容：进入站点时若带 #/xxx 形式 hash，重定向到对应新路由
 * 动态段（#/pills/123）按前缀匹配
 */
export function HashRedirect() {
  const router = useRouter()

  useEffect(() => {
    const hash = window.location.hash
    if (!hash.startsWith('#/')) return

    const path = hash.slice(1) // '/pills/123'
    if (path === '/' || HASH_MAP[`#${path}`]) {
      router.replace(HASH_MAP[`#${path}`] ?? '/')
      return
    }
    // 前缀匹配动态段：/pills/:id、/agents/:id、/chat/:sessionId
    for (const prefix of ['/pills/', '/agents/', '/chat/']) {
      if (path.startsWith(prefix)) {
        router.replace(path)
        return
      }
    }
  }, [router])

  return null
}
