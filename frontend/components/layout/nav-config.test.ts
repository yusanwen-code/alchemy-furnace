import { describe, expect, it } from 'vitest'

import { navItems, isNavItemActive } from '@/components/layout/nav-config'

describe('nav config', () => {
  it('keeps only the approved children', () => {
    const pills = navItems.find(item => item.path === '/pills')!
    const agents = navItems.find(item => item.path === '/agents')!
    expect(pills.children?.map(child => child.path)).toEqual(['/pills', '/fusion'])
    expect(agents.children?.map(child => child.path)).toEqual(['/agents'])
    expect(navItems.some(item => item.path === '/fusion')).toBe(false)
    expect(isNavItemActive(pills, '/fusion')).toBe(true)
  })

  it('activates via activePaths without leaking across items', () => {
    const pills = navItems.find(item => item.path === '/pills')!
    const agents = navItems.find(item => item.path === '/agents')!
    const chat = navItems.find(item => item.path === '/chat')!
    expect(isNavItemActive(pills, '/fusion')).toBe(true)
    expect(isNavItemActive(pills, '/pills/123')).toBe(true)
    expect(isNavItemActive(agents, '/agents')).toBe(true)
    expect(isNavItemActive(agents, '/fusion')).toBe(false)
    expect(isNavItemActive(chat, '/pills')).toBe(false)
    expect(isNavItemActive(agents, '/agents/detail/abc')).toBe(true)
  })
})
