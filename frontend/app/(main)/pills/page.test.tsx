import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import PillsPage from '@/app/(main)/pills/page'
import type { PillItemListItem } from '@/services/types'

// ---- 可变的测试替身 ----
const td = vi.hoisted(() => ({
  listPillItems: vi.fn(),
}))

// 真实消息解析(命名空间点路径 + {value} 插值,与旧 pills page test 一致)
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

vi.mock('@/services/pillInventoryService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/pillInventoryService')>()
  return { ...actual, listPillItems: td.listPillItems }
})

const RECIPE_A = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const RECIPE_B = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
const REVISION_A = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'
const REVISION_B = 'dddddddd-dddd-4ddd-8ddd-dddddddddddd'

function item(idSuffix: string, recipeId: string, name: string, revision: number): PillItemListItem {
  const id = `00000000-0000-4000-8000-${idSuffix.padStart(12, '0')}`
  return {
    id,
    name,
    state: 'available',
    recipe_id: recipeId,
    revision_id: recipeId === RECIPE_A ? REVISION_A : REVISION_B,
    revision,
    created_at: '2026-08-20T00:00:00Z',
  }
}

describe('PillsPage 金丹库存页', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  afterEach(() => cleanup())

  it('首屏分页加载可用库存（page=1, size=24）', async () => {
    td.listPillItems.mockResolvedValue({ total: 2, items: [item('1', RECIPE_A, '文言文丹方', 1)] })

    render(<PillsPage />)
    expect(await screen.findByText('文言文丹方')).toBeInTheDocument()
    expect(td.listPillItems).toHaveBeenCalledWith({ page: 1, size: 24 })
  })

  it('库存按丹方分组显示数量，条目链接到具体实例详情', async () => {
    td.listPillItems.mockResolvedValue({
      total: 3,
      items: [
        item('1', RECIPE_A, '文言文丹方', 3),
        item('2', RECIPE_A, '文言文丹方', 3),
        item('3', RECIPE_B, '俳句丹方', 1),
      ],
    })

    render(<PillsPage />)

    // 两组标题 + 各自数量徽标
    expect(await screen.findByText('文言文丹方')).toBeInTheDocument()
    expect(screen.getAllByText('2 枚可用').length).toBe(1)
    expect(screen.getAllByText('俳句丹方').length).toBe(1)
    expect(screen.getAllByText('1 枚可用').length).toBe(1)

    // 组标题链接到丹方详情；行条目链接到实例详情（?id=<itemId>）
    expect(screen.getByRole('link', { name: '文言文丹方' })).toHaveAttribute(
      'href',
      `/recipes/detail?id=${RECIPE_A}`,
    )
    const itemLinks = screen
      .getAllByRole('link')
      .filter((link) => link.getAttribute('href')?.startsWith('/pills/detail?id='))
    expect(itemLinks.length).toBe(3)
    expect(itemLinks[0]).toHaveAttribute(
      'href',
      '/pills/detail?id=00000000-0000-4000-8000-000000000001',
    )
  })

  it('没有库存时提示去丹方炼制（不再显示可无限服用的内置定义）', async () => {
    td.listPillItems.mockResolvedValue({ total: 0, items: [] })

    render(<PillsPage />)

    expect(await screen.findByText('暂无可用金丹')).toBeInTheDocument()
    const cta = screen.getByRole('link', { name: '去丹方炼制' })
    expect(cta).toHaveAttribute('href', '/recipes')
  })

  it('加载失败展示错误态，重试重新请求', async () => {
    td.listPillItems.mockRejectedValueOnce(new Error('boom')).mockResolvedValueOnce({
      total: 1,
      items: [item('1', RECIPE_A, '文言文丹方', 1)],
    })

    render(<PillsPage />)
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')

    await userEvent.click(screen.getByText('重新加载'))
    expect(await screen.findByText('文言文丹方')).toBeInTheDocument()
    expect(td.listPillItems).toHaveBeenCalledTimes(2)
  })

  it('分页：有更多时展示「加载更多」，追加下一页并继续分组合并', async () => {
    const pageOne = Array.from({ length: 24 }, (_, i) => item(String(i + 1), RECIPE_A, '文言文丹方', 1))
    td.listPillItems
      .mockResolvedValueOnce({ total: 26, items: pageOne })
      .mockResolvedValueOnce({
        total: 26,
        items: [item('25', RECIPE_B, '俳句丹方', 1), item('26', RECIPE_A, '文言文丹方', 1)],
      })

    render(<PillsPage />)
    await screen.findByText('文言文丹方')

    await userEvent.click(screen.getByText('加载更多'))
    await waitFor(() => expect(td.listPillItems).toHaveBeenLastCalledWith({ page: 2, size: 24 }))
    expect(await screen.findByText('俳句丹方')).toBeInTheDocument()
    expect(screen.getAllByText('25 枚可用').length).toBe(1)
    expect(screen.getAllByText('1 枚可用').length).toBe(1)
  })
})
