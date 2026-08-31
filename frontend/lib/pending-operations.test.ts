import { describe, expect, it, beforeEach } from 'vitest'
import {
  clearPendingOperation,
  getPendingOperation,
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
})
