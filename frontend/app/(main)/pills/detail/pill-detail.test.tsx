import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import PillItemDetailPage from '@/app/(main)/pills/detail/pill-detail'
import { ApiError } from '@/services/api'
import type { PillItemDetail } from '@/services/types'

const getPillItem = vi.hoisted(() => vi.fn())
const resolveLegacyPill = vi.hoisted(() => vi.fn())
const consumePill = vi.hoisted(() => vi.fn())
const listAgents = vi.hoisted(() => vi.fn())

vi.mock('@/services/pillInventoryService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/pillInventoryService')>()
  return { ...actual, getPillItem, resolveLegacyPill, consumePill }
})

vi.mock('@/services/agentService', () => ({
  listAgents,
}))

// key 透传：文案断言只关心键/角色，不依赖具体语言
vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

const ITEM_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const RECIPE_ID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
const REVISION_ID = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'

const item: PillItemDetail = {
  id: ITEM_ID,
  name: '文言文丹',
  description: '以文言为骨',
  tags: ['文言'],
  state: 'available',
  recipe_id: RECIPE_ID,
  revision_id: REVISION_ID,
  revision: 3,
  version_label: '2.1.0',
  archived_at: null,
  consumed_at: null,
  created_at: '2026-08-20T00:00:00Z',
}

describe('PillItemDetailPage 金丹库存实例详情', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    getPillItem.mockResolvedValue(item)
    listAgents.mockResolvedValue({ list: [], total: 0 })
  })

  afterEach(() => cleanup())

  it('无 id 展示「链接无效」，不请求 API', () => {
    render(<PillItemDetailPage />)
    expect(screen.getByText('invalidLink')).toBeInTheDocument()
    expect(getPillItem).not.toHaveBeenCalled()
    expect(resolveLegacyPill).not.toHaveBeenCalled()
  })

  it('可用实例展示名称、状态徽标、版本与来源丹方入口', async () => {
    render(<PillItemDetailPage itemId={ITEM_ID} />)
    expect(await screen.findByText('文言文丹')).toBeInTheDocument()
    expect(screen.getByText('stateAvailable')).toBeInTheDocument()
    expect(screen.getByText('stateDescAvailable')).toBeInTheDocument()
    // 版本 v{revision} 与丹方版本标签（标签与值同处一个 span，用正则匹配）
    expect(screen.getByText('revisionLabel')).toBeInTheDocument()
    expect(screen.getByText(/^versionLabel/)).toBeInTheDocument()
    // 来源丹方链接指向丹方详情
    const recipeLink = screen.getByRole('link', { name: 'recipeOrigin' })
    expect(recipeLink).toHaveAttribute('href', `/recipes/detail?id=${RECIPE_ID}`)
    // 未消耗不显示消耗时间
    expect(screen.queryByText('consumedAtLabel')).not.toBeInTheDocument()
  })

  it('已服用实例展示去向与消耗时间', async () => {
    getPillItem.mockResolvedValue({
      ...item,
      state: 'consumed_by_agent',
      consumed_at: '2026-08-21T10:00:00Z',
    })
    render(<PillItemDetailPage itemId={ITEM_ID} />)
    expect(await screen.findByText('stateConsumedByAgent')).toBeInTheDocument()
    expect(screen.getByText('stateDescConsumedByAgent')).toBeInTheDocument()
    expect(screen.getByText(/^consumedAtLabel/)).toBeInTheDocument()
  })

  it('已融合实例展示「已融合」去向', async () => {
    getPillItem.mockResolvedValue({
      ...item,
      state: 'consumed_by_fusion',
      consumed_at: '2026-08-21T10:00:00Z',
    })
    render(<PillItemDetailPage itemId={ITEM_ID} />)
    expect(await screen.findByText('stateConsumedByFusion')).toBeInTheDocument()
    expect(screen.getByText('stateDescConsumedByFusion')).toBeInTheDocument()
  })

  it('已弃置实例展示「已弃置」去向', async () => {
    getPillItem.mockResolvedValue({ ...item, state: 'discarded', consumed_at: '2026-08-21T10:00:00Z' })
    render(<PillItemDetailPage itemId={ITEM_ID} />)
    expect(await screen.findByText('stateDiscarded')).toBeInTheDocument()
    expect(screen.getByText('stateDescDiscarded')).toBeInTheDocument()
  })

  it('来源丹方已归档时展示归档徽标', async () => {
    getPillItem.mockResolvedValue({ ...item, archived_at: '2026-08-25T00:00:00Z' })
    render(<PillItemDetailPage itemId={ITEM_ID} />)
    expect(await screen.findByText('archivedBadge')).toBeInTheDocument()
  })

  it('旧金丹 ID：实例 404 时走显式 legacy 解析，命中后展示「已升级为丹方」并跳转丹方', async () => {
    getPillItem.mockRejectedValue(new ApiError('pill not found', 404))
    resolveLegacyPill.mockResolvedValue({ entity_type: 'recipe', recipe_id: RECIPE_ID })

    render(<PillItemDetailPage itemId={ITEM_ID} />)

    expect(await screen.findByText('legacyTitle')).toBeInTheDocument()
    expect(screen.getByText('legacyDesc')).toBeInTheDocument()
    expect(resolveLegacyPill).toHaveBeenCalledWith(ITEM_ID)
    const goLink = screen.getByRole('link', { name: 'viewRecipeCta' })
    expect(goLink).toHaveAttribute('href', `/recipes/detail?id=${RECIPE_ID}`)
  })

  it('实例 404 且 legacy 解析 404：才展示「不存在」', async () => {
    getPillItem.mockRejectedValue(new ApiError('pill not found', 404))
    resolveLegacyPill.mockRejectedValue(new ApiError('no mapping', 404))

    render(<PillItemDetailPage itemId={ITEM_ID} />)

    expect(await screen.findByText('notFound')).toBeInTheDocument()
    expect(screen.queryByText('legacyTitle')).not.toBeInTheDocument()
  })

  it('网络错误展示错误态，重试重新请求实例', async () => {
    getPillItem.mockRejectedValueOnce(new ApiError('网络请求失败', 0))

    render(<PillItemDetailPage itemId={ITEM_ID} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('网络请求失败')

    await userEvent.click(screen.getByText('retry'))
    expect(await screen.findByText('文言文丹')).toBeInTheDocument()
    expect(getPillItem).toHaveBeenCalledTimes(2)
  })

  it('可用实例展示「服用」按钮，点击打开服用对话框（消耗提示）', async () => {
    const user = userEvent.setup()
    listAgents.mockResolvedValue({ list: [{ id: '11111111-1111-4111-8111-111111111111', name: '太上老君' }], total: 1 })
    render(<PillItemDetailPage itemId={ITEM_ID} />)
    await screen.findByText('文言文丹')

    expect(screen.getByRole('button', { name: 'consumeCta' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'consumeCta' }))

    // 服用对话框打开（key-透传 mock：t('title')='title'、t('prompt')='prompt'）
    expect(await screen.findByRole('heading', { name: 'title' })).toBeInTheDocument()
    expect(screen.getByText('prompt')).toBeInTheDocument()
    await screen.findByText('太上老君')
  })

  it('服用成功后才重读实例：状态变为已服用、按钮消失', async () => {
    const user = userEvent.setup()
    consumePill.mockResolvedValue({ operation_id: 'op-1', effect_id: 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee' })
    listAgents.mockResolvedValue({ list: [{ id: '11111111-1111-4111-8111-111111111111', name: '太上老君' }], total: 1 })
    getPillItem.mockResolvedValueOnce(item)
    getPillItem.mockResolvedValueOnce({
      ...item,
      state: 'consumed_by_agent',
      consumed_at: '2026-08-21T10:00:00Z',
    })
    render(<PillItemDetailPage itemId={ITEM_ID} />)
    await screen.findByText('文言文丹')

    await user.click(screen.getByRole('button', { name: 'consumeCta' }))
    await screen.findByText('太上老君')
    await user.click(screen.getByRole('button', { name: /太上老君/ }))
    await user.click(screen.getByRole('button', { name: 'submit' }))

    // 成功后重读：状态徽标切换为已服用，服用按钮消失
    expect(await screen.findByText('stateConsumedByAgent')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'consumeCta' })).toBeNull()
    expect(getPillItem).toHaveBeenCalledTimes(2)
  })
})
