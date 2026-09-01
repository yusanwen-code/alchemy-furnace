import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ConsumeInventoryPillModal } from '@/components/agents/consume-inventory-pill-modal'
import { getPendingOperation } from '@/lib/pending-operations'
import type { AgentEffect, PillItemListItem, PillOperationResult } from '@/services/types'

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

vi.mock('next-intl', async () => {
  const zh = (await import('@/messages/zh-CN.json')).default
  return {
    useTranslations: (namespace: string) => (key: string, values?: Record<string, unknown>) =>
      resolveMsg(zh, namespace, key, values),
    useLocale: () => 'zh-CN',
  }
})

const td = vi.hoisted(() => ({
  listPillItems: vi.fn(),
  consumePill: vi.fn(),
  getOperation: vi.fn(),
}))

vi.mock('@/services/pillInventoryService', () => ({
  listPillItems: td.listPillItems,
  consumePill: td.consumePill,
  getOperation: td.getOperation,
}))

const AGENT_ID = '11111111-1111-4111-8111-111111111111'
const ITEM_A_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const ITEM_B_ID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'

const itemA: PillItemListItem = {
  id: ITEM_A_ID,
  name: '丹心妙语',
  state: 'available',
  recipe_id: 'r-recipe-a',
  revision_id: 'rev-a',
  revision: 1,
  created_at: '2026-08-20T00:00:00Z',
}
const itemB: PillItemListItem = { ...itemA, id: ITEM_B_ID, name: '天机演算', recipe_id: 'r-recipe-b', revision_id: 'rev-b' }

/** 已吸收能力快照（服用/同步都以 revision_id 判重） */
const absorbedEffect: AgentEffect = {
  id: 'effect-1',
  name: '丹心妙语',
  schema: {},
  weight: 1,
  sort_order: 0,
  item_id: ITEM_A_ID,
  revision_id: 'rev-a',
  created_at: '2026-08-20T00:00:00Z',
}

/** 契约准确的成功响应：operation_id 回显幂等 key，含 effect_id 与 consumed_item_ids */
function okResult(key: string, itemId: string = ITEM_A_ID): PillOperationResult {
  return { operation_id: key, effect_id: 'e-1', consumed_item_ids: [itemId] }
}

/** 网络层错误:getOperation 404（recoverOperation 判定为「未提交」） */
function notFoundError(): Error & { status: number } {
  return Object.assign(new Error('not found'), { status: 404 })
}

/** 从 consumePill 调用参数里取出幂等 key（UUID） */
function keyOf(): string {
  const args = td.consumePill.mock.calls[td.consumePill.mock.calls.length - 1]
  return args[0] as string
}

function renderModal(over: Partial<Parameters<typeof ConsumeInventoryPillModal>[0]> = {}) {
  const props = {
    agentId: AGENT_ID,
    agentName: '太上老君',
    activeEffects: [],
    onClose: vi.fn(),
    onCommitted: vi.fn(),
    ...over,
  }
  const utils = render(<ConsumeInventoryPillModal {...props} />)
  return { ...utils, props }
}

describe('ConsumeInventoryPillModal 服用金丹·库存选择弹窗', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    td.listPillItems.mockResolvedValue({ total: 1, items: [itemA] })
    td.consumePill.mockRejectedValue(notFoundError())
    td.getOperation.mockRejectedValue(notFoundError())
  })
  afterEach(() => cleanup())

  it('确认传 {agentId,itemId,weight:1}（无 sortOrder）；按实例 ID 选择而非 recipe/revision', async () => {
    const user = userEvent.setup()
    const onCommitted = vi.fn()
    td.consumePill.mockImplementation(async (key: string, _agentId: string, itemId: string) => okResult(key, itemId))
    td.listPillItems.mockResolvedValue({ total: 2, items: [itemA, itemB] })
    const { props } = renderModal({ onCommitted })
    // 归档丹方产出的现存可用库存只看 state：itemB 带 recipe_id 仍可选
    await screen.findByRole('radio', { name: /天机演算/ })

    await user.click(screen.getByRole('radio', { name: /天机演算/ }))
    await user.click(screen.getByRole('button', { name: /确认服用 1 枚/ }))

    await waitFor(() => expect(onCommitted).toHaveBeenCalledTimes(1))
    expect(td.consumePill).toHaveBeenCalledTimes(1)
    const [key, agentId, itemId, opts] = td.consumePill.mock.calls[0]
    expect(key).toMatch(/^[0-9a-f-]{36}$/)
    expect(agentId).toBe(AGENT_ID)
    expect(itemId).toBe(ITEM_B_ID) // itemId 是库存实例 UUID，不是 recipe/revision/effect
    expect(opts).toEqual({ weight: 1 })
    expect(props.onClose).toHaveBeenCalledTimes(1)
    expect(getPendingOperation(key)).toBeNull()
  })

  it('双击确认只产生一次 POST', async () => {
    const user = userEvent.setup()
    let resolve!: (r: PillOperationResult) => void
    td.consumePill.mockImplementation(
      (key: string) =>
        new Promise((r) => {
          resolve = () => r(okResult(key))
        }),
    )
    const { props } = renderModal()
    await screen.findByRole('radio', { name: /丹心妙语/ })

    await user.click(screen.getByRole('radio', { name: /丹心妙语/ }))
    const confirm = screen.getByRole('button', { name: /确认服用 1 枚/ })
    await user.click(confirm)
    await user.click(confirm)
    expect(td.consumePill).toHaveBeenCalledTimes(1)

    resolve(okResult(keyOf()))
    await waitFor(() => expect(props.onClose).toHaveBeenCalledTimes(1))
  })

  it('activeEffects 的 revision_id 命中:选项禁用并标注「已吸收此版本」；未命中版本仍可服用', async () => {
    const user = userEvent.setup()
    const onCommitted = vi.fn()
    td.consumePill.mockImplementation(async (key: string, _agentId: string, itemId: string) => okResult(key, itemId))
    td.listPillItems.mockResolvedValue({ total: 2, items: [itemA, itemB] })
    renderModal({ activeEffects: [absorbedEffect], onCommitted })

    const absorbed = await screen.findByRole('radio', { name: /丹心妙语/ })
    expect(absorbed).toBeDisabled()
    expect(screen.getByText('已吸收此版本')).toBeInTheDocument()

    await user.click(screen.getByRole('radio', { name: /天机演算/ }))
    await user.click(screen.getByRole('button', { name: /确认服用 1 枚/ }))
    await waitFor(() => expect(onCommitted).toHaveBeenCalledTimes(1))
    const itemId = td.consumePill.mock.calls[0][2]
    expect(itemId).toBe(ITEM_B_ID)
  })

  it('空库存与加载错误分开展示；加载失败可重试', async () => {
    const user = userEvent.setup()
    td.listPillItems.mockResolvedValueOnce({ total: 0, items: [] })
    renderModal()

    expect(await screen.findByText('暂无可用金丹，请先到金丹阁按丹方炼制')).toBeInTheDocument()
    expect(screen.queryByText('库存加载失败')).toBeNull()
    cleanup()
    vi.clearAllMocks()
    sessionStorage.clear()
    td.listPillItems.mockRejectedValueOnce(new Error('boom'))
    renderModal()
    expect(await screen.findByText('库存加载失败')).toBeInTheDocument()

    td.listPillItems.mockResolvedValueOnce({ total: 1, items: [itemA] })
    await user.click(screen.getByRole('button', { name: /重试/ }))
    await waitFor(() => expect(screen.getByRole('radio', { name: /丹心妙语/ })).toBeInTheDocument())
  })

  it('load-more 按实例 ID 去重（跨页重复不重复渲染）', async () => {
    const user = userEvent.setup()
    const page1 = Array.from({ length: 24 }, (_, i) => ({ ...itemA, id: `item-${i}`, name: `金丹 ${i + 1}`, revision_id: `rev-${i}` }))
    // 第 2 页含 1 个与第 1 页重复的实例 + 23 个新实例
    const page2 = [
      { ...page1[0] },
      ...Array.from({ length: 23 }, (_, i) => ({ ...itemA, id: `item-b-${i}`, name: `金丹 B${i + 1}`, revision_id: `rev-b-${i}` })),
    ]
    td.listPillItems
      .mockResolvedValueOnce({ total: 48, items: page1 })
      .mockResolvedValueOnce({ total: 48, items: page2 })
    renderModal()
    await screen.findByRole('radio', { name: '金丹 1' })

    await user.click(screen.getByRole('button', { name: /加载更多/ }))
    await waitFor(() => expect(td.listPillItems).toHaveBeenCalledTimes(2))
    const radios = await screen.findAllByRole('radio')
    expect(radios).toHaveLength(47) // 24 + 24 - 1 重复
    expect(td.listPillItems.mock.calls[1][0]).toEqual({ page: 2, size: 24 })
  })

  it('后续页加载失败:保留已加载项与选择,展示失败并允许重试', async () => {
    const user = userEvent.setup()
    const page1 = Array.from({ length: 24 }, (_, i) => ({ ...itemA, id: `item-${i}`, name: `金丹 ${i + 1}`, revision_id: `rev-${i}` }))
    td.listPillItems
      .mockResolvedValueOnce({ total: 48, items: page1 })
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValueOnce({ total: 48, items: Array.from({ length: 24 }, (_, i) => ({ ...itemA, id: `item-2-${i}`, name: `金丹 2-${i + 1}`, revision_id: `rev-2-${i}` })) })
    renderModal()
    await screen.findByRole('radio', { name: '金丹 1' })
    await user.click(screen.getByRole('radio', { name: '金丹 3' }))

    await user.click(screen.getByRole('button', { name: /加载更多/ }))
    expect(await screen.findByText('后续库存加载失败')).toBeInTheDocument()
    // 已加载项与选择保留
    expect(screen.getByRole('radio', { name: '金丹 3' })).toHaveAttribute('aria-checked', 'true')

    await user.click(screen.getByRole('button', { name: /重试/ }))
    await waitFor(() => expect(screen.queryByText('后续库存加载失败')).not.toBeInTheDocument())
    const radios = await screen.findAllByRole('radio')
    expect(radios).toHaveLength(48)
  })

  it('整页都没有可用实例时自动翻到下一页', async () => {
    const user = userEvent.setup()
    const page1 = Array.from({ length: 24 }, (_, i) => ({ ...itemA, id: `item-${i}`, name: `金丹 ${i + 1}`, revision_id: `rev-${i}` }))
    // 第 2 页全部是已消费/已弃置实例（无 available）
    const page2 = Array.from({ length: 24 }, (_, i) => ({ ...itemA, id: `item-2-${i}`, name: `金丹 2-${i + 1}`, revision_id: `rev-2-${i}`, state: 'consumed' as const }))
    const page3 = Array.from({ length: 2 }, (_, i) => ({ ...itemA, id: `item-c-${i}`, name: `金丹 C${i + 1}`, revision_id: `rev-c-${i}` }))
    td.listPillItems
      .mockResolvedValueOnce({ total: 50, items: page1 })
      .mockResolvedValueOnce({ total: 50, items: page2 })
      .mockResolvedValueOnce({ total: 50, items: page3 })
    renderModal()
    await screen.findByRole('radio', { name: '金丹 1' })

    await user.click(screen.getByRole('button', { name: /加载更多/ }))
    await waitFor(() => expect(td.listPillItems).toHaveBeenCalledTimes(3))
    const radios = await screen.findAllByRole('radio')
    expect(radios).toHaveLength(26)
  })

  it('提交中禁止关闭（X 与 Escape 均不触发 onClose）', async () => {
    const user = userEvent.setup()
    let resolve!: (r: PillOperationResult) => void
    td.consumePill.mockImplementation(
      (key: string) =>
        new Promise((r) => {
          resolve = () => r(okResult(key))
        }),
    )
    const onClose = vi.fn()
    renderModal({ onClose })
    await screen.findByRole('radio', { name: /丹心妙语/ })

    await user.click(screen.getByRole('radio', { name: /丹心妙语/ }))
    await user.click(screen.getByRole('button', { name: /确认服用 1 枚/ }))
    expect(screen.getByText('服用中')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /关闭/ }))
    expect(onClose).not.toHaveBeenCalled()
    await user.keyboard('{Escape}')
    expect(onClose).not.toHaveBeenCalled()

    resolve(okResult(keyOf()))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })

  it('结果未知(uncertain):「稍后核对」关闭且保留 pending 记录', async () => {
    const user = userEvent.setup()
    td.consumePill.mockRejectedValue(new Error('network down'))
    td.getOperation.mockRejectedValue(new Error('offline'))
    const onClose = vi.fn()
    renderModal({ onClose })
    await screen.findByRole('radio', { name: /丹心妙语/ })

    await user.click(screen.getByRole('radio', { name: /丹心妙语/ }))
    await user.click(screen.getByRole('button', { name: /确认服用 1 枚/ }))

    expect(await screen.findByText('服用结果尚未确认，请先核对，勿重复操作')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /稍后核对/ }))
    expect(onClose).toHaveBeenCalledTimes(1)
    expect(getPendingOperation(keyOf())).not.toBeNull()
  })

  it('结果未知时同 key 重试可收敛为 committed（不换 key）', async () => {
    const user = userEvent.setup()
    const onCommitted = vi.fn()
    td.consumePill.mockRejectedValueOnce(new Error('network down')).mockImplementation(async (key: string) => okResult(key))
    td.getOperation.mockRejectedValue(new Error('offline'))
    renderModal({ onCommitted })
    await screen.findByRole('radio', { name: /丹心妙语/ })

    await user.click(screen.getByRole('radio', { name: /丹心妙语/ }))
    await user.click(screen.getByRole('button', { name: /确认服用 1 枚/ }))
    await screen.findByText('服用结果尚未确认，请先核对，勿重复操作')
    const firstKey = keyOf()

    await user.click(screen.getByRole('button', { name: /重试/ }))
    await waitFor(() => expect(onCommitted).toHaveBeenCalledTimes(1))
    expect(td.consumePill).toHaveBeenCalledTimes(2)
    expect(keyOf()).toBe(firstKey) // 同 key，服务端幂等去重
    expect(getPendingOperation(firstKey)).toBeNull()
  })

  it('能力同步失败:只允许「重试同步」(仅再次 GET 能力,绝不再次 POST consume)', async () => {
    const user = userEvent.setup()
    const onCommitted = vi
      .fn()
      .mockRejectedValueOnce(new Error('sync boom'))
      .mockResolvedValueOnce(undefined)
    td.consumePill.mockImplementation(async (key: string) => okResult(key))
    const onClose = vi.fn()
    renderModal({ onCommitted, onClose })
    await screen.findByRole('radio', { name: /丹心妙语/ })

    await user.click(screen.getByRole('radio', { name: /丹心妙语/ }))
    await user.click(screen.getByRole('button', { name: /确认服用 1 枚/ }))

    expect(await screen.findByText('金丹已服用，能力列表同步失败')).toBeInTheDocument()
    expect(td.consumePill).toHaveBeenCalledTimes(1)
    expect(onClose).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: /重试同步/ }))
    await waitFor(() => expect(onCommitted).toHaveBeenCalledTimes(2))
    expect(td.consumePill).toHaveBeenCalledTimes(1) // 同步重试绝不 POST
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('a11y: dialog 语义 + 单选语义 + 焦点进入弹窗 + 方向键移动选择', async () => {
    const user = userEvent.setup()
    td.listPillItems.mockResolvedValue({ total: 2, items: [itemA, itemB] })
    renderModal()

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-labelledby', 'consume-inventory-title')
    expect(dialog).toHaveFocus() // 焦点进入弹窗容器
    const group = screen.getByRole('radiogroup')
    expect(group).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /丹心妙语/ })).toHaveAttribute('aria-checked', 'false')

    await user.keyboard('{ArrowDown}')
    expect(screen.getByRole('radio', { name: /丹心妙语/ })).toHaveAttribute('aria-checked', 'true')
    await user.keyboard('{ArrowDown}')
    expect(screen.getByRole('radio', { name: /天机演算/ })).toHaveAttribute('aria-checked', 'true')
  })

  it('Escape 关闭（非提交中）', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderModal({ onClose })
    await screen.findByRole('radio', { name: /丹心妙语/ })

    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('道人 ID 变化:关闭弹窗丢弃旧响应', async () => {
    const onClose = vi.fn()
    const { rerender } = render(
      <ConsumeInventoryPillModal
        agentId={AGENT_ID}
        agentName="太上老君"
        activeEffects={[]}
        onClose={onClose}
        onCommitted={vi.fn()}
      />,
    )
    await screen.findByRole('radio', { name: /丹心妙语/ })
    rerender(
      <ConsumeInventoryPillModal
        agentId="22222222-2222-4222-8222-222222222222"
        agentName="沉睡道人"
        activeEffects={[]}
        onClose={onClose}
        onCommitted={vi.fn()}
      />,
    )
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
