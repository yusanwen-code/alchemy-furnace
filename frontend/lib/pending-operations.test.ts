import { describe, expect, it, beforeEach, vi } from 'vitest'
import {
  clearPendingOperation,
  consumeTarget,
  getPendingOperation,
  listPendingOperations,
  startPendingOperation,
  type PendingAction,
} from './pending-operations'

describe('pending-operations 桌面会话级幂等操作记录', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('start 生成 UUID key 并记录动作与目标', () => {
    const key = startPendingOperation('craft', '丹方·深思')
    expect(key).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
    const record = getPendingOperation(key)
    expect(record).not.toBeNull()
    expect(record?.action).toBe('craft')
    expect(record?.label).toBe('丹方·深思')
  })

  it('同动作同目标 pending 期间复用同一 key（重复点击只产生一项逻辑操作）', () => {
    const first = startPendingOperation('consume', '道人·青鸾')
    const second = startPendingOperation('consume', '道人·青鸾')
    expect(second).toBe(first)
  })

  it('不同目标生成不同 key', () => {
    const a = startPendingOperation('craft', '丹方·甲')
    const b = startPendingOperation('craft', '丹方·乙')
    expect(a).not.toBe(b)
  })

  it('clear 后再次 start 生成新 key（完成一次动作后才能发起下一次）', () => {
    const first = startPendingOperation('craft', '丹方·甲')
    clearPendingOperation(first)
    const second = startPendingOperation('craft', '丹方·甲')
    expect(second).not.toBe(first)
  })

  it('未知 key 返回 null（断线恢复 404 后仍用原 key 重试，不自动换 key）', () => {
    expect(getPendingOperation('00000000-0000-4000-8000-000000000000')).toBeNull()
  })

  it('consume 带 target：同 action+target 复用同一 key；换目标即新动作', () => {
    const input = {
      agentId: '11111111-1111-4111-8111-111111111111',
      itemId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
      weight: 1,
    }
    const first = startPendingOperation('consume', '服用金丹→太上老君', consumeTarget(input), input)
    const second = startPendingOperation('consume', '服用金丹→太上老君', consumeTarget(input), input)
    expect(second).toBe(first)
    // 换金丹（不同 itemId）即新动作 → 新 key
    const otherInput = { ...input, itemId: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb' }
    const third = startPendingOperation('consume', '服用金丹→太上老君', consumeTarget(otherInput), otherInput)
    expect(third).not.toBe(first)
  })

  it('consume target = JSON.stringify([agentId, itemId, weight, sortOrder ?? 0])；快照只存 UUID/权重/顺序', () => {
    const input = {
      agentId: '11111111-1111-4111-8111-111111111111',
      itemId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
      weight: 2,
      sortOrder: 3,
      label: '服用金丹→太上老君',
    }
    const key = startPendingOperation('consume', input.label, consumeTarget(input), input)
    const record = getPendingOperation(key)
    expect(record?.target).toBe(
      JSON.stringify(['11111111-1111-4111-8111-111111111111', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', 2, 3]),
    )
    // 快照不得携带名称/密钥/请求体
    expect(record?.consumeInput).toEqual({
      agentId: '11111111-1111-4111-8111-111111111111',
      itemId: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
      weight: 2,
      sortOrder: 3,
    })
    expect(JSON.stringify(record?.consumeInput)).not.toMatch(/太上老君|服用金丹/)
  })

  it('sortOrder 缺省按 0 参与签名（默认与显式 0 同 key）', () => {
    const a = consumeTarget({ agentId: 'x', itemId: 'y', weight: 1 })
    const b = consumeTarget({ agentId: 'x', itemId: 'y', weight: 1, sortOrder: 0 })
    expect(a).toBe(b)
  })

  it('listPendingOperations 返回当前全部记录', () => {
    const k1 = startPendingOperation('craft', '丹方·甲')
    const k2 = startPendingOperation('consume', '服用金丹→太上老君', consumeTarget({ agentId: 'a', itemId: 'b', weight: 1 }), { agentId: 'a', itemId: 'b', weight: 1 })
    expect(listPendingOperations().map(r => r.key)).toEqual(expect.arrayContaining([k1, k2]))
    clearPendingOperation(k1)
    expect(listPendingOperations().map(r => r.key)).not.toContain(k1)
  })

  it('sessionStorage 不可用：模块级内存后备仍保证同 key 去重与清除', () => {
    const getSpy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('denied')
    })
    const setSpy = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('denied')
    })
    try {
      const input = { agentId: 'a', itemId: 'b', weight: 1 }
      const first = startPendingOperation('consume', 'l', consumeTarget(input), input)
      const second = startPendingOperation('consume', 'l', consumeTarget(input), input)
      expect(second).toBe(first)
      expect(getPendingOperation(first)).not.toBeNull()
      clearPendingOperation(first)
      expect(getPendingOperation(first)).toBeNull()
    } finally {
      getSpy.mockRestore()
      setSpy.mockRestore()
    }
  })
})
