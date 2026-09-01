import { act, cleanup, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useConsumePillOperation } from '@/hooks/use-consume-pill-operation'
import { getPendingOperation } from '@/lib/pending-operations'
import type { ConsumeInput } from '@/lib/pending-operations'

const consumePill = vi.hoisted(() => vi.fn())
const getOperation = vi.hoisted(() => vi.fn())

vi.mock('@/services/pillInventoryService', () => ({
  consumePill: consumePill,
  getOperation: getOperation,
}))

const AGENT_ID = '11111111-1111-4111-8111-111111111111'
const ITEM_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'

const input: ConsumeInput = {
  agentId: AGENT_ID,
  itemId: ITEM_ID,
  weight: 1,
  sortOrder: 0,
  label: '服用金丹→太上老君',
}

/** 与后端一致的合法响应：operation_id 恒等于幂等 key */
const okResult = (key: string) => ({
  operation_id: key,
  effect_id: 'e-1',
  consumed_item_ids: [ITEM_ID],
})

/** 网络层错误:getOperation 404(recoverOperation 判定为「未提交」) */
function notFoundError(): Error & { status: number } {
  return Object.assign(new Error('not found'), { status: 404 })
}

/** 从 consumePill 调用参数里取出幂等 key(UUID) */
function keyOf(): string {
  const args = consumePill.mock.calls[consumePill.mock.calls.length - 1]
  return args[0] as string
}

describe('useConsumePillOperation 共享服用执行器', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessionStorage.clear()
    consumePill.mockImplementation(async (key: string) => okResult(key))
    getOperation.mockRejectedValue(notFoundError())
  })
  afterEach(() => cleanup())

  it('submit 成功：一次 consumePill（幂等 key + 输入），状态 committed，onCommitted 收到输入，pending 清除', async () => {
    const onCommitted = vi.fn()
    const { result } = renderHook(() => useConsumePillOperation({ onCommitted }))

    await act(async () => {
      await result.current.submit(input)
    })
    expect(result.current.status).toBe('committed')
    expect(consumePill).toHaveBeenCalledTimes(1)
    const [key, agentId, itemId, opts] = consumePill.mock.calls[0]
    expect(key).toMatch(/^[0-9a-f-]{36}$/)
    expect(agentId).toBe(AGENT_ID)
    expect(itemId).toBe(ITEM_ID)
    expect(opts).toEqual({ weight: 1, sortOrder: 0 })
    expect(onCommitted).toHaveBeenCalledTimes(1)
    // onCommitted 收到输入与终局响应(operation_id 回显幂等 key)
    expect(onCommitted.mock.calls[0][0]).toEqual(input)
    expect(onCommitted.mock.calls[0][1]).toMatchObject({
      operation_id: expect.any(String),
      effect_id: 'e-1',
      consumed_item_ids: [ITEM_ID],
    })
    expect(getPendingOperation(key)).toBeNull()
  })

  it('双击 submit 只产生一次 POST（ref 去重）', async () => {
    let resolve!: () => void
    consumePill.mockImplementation(
      (key: string) =>
        new Promise((r) => {
          resolve = () => r(okResult(key))
        }),
    )
    const { result } = renderHook(() => useConsumePillOperation())

    await act(async () => {
      const first = result.current.submit(input)
      void result.current.submit(input)
      resolve()
      await first
    })
    expect(consumePill).toHaveBeenCalledTimes(1)
  })

  it('committed 后不再 POST（含换目标的新输入）', async () => {
    const { result } = renderHook(() => useConsumePillOperation())

    await act(async () => {
      await result.current.submit(input)
    })
    await act(async () => {
      await result.current.submit({ ...input, itemId: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb' })
    })
    expect(consumePill).toHaveBeenCalledTimes(1)
    expect(result.current.status).toBe('committed')
  })

  it('失败（未提交）：状态 error，pending 保留，onCommitted 未调用；retry 同 key 成功后 committed', async () => {
    const onCommitted = vi.fn()
    consumePill.mockRejectedValueOnce(new Error('boom')).mockImplementation(async (key: string) => okResult(key))
    const { result } = renderHook(() => useConsumePillOperation({ onCommitted }))

    await act(async () => {
      await result.current.submit(input)
    })
    expect(result.current.status).toBe('error')
    expect(result.current.error).toBe('boom')
    expect(onCommitted).not.toHaveBeenCalled()
    const firstKey = keyOf()
    expect(getPendingOperation(firstKey)).not.toBeNull()

    await act(async () => {
      await result.current.retry()
    })
    expect(result.current.status).toBe('committed')
    expect(consumePill).toHaveBeenCalledTimes(2)
    expect(keyOf()).toBe(firstKey)
    expect(onCommitted).toHaveBeenCalledTimes(1)
  })

  it('失败但 recover 命中已提交结果：按成功处理，不重复 POST', async () => {
    const onCommitted = vi.fn()
    consumePill.mockRejectedValue(new Error('network down'))
    getOperation.mockImplementation(async (key: string) => okResult(key))
    const { result } = renderHook(() => useConsumePillOperation({ onCommitted }))

    await act(async () => {
      await result.current.submit(input)
    })
    expect(result.current.status).toBe('committed')
    expect(consumePill).toHaveBeenCalledTimes(1)
    expect(onCommitted).toHaveBeenCalledTimes(1)
  })

  it('recover 结果不匹配（operation_id ≠ key）：视为未提交，展示错误', async () => {
    consumePill.mockRejectedValue(new Error('network down'))
    getOperation.mockResolvedValue({
      operation_id: 'other-op',
      effect_id: 'e-1',
      consumed_item_ids: [ITEM_ID],
    })
    const { result } = renderHook(() => useConsumePillOperation())

    await act(async () => {
      await result.current.submit(input)
    })
    expect(result.current.status).toBe('error')
    expect(result.current.error).toBe('network down')
  })

  it('recover 查询本身失败：结果未知（uncertain），pending 保留；retry 同 key 收敛为 committed', async () => {
    const onCommitted = vi.fn()
    consumePill.mockRejectedValueOnce(new Error('network down')).mockImplementation(async (key: string) => okResult(key))
    getOperation.mockRejectedValue(new Error('offline'))
    const { result } = renderHook(() => useConsumePillOperation({ onCommitted }))

    await act(async () => {
      await result.current.submit(input)
    })
    expect(result.current.status).toBe('uncertain')
    expect(result.current.error).toBeNull()
    expect(onCommitted).not.toHaveBeenCalled()
    const firstKey = keyOf()
    expect(getPendingOperation(firstKey)).not.toBeNull()

    await act(async () => {
      await result.current.retry()
    })
    expect(result.current.status).toBe('committed')
    expect(consumePill).toHaveBeenCalledTimes(2)
    expect(keyOf()).toBe(firstKey)
    expect(onCommitted).toHaveBeenCalledTimes(1)
  })

  it('isConsumedResult：operation_id===key 且 effect_id 存在且 consumed_item_ids 含 itemId', async () => {
    const { isConsumedResult } = await import('@/hooks/use-consume-pill-operation')
    const key = 'k-1'
    expect(isConsumedResult({ operation_id: key, effect_id: 'e-1', consumed_item_ids: [ITEM_ID] }, key, input)).toBe(true)
    // operation_id 不符
    expect(isConsumedResult({ operation_id: 'other', effect_id: 'e-1', consumed_item_ids: [ITEM_ID] }, key, input)).toBe(false)
    // 无 effect_id
    expect(isConsumedResult({ operation_id: key, consumed_item_ids: [ITEM_ID] }, key, input)).toBe(false)
    // consumed 不含本次 itemId
    expect(isConsumedResult({ operation_id: key, effect_id: 'e-1', consumed_item_ids: ['other-item'] }, key, input)).toBe(false)
  })
})
