// 任务 8 测试：丹方页迁移摘要条（升级用户展示）
// 覆盖：legacy 报告展示完整文案；fresh/无标记不展示；关闭后 localStorage 持久化；
// 已关闭状态重进不展示。丹方列表/新建/女娲逻辑各自已有测试，此处只测摘要条。
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import RecipesPage from '@/app/(main)/recipes/page'

// ---- 可变的测试替身 ----
const td = vi.hoisted(() => ({
  listRecipes: vi.fn(),
  getMigrationSummary: vi.fn(),
}))

const RecipeCardSpy = vi.hoisted(() => vi.fn())
const NuwaDistillPanelSpy = vi.hoisted(() => vi.fn())

vi.mock('@/components/recipe-card', () => ({
  RecipeCard: (props: Record<string, unknown>) => {
    RecipeCardSpy(props)
    return <div data-testid="recipe-card" />
  },
}))

vi.mock('@/components/nuwa-distill-panel', () => ({
  NuwaDistillPanel: () => {
    NuwaDistillPanelSpy()
    return <div data-testid="nuwa-panel" />
  },
}))

// 真实消息解析(命名空间点路径 + {value} 插值)
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

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('@/services/recipeService', () => ({
  listRecipes: td.listRecipes,
  saveRecipe: vi.fn(),
}))

vi.mock('@/services/pillInventoryService', () => ({
  getMigrationSummary: td.getMigrationSummary,
}))

/** legacy 迁移摘要（5 丹方 / 3 能力 / 1 可用） */
const legacySummary = {
  migrated: true,
  is_fresh_install: false,
  legacy_pills: 5,
  legacy_binds: 3,
  recipes: 5,
  available_items: 1,
  history_items: 4,
  effects: 3,
  backup_path: '/tmp/backups/pill-inventory-v1.db',
  completed_at: '2026-08-31T12:00:00Z',
}

describe('RecipesPage 迁移摘要条', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    td.listRecipes.mockResolvedValue({ total: 0, items: [] })
    td.getMigrationSummary.mockResolvedValue(legacySummary)
    window.localStorage.clear()
  })
  afterEach(() => cleanup())

  it('legacy 升级用户展示完整文案（已保存 X 丹方、保留 Y 能力、可用 Z 枚）', async () => {
    render(<RecipesPage />)
    expect(await screen.findByText('已保存 5 份丹方、保留 3 项已吸收能力、可用金丹 1 枚；')).toBeInTheDocument()
    expect(screen.getByText('已服用金丹不再作为库存展示。')).toBeInTheDocument()
  })

  it('fresh 安装不展示摘要条', async () => {
    td.getMigrationSummary.mockResolvedValue({ ...legacySummary, is_fresh_install: true, recipes: 0 })
    render(<RecipesPage />)
    await waitFor(() => expect(td.getMigrationSummary).toHaveBeenCalled())
    expect(screen.queryByText(/已保存 0 份丹方/)).toBeNull()
  })

  it('无迁移标记（migrated=false）不展示摘要条', async () => {
    td.getMigrationSummary.mockResolvedValue({ ...legacySummary, migrated: false, is_fresh_install: true })
    render(<RecipesPage />)
    await waitFor(() => expect(td.getMigrationSummary).toHaveBeenCalled())
    expect(screen.queryByText(/已保存/)).toBeNull()
  })

  it('关闭后摘要条消失并写入 localStorage；重进不再展示', async () => {
    const user = userEvent.setup()
    const { unmount } = render(<RecipesPage />)
    await screen.findByText('已保存 5 份丹方、保留 3 项已吸收能力、可用金丹 1 枚；')

    await user.click(screen.getByRole('button', { name: '关闭迁移摘要' }))
    expect(screen.queryByText(/已保存 5 份丹方/)).toBeNull()
    expect(window.localStorage.getItem('pill-migration-banner-dismissed')).toBe('1')

    // 重进（同页面已持久化）
    unmount()
    render(<RecipesPage />)
    await waitFor(() => expect(td.getMigrationSummary).toHaveBeenCalled())
    expect(screen.queryByText(/已保存/)).toBeNull()
  })

  it('摘要读取失败静默：不阻塞丹方列表', async () => {
    td.getMigrationSummary.mockRejectedValue(new Error('网络错误'))
    render(<RecipesPage />)
    await waitFor(() => expect(td.getMigrationSummary).toHaveBeenCalled())
    expect(screen.queryByText(/已保存/)).toBeNull()
    // 页面标题照常渲染，列表照常请求，未被摘要失败阻塞
    await waitFor(() => expect(td.listRecipes).toHaveBeenCalled())
    expect(screen.getByRole('heading', { level: 1 }).textContent).toContain('丹方')
  })
})
