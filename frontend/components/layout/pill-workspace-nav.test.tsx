import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { PillWorkspaceNav } from '@/components/layout/pill-workspace-nav'
import { PILL_WORKSPACE_FRAME } from '@/components/layout/pill-workspace-layout'

// 真实消息解析(命名空间点路径 + {value} 插值),与既有组件测试同模式
function resolveMsg(
  messages: unknown,
  namespace: string,
  key: string,
  values?: Record<string, unknown>,
): string {
  let node: unknown = messages
  for (const part of `${namespace}.${key}`.split('.')) {
    if (node == null || typeof node !== 'object') {
      node = undefined
      break
    }
    node = (node as Record<string, unknown>)[part]
  }
  let text = typeof node === 'string' ? node : `${namespace}.${key}`
  if (values) for (const [k, v] of Object.entries(values)) text = text.split(`{${k}}`).join(String(v))
  return text
}

const pathname = vi.hoisted(() => ({ value: '/recipes' }))

vi.mock('next/navigation', () => ({
  usePathname: () => pathname.value,
}))

vi.mock('next-intl', async () => {
  const zh = (await import('@/messages/zh-CN.json')).default
  return {
    useTranslations: (namespace: string) => (key: string, values?: Record<string, unknown>) =>
      resolveMsg(zh, namespace, key, values),
    useLocale: () => 'zh-CN',
  }
})

const WORKSPACE_ROUTES = ['/recipes', '/pills', '/fusion', '/recipes/detail', '/pills/detail']

describe('PillWorkspaceNav 分区栏外框统一', () => {
  beforeEach(() => {
    pathname.value = '/recipes'
  })
  afterEach(() => cleanup())

  it('金丹阁各路由均渲染分区栏，外框与页面统一(不再随路径收缩 4xl)', () => {
    for (const route of WORKSPACE_ROUTES) {
      pathname.value = route
      cleanup()
      render(<PillWorkspaceNav />)
      const nav = screen.getByRole('navigation')
      // 与页面内容共用左右边界:同一 PILL_WORKSPACE_FRAME + 顶部间距
      for (const cls of PILL_WORKSPACE_FRAME.split(' ')) {
        expect(nav, `${route} 缺少 ${cls}`).toHaveClass(cls)
      }
      expect(nav, `${route} 缺少 pt-6`).toHaveClass('pt-6')
      // 旧行为:库存/丹方详情曾收缩为 max-w-4xl —— 统一后必须消失
      expect(nav, `${route} 残留 max-w-4xl`).not.toHaveClass('max-w-4xl')
    }
  })

  it('分区三项齐全且当前项正确', () => {
    pathname.value = '/pills'
    render(<PillWorkspaceNav />)
    const nav = screen.getByRole('navigation')
    const links = within(nav).getAllByRole('link')
    expect(links).toHaveLength(3)
    // 当前项 = /pills 对应「金丹库存」,aria-current=page
    const current = links.find(l => l.getAttribute('aria-current') === 'page')
    expect(current).toBeDefined()
    expect(current).toHaveTextContent('金丹库存')
    expect(current).toHaveAttribute('href', '/pills')
  })

  it('详情路由同样高亮所属分区', () => {
    for (const [route, title] of [
      ['/recipes/detail', '丹方'],
      ['/pills/detail', '金丹库存'],
    ] as const) {
      pathname.value = route
      cleanup()
      render(<PillWorkspaceNav />)
      const nav = screen.getByRole('navigation')
      const current = within(nav)
        .getAllByRole('link')
        .find(l => l.getAttribute('aria-current') === 'page')
      expect(current, route).toBeDefined()
      expect(current, route).toHaveTextContent(title)
    }
  })

  it('其他模块不显示金丹阁分区栏', () => {
    for (const route of ['/agents', '/agents/detail/abc', '/chat', '/settings', '/']) {
      pathname.value = route
      cleanup()
      render(<PillWorkspaceNav />)
      expect(screen.queryByRole('navigation'), route).toBeNull()
    }
  })
})
