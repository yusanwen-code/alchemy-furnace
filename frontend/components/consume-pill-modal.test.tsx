import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ConsumePillModal } from '@/components/consume-pill-modal'
import { getPendingOperation } from '@/lib/pending-operations'
import type { Agent } from '@/services/types'

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
  listAgents: vi.fn(),
  consumePill: vi.fn(),
  getOperation: vi.fn(),
}))

vi.mock('@/services/agentService', () => ({
  listAgents: td.listAgents,
}))

vi.mock('@/services/pillInventoryService', () => ({
  consumePill: td.consumePill,
  getOperation: td.getOperation,
}))

const AGENT_1_ID = '11111111-1111-4111-8111-111111111111'
const AGENT_2_ID = '22222222-2222-4222-8222-222222222222'
const ITEM_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'

const agentWithAvatar: Agent = {
  id: AGENT_1_ID,
  name: '太上老君',
  avatar: 'https://example.com/laojun.png',
  personality: '沉稳如山',
  model_name: 'gpt-4o',
  status: 'active',
  proactivity: 60,
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
}

const agentNoAvatar: Agent = { ...agentWithAvatar, id: AGENT_2_ID, name: '沉睡道人', avatar: undefined }

/** 从 consumePill 调用参数里取出幂等 key（UUID） */
function keyOf(): string {
  const args = td.consumePill.mock.calls[td.consumePill.mock.calls.length - 1]
  return args[0] as string
}

/** 网络层错误:getOperation 404（recoverOperation 判定为「未提交」） */
function notFoundError(): Error & { status: number } {
  return Object.assign(new Error('not found'), { status: 404 })
}

describe('ConsumePillModal 服用对话框', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    td.listAgents.mockResolvedValue({ list: [agentWithAvatar, agentNoAvatar], total: 2 })
    td.consumePill.mockResolvedValue({ operation_id: 'op-1' })
    td.getOperation.mockRejectedValue(notFoundError())
  })
  afterEach(() => cleanup())

  it('展示服用消耗提示与道人列表；未选道人时提交禁用', async () => {
    render(<ConsumePillModal itemId={ITEM_ID} itemName="丹心妙语" onClose={vi.fn()} />)

    expect(await screen.findByText('服用金丹')).toBeInTheDocument()
    expect(
      screen.getByText('服用后将消耗 1 枚金丹，道人保留能力；需要再次使用可按丹方炼制。'),
    ).toBeInTheDocument()
    await screen.findByText('太上老君')
    expect(screen.getByRole('button', { name: /服用/ })).toBeDisabled()
  })

  it('选择道人后提交：一次 consumePill（幂等 key + itemId），成功后回调并清 pending 记录', async () => {
    const user = userEvent.setup()
    const onConsumed = vi.fn()
    // 契约准确的成功响应：operation_id 回显幂等 key，含 effect_id 与 consumed_item_ids
    td.consumePill.mockImplementation(async (key: string) => ({
      operation_id: key,
      effect_id: 'e-1',
      consumed_item_ids: [ITEM_ID],
    }))
    render(<ConsumePillModal itemId={ITEM_ID} itemName="丹心妙语" onClose={vi.fn()} onConsumed={onConsumed} />)
    await screen.findByText('太上老君')

    await user.click(screen.getByRole('button', { name: /太上老君/ }))
    await user.click(screen.getByRole('button', { name: /服用/ }))

    await waitFor(() => expect(onConsumed).toHaveBeenCalledTimes(1))
    expect(td.consumePill).toHaveBeenCalledTimes(1)
    const [key, agentId, itemId, opts] = td.consumePill.mock.calls[0]
    expect(key).toMatch(/^[0-9a-f-]{36}$/)
    expect(agentId).toBe(AGENT_1_ID)
    expect(itemId).toBe(ITEM_ID)
    expect(opts).toEqual({ weight: 1, sortOrder: 0 })
    // 成功后 pending 记录清除
    expect(getPendingOperation(key)).toBeNull()
  })

  it('重复点击提交只产生一项逻辑操作', async () => {
    const user = userEvent.setup()
    let resolve!: () => void
    td.consumePill.mockImplementation(
      (key: string) =>
        new Promise((r) => {
          resolve = () => r({ operation_id: key, effect_id: 'e-1', consumed_item_ids: [ITEM_ID] })
        }),
    )
    render(<ConsumePillModal itemId={ITEM_ID} itemName="丹心妙语" onClose={vi.fn()} onConsumed={vi.fn()} />)
    await screen.findByText('太上老君')

    await user.click(screen.getByRole('button', { name: /太上老君/ }))
    const submit = screen.getByRole('button', { name: /服用/ })
    await user.click(submit)
    await user.click(submit)
    expect(td.consumePill).toHaveBeenCalledTimes(1)

    resolve()
    await waitFor(() => expect(screen.getByRole('button', { name: /已服用/ })).toBeInTheDocument())
  })

  it('失败保留库存：对话框不关闭、错误展示、重试沿用同一幂等 key', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    td.consumePill.mockRejectedValueOnce(new Error('boom')).mockResolvedValueOnce({ operation_id: 'op-1' })
    render(<ConsumePillModal itemId={ITEM_ID} itemName="丹心妙语" onClose={onClose} onConsumed={vi.fn()} />)
    await screen.findByText('太上老君')

    await user.click(screen.getByRole('button', { name: /太上老君/ }))
    await user.click(screen.getByRole('button', { name: /服用/ }))

    // 断线恢复先查 operation：404 → 未提交 → 展示错误、对话框保留
    expect(await screen.findByText('boom')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
    const firstKey = keyOf()
    expect(getPendingOperation(firstKey)).not.toBeNull()

    // 重试:同 key（不能换 key）
    await user.click(screen.getByRole('button', { name: /服用/ }))
    await waitFor(() => expect(td.consumePill).toHaveBeenCalledTimes(2))
    expect(keyOf()).toBe(firstKey)
  })

  it('不同道人服用同一金丹产生不同幂等 key', async () => {
    const user = userEvent.setup()
    td.consumePill.mockRejectedValue(new Error('boom'))
    render(<ConsumePillModal itemId={ITEM_ID} itemName="丹心妙语" onClose={vi.fn()} onConsumed={vi.fn()} />)
    await screen.findByText('太上老君')

    await user.click(screen.getByRole('button', { name: /太上老君/ }))
    await user.click(screen.getByRole('button', { name: /服用/ }))
    await screen.findByText('boom')
    const firstKey = keyOf()

    await user.click(screen.getByRole('button', { name: /沉睡道人/ }))
    await user.click(screen.getByRole('button', { name: /服用/ }))
    await waitFor(() => expect(td.consumePill).toHaveBeenCalledTimes(2))
    expect(keyOf()).not.toBe(firstKey)
  })

  it('断线恢复命中已提交结果时按成功处理（不重复服用）', async () => {
    const user = userEvent.setup()
    const onConsumed = vi.fn()
    td.consumePill.mockRejectedValue(new Error('network down'))
    // 契约准确的恢复响应：operation_id 回显幂等 key
    td.getOperation.mockImplementation(async (key: string) => ({
      operation_id: key,
      effect_id: 'e-1',
      consumed_item_ids: [ITEM_ID],
    }))
    render(<ConsumePillModal itemId={ITEM_ID} itemName="丹心妙语" onClose={vi.fn()} onConsumed={onConsumed} />)
    await screen.findByText('太上老君')

    await user.click(screen.getByRole('button', { name: /太上老君/ }))
    await user.click(screen.getByRole('button', { name: /服用/ }))

    await waitFor(() => expect(onConsumed).toHaveBeenCalledTimes(1))
    expect(screen.queryByText('network down')).toBeNull()
  })
})
