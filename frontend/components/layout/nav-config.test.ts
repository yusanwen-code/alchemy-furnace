import { describe, expect, it } from 'vitest'

import { navItems, isNavItemActive, pillWorkspaceItems } from '@/components/layout/nav-config'

describe('nav config', () => {
  it('keeps recipes, inventory and fusion under one hoverable workspace menu', () => {
    const pills = navItems.find(item => item.labelKey === 'items.pills.label')!
    const agents = navItems.find(item => item.path === '/agents')!
    // 金丹阁与道人府、论道使用相同的悬停二级菜单交互。
    expect(pills.path).toBe('/recipes')
    expect(pills.children?.map(item => item.path)).toEqual(['/recipes', '/pills', '/fusion'])
    expect(pillWorkspaceItems.map(item => item.path)).toEqual(['/recipes', '/pills', '/fusion'])
    expect(agents.children?.map(child => child.path)).toEqual(['/agents'])
    for (const path of ['/recipes', '/recipes/detail', '/pills', '/pills/detail', '/fusion']) {
      expect(navItems.filter(item => isNavItemActive(item, path))).toEqual([pills])
    }
  })

  it('activates via activePaths without leaking across items', () => {
    const pills = navItems.find(item => item.labelKey === 'items.pills.label')!
    const agents = navItems.find(item => item.path === '/agents')!
    const chat = navItems.find(item => item.path === '/chat')!
    expect(isNavItemActive(pills, '/fusion')).toBe(true)
    expect(isNavItemActive(pills, '/pills/123')).toBe(true)
    expect(isNavItemActive(agents, '/agents')).toBe(true)
    expect(isNavItemActive(agents, '/fusion')).toBe(false)
    expect(isNavItemActive(chat, '/pills')).toBe(false)
    expect(isNavItemActive(pills, '/recipes-other')).toBe(false)
    expect(isNavItemActive(agents, '/agents/detail/abc')).toBe(true)
  })
})
