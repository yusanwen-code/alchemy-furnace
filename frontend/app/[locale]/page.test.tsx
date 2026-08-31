import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import HomePage from '@/app/[locale]/page'
import type { PillItemDetail, PillItemListItem, RecipeListItem } from '@/services/types'

// jsdom 未实现 matchMedia:BaguaFurnace 烟效 media query 依赖它(bagua-furnace.test.tsx 同款桩)
beforeAll(() => {
  window.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia
})

// ---- 可变的测试替身 ----
const td = vi.hoisted(() => ({
  listPillItems: vi.fn(),
  getPillItem: vi.fn(),
  listRecipes: vi.fn(),
  fetchAgents: vi.fn(),
  fetchSessions: vi.fn(),
}))

// 真实消息解析(命名空间点路径 + {value} 插值),与 pills/page.test.tsx 同模式
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

vi.mock('next-intl', async () => {
  const zh = (await import('@/messages/zh-CN.json')).default
  return {
    useTranslations: (namespace: string) => (key: string, values?: Record<string, unknown>) =>
      resolveMsg(zh, namespace, key, values),
    useLocale: () => 'zh-CN',
  }
})

vi.mock('next/image', () => ({
  __esModule: true,
  default: (props: Record<string, unknown>) => (
    <span data-alt={String(props.alt ?? '')} style={props.style as React.CSSProperties} />
  ),
}))

vi.mock('@/services/pillInventoryService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/pillInventoryService')>()
  return { ...actual, listPillItems: td.listPillItems, getPillItem: td.getPillItem }
})

vi.mock('@/services/recipeService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/recipeService')>()
  return { ...actual, listRecipes: td.listRecipes }
})

vi.mock('@/contexts/AgentContext', () => ({
  useAgent: () => ({
    state: { agents: [], total: 0, loading: false, error: null, currentAgent: null },
    fetchAgents: td.fetchAgents,
  }),
}))

vi.mock('@/contexts/ChatContext', () => ({
  useChat: () => ({
    state: {
      sessions: [],
      currentSession: null,
      messages: [],
      loading: false,
      streaming: false,
      error: null,
    },
    fetchSessions: td.fetchSessions,
  }),
}))

const ITEM_A = '11111111-1111-4111-8111-111111111111'
const ITEM_B = '22222222-2222-4222-8222-222222222222'
const RECIPE_A = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const RECIPE_B = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'

function item(id: string, recipeId: string, name: string, revision: number): PillItemListItem {
  return {
    id,
    name,
    state: 'available',
    recipe_id: recipeId,
    revision_id: 'cccccccc-cccc-4ccc-8ccc-cccccccccccc',
    revision,
    created_at: '2026-08-30T08:00:00Z',
  }
}

const itemA = item(ITEM_A, RECIPE_A, '浩然丹', 2)
const itemB = item(ITEM_B, RECIPE_B, '清风方丹', 1)

function recipe(
  id: string,
  name: string,
  revision: number,
  availableCount: number,
): RecipeListItem {
  return {
    id,
    name,
    current_revision_id: 'cccccccc-cccc-4ccc-8ccc-cccccccccccc',
    revision,
    available_count: availableCount,
    created_at: '2026-08-29T08:00:00Z',
  }
}

const recipeA = recipe(RECIPE_A, '浩然方', 2, 3)
const recipeB = recipe(RECIPE_B, '清风方', 1, 1)

const itemADetail: PillItemDetail = {
  ...itemA,
  description: '养浩然正气,立不欺之心。',
  tags: ['内丹', '心法'],
  version_label: '2.0.0',
  created_at: '2026-08-30T08:00:00Z',
}

describe('HomePage（丹方与消耗性金丹）', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    td.listPillItems.mockResolvedValue({ total: 2, items: [itemA, itemB] })
    td.getPillItem.mockResolvedValue(itemADetail)
    td.listRecipes.mockResolvedValue({ total: 2, items: [recipeA, recipeB] })
    td.fetchAgents.mockResolvedValue(undefined)
    td.fetchSessions.mockResolvedValue(undefined)
  })

  afterEach(() => cleanup())

  it('挂载并发拉取库存与丹方;统计卡展示库存实例数与丹方数', async () => {
    render(<HomePage />)

    expect(td.listPillItems).toHaveBeenCalledWith({ size: 4 })
    expect(td.listRecipes).toHaveBeenCalledWith({ size: 4 })
    expect(td.fetchAgents).toHaveBeenCalled()
    expect(td.fetchSessions).toHaveBeenCalled()

    // 库存统计(available 实例数)与丹方统计(永久保留的丹方数)
    const stats = within(await screen.findByLabelText('炉房概览'))
    expect(stats.getByText('库存金丹')).toBeInTheDocument()
    expect(stats.getByText('阁中可用之丹')).toBeInTheDocument()
    expect(stats.getByText('丹方')).toBeInTheDocument()
    expect(stats.getByText('可永久传承之方')).toBeInTheDocument()
    // 两张统计卡各显示 2;道人/会话空列表显示 0
    expect(stats.getAllByText('2')).toHaveLength(2)
    expect(stats.getAllByText('0')).toHaveLength(2)
  })

  it('spotlight 展示最新库存实例:名称/版本/炼制时间 + 详情描述与标签,CTA 指向实例详情', async () => {
    render(<HomePage />)

    await screen.findByRole('heading', { name: '浩然丹' })
    expect(td.getPillItem).toHaveBeenCalledWith(ITEM_A)
    // 版本用整数 revision(计划:丹方入口显示「版本 vN」),非 version_label 字符串;
    // 「版本 v2 · 」限定 spotlight 行,避免与丹方录徽章的「版本 v2」重复匹配
    expect(screen.getByText(/版本 v2 ·/)).toBeInTheDocument()
    expect(screen.getByText('养浩然正气,立不欺之心。')).toBeInTheDocument()
    expect(screen.getByText('内丹')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '观其丹性' })).toHaveAttribute(
      'href',
      `/pills/detail?id=${ITEM_A}`,
    )
  })

  it('丹方录展示丹方:名称/版本徽章/该丹方可用库存数,链接指向丹方详情', async () => {
    render(<HomePage />)

    const recipeLink = await screen.findByRole('link', { name: /浩然方/ })
    expect(recipeLink).toHaveTextContent('版本 v2')
    expect(recipeLink).toHaveTextContent('库存 3 枚')
    expect(recipeLink).toHaveAttribute('href', `/recipes/detail?id=${RECIPE_A}`)
  })

  it('库存为空:spotlight 空态引导去丹方炼制(不是回到旧内置金丹列表)', async () => {
    td.listPillItems.mockResolvedValue({ total: 0, items: [] })
    render(<HomePage />)

    await screen.findByText('阁中尚无一丹')
    const cta = screen.getByRole('link', { name: '前往丹方炼制' })
    expect(cta).toHaveAttribute('href', '/recipes')
    expect(td.getPillItem).not.toHaveBeenCalled()
  })

  it('spotlight 详情拉取失败:用列表字段兜底展示,不崩溃', async () => {
    td.getPillItem.mockRejectedValue(new Error('boom'))
    render(<HomePage />)

    // 名称/版本来自列表项,描述回退到 noDescription
    await screen.findByRole('heading', { name: '浩然丹' })
    expect(screen.getByText(/版本 v2 ·/)).toBeInTheDocument()
    expect(screen.getByText('此丹未留丹解，入阁可观其详。')).toBeInTheDocument()
  })

  it('丹方为空:丹方录显示空提示', async () => {
    td.listRecipes.mockResolvedValue({ total: 0, items: [] })
    render(<HomePage />)

    await screen.findByText('暂无丹方,炉火待温')
  })
})
