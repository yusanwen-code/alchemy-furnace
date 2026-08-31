'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

import { chatLobbyHref, chatSessionHref, parseLegacyChatPath } from '@/lib/chat-route'
import {
  agentDetailHref,
  parseLegacyEntityDetailPath,
  pillItemDetailHref,
} from '@/lib/entity-detail-route'

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
 * 实体动态段（#/pills/<uuid>、#/agents/<uuid>）规范化到查询参数详情地址，
 * 占位/畸形段回列表页
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
    // 旧会话地址 #/chat/<uuid> 规范化到查询参数路由，其余 #/chat/* 回大厅
    const legacySessionId = parseLegacyChatPath(path)
    if (legacySessionId) {
      router.replace(chatSessionHref(legacySessionId))
      return
    }
    if (path.startsWith('/chat/')) {
      router.replace(chatLobbyHref())
      return
    }
    // 历史实体动态段 #/agents/<uuid>、#/pills/<uuid> → 规范查询地址
    const legacyEntity = parseLegacyEntityDetailPath(path)
    if (legacyEntity) {
      router.replace(
        legacyEntity.kind === 'agents'
          ? agentDetailHref(legacyEntity.id)
          : pillItemDetailHref(legacyEntity.id),
      )
      return
    }
    // 占位/畸形实体段回列表页（静态导出无动态路由可直达）
    if (path.startsWith('/agents/')) {
      router.replace('/agents')
      return
    }
    if (path.startsWith('/pills/')) {
      router.replace('/pills')
      return
    }
  }, [router])

  return null
}
