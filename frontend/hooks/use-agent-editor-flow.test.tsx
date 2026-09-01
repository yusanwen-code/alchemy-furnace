import type { ReactNode } from 'react'
import { act, cleanup, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AgentProvider, useAgent } from '@/contexts/AgentContext'
import { useAgentEditorFlow, type AgentEffectsData } from '@/hooks/use-agent-editor-flow'
import { ApiError } from '@/services/api'
import type { AgentDetail, AgentEffect, DistillationDraft } from '@/services/types'

const getAgent = vi.hoisted(() => vi.fn())
const updateAgent = vi.hoisted(() => vi.fn())
const listEffects = vi.hoisted(() => vi.fn())
const removeEffect = vi.hoisted(() => vi.fn())
const updateEffects = vi.hoisted(() => vi.fn())

vi.mock('@/services/agentService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/agentService')>()
  return { ...actual, getAgent, updateAgent }
})

vi.mock('@/services/pillInventoryService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/pillInventoryService')>()
  return { ...actual, listEffects, removeEffect, updateEffects }
})

const wrapper = ({ children }: { children: ReactNode }) => (
  <AgentProvider>{children}</AgentProvider>
)

const baseAgent: AgentDetail = {
  id: 'agent-1',
  name: '太上老君',
  avatar: 'https://example.com/laojun.png',
  personality: '沉稳如山',
  model_name: 'gpt-4o',
  status: 'active',
  proactivity: 60,
  created_at: '2026-08-20T00:00:00Z',
  updated_at: '2026-08-20T00:00:00Z',
}

const effA: AgentEffect = {
  id: 'eff-a',
  name: '丹心妙语',
  schema: {},
  weight: 2,
  sort_order: 1,
  item_id: 'item-a',
  revision_id: 'rev-1',
  created_at: '2026-08-20T00:00:00Z',
}
const effB: AgentEffect = { ...effA, id: 'eff-b', name: '浩然正气', weight: 1, sort_order: 2, item_id: 'item-b' }
const effC: AgentEffect = { ...effA, id: 'eff-c', name: '清风徐来', weight: 3, sort_order: 3, item_id: 'item-c' }

const effectsData: AgentEffectsData = { effects_revision: 2, effects: [effA, effB] }

const nuwaDraft: DistillationDraft = {
  name: '女娲造人',
  description: '蒸馏候选',
  persona_summary: '悲天悯人的造物主',
  tags: ['神话'],
  skill_schema: { identity_card: '造物主' },
  sources: [{ title: '淮南子', url: 'https://example.com/huainanzi', dimension: 'persona' }],
  model: 'gpt-5',
  research: {
    evidence_level: 'standard',
    document_count: 1,
    domain_count: 1,
    total_characters: 2500,
    warnings: [],
  },
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function effectsConflict(): ApiError {
  return new ApiError('能力编排已变更', 409, {
    code: 409,
    error_code: 'service.agent.effects_conflict',
    message: '能力编排已变更,请刷新后重试',
    data: null,
  })
}

describe('useAgentEditorFlow（effects 数据源）', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    sessionStorage.clear()
    listEffects.mockResolvedValue(effectsData)
    updateEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
  })
  afterEach(() => cleanup())

  it('beginEdit 从能力列表构建草稿（key=effect_id，权重=快照），零引用共享', () => {
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    expect(result.current.mode).toBe('readonly')
    expect(result.current.dirty).toBe(false)

    act(() => result.current.beginEdit())
    expect(result.current.mode).toBe('editing')
    expect(result.current.draft.name).toBe('太上老君')
    // 能力行:key 与 effect_id 同值,weight 沿用快照;顺序 = listEffects 返回顺序
    expect(result.current.draft.effects).toEqual([
      { key: 'eff-a', effect_id: 'eff-a', weight: 2 },
      { key: 'eff-b', effect_id: 'eff-b', weight: 1 },
    ])
    // 引用隔离:草稿行不得与源 effects 共享引用
    expect(result.current.draft.effects[0]).not.toBe(effectsData.effects[0])

    act(() => result.current.updateDraft({ name: '改名' }))
    expect(result.current.dirty).toBe(true)
    expect(baseAgent.name).toBe('太上老君')
  })

  it('discard 放弃修改并回到只读', () => {
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '改名' }))
    expect(result.current.dirty).toBe(true)

    act(() => result.current.discard())
    expect(result.current.mode).toBe('readonly')
    expect(result.current.dirty).toBe(false)
    expect(result.current.draft.name).toBe('太上老君')
  })

  it('保存成功：基础资料 → 重读能力列表取最新乐观锁 → 全量编排 → GET 回读', async () => {
    updateAgent.mockResolvedValue({ ...baseAgent, name: '改名' })
    getAgent.mockResolvedValue({ ...baseAgent, name: '改名', updated_at: '2026-08-23T00:00:00Z' })
    const freshEffects: AgentEffectsData = { effects_revision: 3, effects: [effA, effB] }
    listEffects.mockResolvedValue(freshEffects)
    updateEffects.mockResolvedValue({ effects_revision: 3, effects: [effA, effB] })
    const onEffectsRefreshed = vi.fn()
    const { result } = renderHook(
      () => ({ flow: useAgentEditorFlow(baseAgent, effectsData, onEffectsRefreshed), agent: useAgent() }),
      { wrapper },
    )

    act(() => result.current.flow.beginEdit())
    act(() => result.current.flow.updateDraft({ name: '改名', proactivity: 80 }))

    let ok = false
    await act(async () => {
      ok = await result.current.flow.save()
    })
    expect(ok).toBe(true)
    // 第一步:基础资料(与旧行为一致)
    expect(updateAgent).toHaveBeenCalledTimes(1)
    expect(updateAgent).toHaveBeenCalledWith('agent-1', {
      name: '改名',
      avatar: 'https://example.com/laojun.png',
      personality: '沉稳如山',
      model_name: 'gpt-4o',
      proactivity: 80,
      status: 'active',
    })
    // 无缺失能力 → 不调用移除
    expect(removeEffect).not.toHaveBeenCalled()
    // 第二步:重读能力列表(拿最新 effects_revision,remove 后会递增)
    expect(listEffects).toHaveBeenCalledTimes(1)
    expect(listEffects).toHaveBeenCalledWith('agent-1')
    expect(onEffectsRefreshed).toHaveBeenCalledWith(freshEffects)
    // 第三步:全量编排(乐观锁 = 重读后的 revision;sortOrder = 草稿数组下标)
    expect(updateEffects).toHaveBeenCalledTimes(1)
    expect(updateEffects).toHaveBeenCalledWith('agent-1', 3, [
      { effectId: 'eff-a', weight: 2, sortOrder: 0 },
      { effectId: 'eff-b', weight: 1, sortOrder: 1 },
    ])
    // 第四步:写后重读
    expect(getAgent).toHaveBeenCalledTimes(1)
    expect(getAgent).toHaveBeenCalledWith('agent-1')
    // 顺序:基础资料 → 重读 → 编排 → GET
    expect(updateAgent.mock.invocationCallOrder[0]).toBeLessThan(listEffects.mock.invocationCallOrder[0])
    expect(listEffects.mock.invocationCallOrder[0]).toBeLessThan(updateEffects.mock.invocationCallOrder[0])
    expect(updateEffects.mock.invocationCallOrder[0]).toBeLessThan(getAgent.mock.invocationCallOrder[0])
    // Context 中的最终对象是 GET 回读的 fresh
    expect(result.current.agent.state.currentAgent?.updated_at).toBe('2026-08-23T00:00:00Z')
    expect(result.current.flow.mode).toBe('readonly')
    expect(result.current.flow.dirty).toBe(false)
    expect(result.current.flow.saveStatus).toBe('idle')
  })

  it('草稿删除能力：保存时按幂等 key 移除（成功后清 pending），提交集不含该能力', async () => {
    updateAgent.mockResolvedValue({ ...baseAgent })
    getAgent.mockResolvedValue({ ...baseAgent })
    listEffects.mockResolvedValue({ effects_revision: 3, effects: [effB] })
    updateEffects.mockResolvedValue({ effects_revision: 3, effects: [effB] })
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    // 移除第一枚能力(只保留 eff-b)
    act(() => result.current.updateDraft({ effects: [{ key: 'eff-b', effect_id: 'eff-b', weight: 1 }] }))

    let ok = false
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(true)
    expect(removeEffect).toHaveBeenCalledTimes(1)
    const [key, agentId, effectId] = removeEffect.mock.calls[0]
    expect(key).toMatch(UUID_RE)
    expect(agentId).toBe('agent-1')
    expect(effectId).toBe('eff-a')
    // 移除成功后清除 pending 记录
    expect(sessionStorage.getItem('alchemy_pending_operations')).toBe('[]')
    // 提交集 = 剩余能力
    expect(updateEffects).toHaveBeenCalledWith('agent-1', 3, [
      { effectId: 'eff-b', weight: 1, sortOrder: 0 },
    ])
  })

  it('移除时已不存在（404 dao.agent.remove_effect_uuid）：视为已移除继续', async () => {
    updateAgent.mockResolvedValue({ ...baseAgent })
    getAgent.mockResolvedValue({ ...baseAgent })
    listEffects.mockResolvedValue({ effects_revision: 3, effects: [effB] })
    updateEffects.mockResolvedValue({ effects_revision: 3, effects: [effB] })
    removeEffect.mockRejectedValue(
      new ApiError('能力不存在', 404, {
        code: 404,
        error_code: 'dao.agent.remove_effect_uuid',
        message: '能力不存在',
        data: null,
      }),
    )
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ effects: [{ key: 'eff-b', effect_id: 'eff-b', weight: 1 }] }))

    let ok = false
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(true)
    expect(updateEffects).toHaveBeenCalledTimes(1)
    expect(sessionStorage.getItem('alchemy_pending_operations')).toBe('[]')
  })

  it('移除失败（500）：保存失败保留草稿，不进入编排', async () => {
    updateAgent.mockResolvedValue({ ...baseAgent })
    removeEffect.mockRejectedValue(new ApiError('服务器内部错误', 500))
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ effects: [{ key: 'eff-b', effect_id: 'eff-b', weight: 1 }] }))

    let ok = true
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.saveStatus).toBe('error')
    expect(result.current.mode).toBe('editing')
    expect(result.current.draft.effects).toEqual([{ key: 'eff-b', effect_id: 'eff-b', weight: 1 }])
    expect(updateEffects).not.toHaveBeenCalled()
    expect(getAgent).not.toHaveBeenCalled()
    // pending 记录保留:重试沿用同 key
    const records = JSON.parse(sessionStorage.getItem('alchemy_pending_operations') ?? '[]')
    expect(records).toHaveLength(1)
    expect(records[0].action).toBe('remove_effect')
  })

  it('409 冲突：保留草稿与用户编辑，刷新能力基线并合并新能力，saveStatus=conflict', async () => {
    updateAgent.mockResolvedValue({ ...baseAgent })
    // 第三步重读返回旧基线;冲突处理中的刷新返回含新能力 C 的基线
    listEffects
      .mockResolvedValueOnce({ effects_revision: 2, effects: [effA, effB] })
      .mockResolvedValueOnce({ effects_revision: 3, effects: [effA, effB, effC] })
    updateEffects.mockRejectedValue(effectsConflict())
    const onEffectsRefreshed = vi.fn()
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData, onEffectsRefreshed), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '新名' }))

    let ok = true
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.saveStatus).toBe('conflict')
    expect(result.current.mode).toBe('editing')
    // 基础资料编辑保留
    expect(result.current.draft.name).toBe('新名')
    // 仍在集合中的行保留用户权重/顺序,新能力 C 追加末尾(权重沿用快照)
    expect(result.current.draft.effects).toEqual([
      { key: 'eff-a', effect_id: 'eff-a', weight: 2 },
      { key: 'eff-b', effect_id: 'eff-b', weight: 1 },
      { key: 'eff-c', effect_id: 'eff-c', weight: 3 },
    ])
    expect(onEffectsRefreshed).toHaveBeenCalledWith({ effects_revision: 3, effects: [effA, effB, effC] })
  })

  it('409 后重新保存：用刷新后的乐观锁与合并后的提交集,成功退出编辑', async () => {
    updateAgent.mockResolvedValue({ ...baseAgent })
    getAgent.mockResolvedValue({ ...baseAgent })
    listEffects
      .mockResolvedValueOnce({ effects_revision: 2, effects: [effA, effB] })
      .mockResolvedValueOnce({ effects_revision: 3, effects: [effA, effB, effC] })
      .mockResolvedValue({ effects_revision: 3, effects: [effA, effB, effC] })
    updateEffects
      .mockRejectedValueOnce(effectsConflict())
      .mockResolvedValue({ effects_revision: 3, effects: [effA, effB, effC] })
    const onEffectsRefreshed = vi.fn()
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData, onEffectsRefreshed), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '新名' }))

    let ok = false
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.saveStatus).toBe('conflict')

    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(true)
    expect(result.current.mode).toBe('readonly')
    // 第二次保存:乐观锁 = 刷新后的 revision,提交集 = 合并后的能力
    expect(updateEffects).toHaveBeenCalledTimes(2)
    expect(updateEffects.mock.calls[1][0]).toBe('agent-1')
    expect(updateEffects.mock.calls[1][1]).toBe(3)
    expect(updateEffects.mock.calls[1][2]).toEqual([
      { effectId: 'eff-a', weight: 2, sortOrder: 0 },
      { effectId: 'eff-b', weight: 1, sortOrder: 1 },
      { effectId: 'eff-c', weight: 3, sortOrder: 2 },
    ])
    // 冲突处理中刷新过一次 + 第二次保存的重读 = 共 3 次 listEffects
    expect(listEffects).toHaveBeenCalledTimes(3)
  })

  it('基础资料保存失败：不进入后续步骤，保留草稿与编辑态', async () => {
    updateAgent.mockRejectedValue(new ApiError('服务器内部错误', 500))
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '新名字' }))

    let ok = true
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.saveStatus).toBe('error')
    expect(result.current.mode).toBe('editing')
    expect(result.current.draft.name).toBe('新名字')
    expect(result.current.dirty).toBe(true)
    expect(listEffects).not.toHaveBeenCalled()
    expect(removeEffect).not.toHaveBeenCalled()
    expect(updateEffects).not.toHaveBeenCalled()
    expect(getAgent).not.toHaveBeenCalled()
  })

  it('全量编排失败（非冲突）：保留草稿与编辑态，不发起 GET', async () => {
    updateAgent.mockResolvedValue({ ...baseAgent })
    listEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
    updateEffects.mockRejectedValue(new ApiError('服务器内部错误', 500))
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ effects: [{ key: 'eff-a', effect_id: 'eff-a', weight: 5 }] }))

    let ok = true
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.saveStatus).toBe('error')
    expect(result.current.mode).toBe('editing')
    expect(result.current.draft.effects).toEqual([{ key: 'eff-a', effect_id: 'eff-a', weight: 5 }])
    expect(getAgent).not.toHaveBeenCalled()
  })

  it('保存后 GET 失败不得宣称完成：保持编辑态与草稿', async () => {
    updateAgent.mockResolvedValue({ ...baseAgent })
    listEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
    updateEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
    getAgent.mockRejectedValue(new ApiError('服务器内部错误', 500))
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '新名字' }))

    let ok = true
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.saveStatus).toBe('error')
    expect(result.current.mode).toBe('editing')
    expect(result.current.draft.name).toBe('新名字')
    expect(result.current.dirty).toBe(true)
  })

  it('连续点击保存只产生一次写请求', async () => {
    const gate = deferred<AgentDetail>()
    updateAgent.mockImplementation(() => gate.promise)
    listEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
    updateEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
    getAgent.mockResolvedValue({ ...baseAgent })
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '改名' }))

    let first!: Promise<boolean>
    let second!: Promise<boolean>
    act(() => {
      first = result.current.save()
      second = result.current.save()
    })
    expect(updateAgent).toHaveBeenCalledTimes(1)

    gate.resolve(baseAgent)
    await act(async () => {
      await Promise.all([first, second])
    })
    await expect(first).resolves.toBe(true)
    await expect(second).resolves.toBe(false)
  })

  it('name 为空拒绝保存并写入 fieldErrors，不发起任何 API', async () => {
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '   ' }))

    let ok = true
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.fieldErrors.name).toBeTruthy()
    expect(result.current.mode).toBe('editing')
    expect(updateAgent).not.toHaveBeenCalled()
    expect(updateEffects).not.toHaveBeenCalled()
  })

  it('能力剂量越界写入 fieldErrors（effects.<key>.weight），不发起任何 API', async () => {
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    act(() => result.current.beginEdit())
    act(() =>
      result.current.updateDraft({ effects: [{ key: 'eff-a', effect_id: 'eff-a', weight: 11 }] }),
    )

    let ok = true
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.fieldErrors['effects.eff-a.weight']).toBeTruthy()
    expect(updateAgent).not.toHaveBeenCalled()
    expect(updateEffects).not.toHaveBeenCalled()
  })

  it('Nuwa 草稿只在显式 applyNuwaDraft 后写入 name/personality，不触碰能力编排', () => {
    const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
    expect(result.current.draft.name).toBe('太上老君')

    act(() => result.current.applyNuwaDraft(nuwaDraft))
    expect(result.current.mode).toBe('editing')
    expect(result.current.draft.name).toBe('女娲造人')
    expect(result.current.draft.personality).toBe('悲天悯人的造物主')
    expect(result.current.dirty).toBe(true)
    // 不得触碰能力编排
    expect(result.current.draft.effects).toEqual([
      { key: 'eff-a', effect_id: 'eff-a', weight: 2 },
      { key: 'eff-b', effect_id: 'eff-b', weight: 1 },
    ])
  })

  describe('avatar 字段契约校验(与后端 validateAvatar 对齐)', () => {
    beforeEach(() => {
      vi.resetAllMocks()
      sessionStorage.clear()
      listEffects.mockResolvedValue(effectsData)
      updateEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
    })

    it('非法协议(javascript:)头像拒绝保存并写入 fieldErrors.avatar,零 API', async () => {
      const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ avatar: 'javascript:alert(1)' }))

      let ok = true
      await act(async () => {
        ok = await result.current.save()
      })
      expect(ok).toBe(false)
      expect(result.current.fieldErrors.avatar).toBeTruthy()
      expect(result.current.mode).toBe('editing')
      expect(updateAgent).not.toHaveBeenCalled()
      expect(getAgent).not.toHaveBeenCalled()
    })

    it('相对路径头像拒绝保存并写入 fieldErrors.avatar', async () => {
      const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ avatar: '/avatar.png' }))

      let ok = true
      await act(async () => {
        ok = await result.current.save()
      })
      expect(ok).toBe(false)
      expect(result.current.fieldErrors.avatar).toBe('invalid')
      expect(updateAgent).not.toHaveBeenCalled()
      expect(updateEffects).not.toHaveBeenCalled()
    })

    it('空白头像视为清空:通过校验并提交空字符串', async () => {
      updateAgent.mockResolvedValue({ ...baseAgent })
      listEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
      updateEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
      getAgent.mockResolvedValue({ ...baseAgent })
      const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ avatar: '   ' }))

      let ok = false
      await act(async () => {
        ok = await result.current.save()
      })
      expect(ok).toBe(true)
      expect(updateAgent).toHaveBeenCalledWith('agent-1', expect.objectContaining({ avatar: '' }))
      expect(result.current.mode).toBe('readonly')
    })

    it('合法 data URI 头像通过校验并原样提交', async () => {
      updateAgent.mockResolvedValue({ ...baseAgent })
      listEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
      updateEffects.mockResolvedValue({ effects_revision: 2, effects: [effA, effB] })
      getAgent.mockResolvedValue({ ...baseAgent })
      const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ avatar: 'data:image/png;base64,AAAA' }))

      let ok = false
      await act(async () => {
        ok = await result.current.save()
      })
      expect(ok).toBe(true)
      expect(updateAgent).toHaveBeenCalledWith(
        'agent-1',
        expect.objectContaining({ avatar: 'data:image/png;base64,AAAA' }),
      )
    })

    it('非白名单 MIME 的 data URI 拒绝保存', async () => {
      const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ avatar: 'data:image/svg+xml;base64,PHN2Zz4=' }))

      let ok = true
      await act(async () => {
        ok = await result.current.save()
      })
      expect(ok).toBe(false)
      expect(result.current.fieldErrors.avatar).toBeTruthy()
      expect(updateAgent).not.toHaveBeenCalled()
    })

    it('超长 URL 头像拒绝保存(>2048)', async () => {
      const { result } = renderHook(() => useAgentEditorFlow(baseAgent, effectsData), { wrapper })
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ avatar: `https://example.com/${'a'.repeat(2050)}` }))

      let ok = true
      await act(async () => {
        ok = await result.current.save()
      })
      expect(ok).toBe(false)
      expect(result.current.fieldErrors.avatar).toBe('tooLong')
      expect(updateAgent).not.toHaveBeenCalled()
    })
  })

  describe('服用同步 reconcileConsumedEffects + mutationBlocked', () => {
    it('reconcile:服用结果并入草稿(保留基础资料/用户权重/删除意图),刷新能力基线', () => {
      const fresh: AgentEffectsData = { effects_revision: 3, effects: [effA, effB, effC] }
      const onEffectsRefreshed = vi.fn()
      const { result } = renderHook(
        () => useAgentEditorFlow(baseAgent, effectsData, onEffectsRefreshed),
        { wrapper },
      )
      act(() => result.current.beginEdit())
      // 用户编辑:基础资料改名 + B 权重调为 2 + A 从草稿移除(待删除意图)
      act(() => result.current.updateDraft({ name: '改名' }))
      act(() => result.current.updateDraft({ effects: [{ key: 'eff-b', effect_id: 'eff-b', weight: 2 }] }))

      let ok = false
      act(() => {
        ok = result.current.reconcileConsumedEffects('agent-1', fresh)
      })
      expect(ok).toBe(true)
      // 基础资料草稿保留
      expect(result.current.draft.name).toBe('改名')
      // A 待删除意图保留(不复活);B 保留用户权重;新能力 C 追加末尾(服务端权重)
      expect(result.current.draft.effects).toEqual([
        { key: 'eff-b', effect_id: 'eff-b', weight: 2 },
        { key: 'eff-c', effect_id: 'eff-c', weight: 3 },
      ])
      // 基线刷新(供 dirty 比较与后续保存)
      expect(onEffectsRefreshed).toHaveBeenCalledWith(fresh)
    })

    it('重复同步同一 fresh:不重复追加(幂等)', () => {
      const fresh: AgentEffectsData = { effects_revision: 3, effects: [effA, effB, effC] }
      const onEffectsRefreshed = vi.fn()
      const { result } = renderHook(
        () => useAgentEditorFlow(baseAgent, effectsData, onEffectsRefreshed),
        { wrapper },
      )
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ effects: [{ key: 'eff-b', effect_id: 'eff-b', weight: 2 }] }))

      let first = false
      let second = false
      act(() => {
        first = result.current.reconcileConsumedEffects('agent-1', fresh)
      })
      act(() => {
        second = result.current.reconcileConsumedEffects('agent-1', fresh)
      })
      expect(first).toBe(true)
      expect(second).toBe(true)
      expect(result.current.draft.effects).toEqual([
        { key: 'eff-b', effect_id: 'eff-b', weight: 2 },
        { key: 'eff-c', effect_id: 'eff-c', weight: 3 },
      ])
    })

    it('道人不匹配:拒绝旧响应,零写入', () => {
      const fresh: AgentEffectsData = { effects_revision: 3, effects: [effA, effB, effC] }
      const onEffectsRefreshed = vi.fn()
      const { result } = renderHook(
        () => useAgentEditorFlow(baseAgent, effectsData, onEffectsRefreshed),
        { wrapper },
      )
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ effects: [{ key: 'eff-b', effect_id: 'eff-b', weight: 2 }] }))

      let ok = true
      act(() => {
        ok = result.current.reconcileConsumedEffects('agent-2', fresh)
      })
      expect(ok).toBe(false)
      expect(onEffectsRefreshed).not.toHaveBeenCalled()
      // 草稿零变化
      expect(result.current.draft.effects).toEqual([{ key: 'eff-b', effect_id: 'eff-b', weight: 2 }])
    })

    it('切换到新道人后,旧道人的服用响应被拒绝且不写入新草稿', () => {
      const otherAgent: AgentDetail = { ...baseAgent, id: 'agent-2', name: '哪吒' }
      const fresh: AgentEffectsData = { effects_revision: 3, effects: [effA, effB, effC] }
      const onEffectsRefreshed = vi.fn()
      const { result, rerender } = renderHook(
        ({ agent }) => useAgentEditorFlow(agent, effectsData, onEffectsRefreshed),
        { wrapper, initialProps: { agent: baseAgent } },
      )
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ name: '改老君' }))
      // 切到另一位道人 → 整体复位
      rerender({ agent: otherAgent })
      expect(result.current.mode).toBe('readonly')
      expect(result.current.draft.name).toBe('哪吒')
      // 旧道人的服用响应(agentId=agent-1)到达 → 拒绝
      let ok = true
      act(() => {
        ok = result.current.reconcileConsumedEffects('agent-1', fresh)
      })
      expect(ok).toBe(false)
      expect(result.current.draft.name).toBe('哪吒')
      expect(onEffectsRefreshed).not.toHaveBeenCalled()
    })

    it('mutationBlocked:保存被拒绝,零 API', async () => {
      const { result } = renderHook(
        () => useAgentEditorFlow(baseAgent, effectsData, undefined, { mutationBlocked: true }),
        { wrapper },
      )
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ name: '改名' }))

      let ok = true
      await act(async () => {
        ok = await result.current.save()
      })
      expect(ok).toBe(false)
      expect(updateAgent).not.toHaveBeenCalled()
      expect(removeEffect).not.toHaveBeenCalled()
      expect(updateEffects).not.toHaveBeenCalled()
      expect(getAgent).not.toHaveBeenCalled()
      // 编辑态与草稿保留
      expect(result.current.mode).toBe('editing')
      expect(result.current.draft.name).toBe('改名')
    })

    it('mutationBlocked:discard 不执行,草稿与编辑态保留', () => {
      const { result } = renderHook(
        () => useAgentEditorFlow(baseAgent, effectsData, undefined, { mutationBlocked: true }),
        { wrapper },
      )
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ name: '改名' }))
      expect(result.current.dirty).toBe(true)

      act(() => result.current.discard())
      expect(result.current.mode).toBe('editing')
      expect(result.current.draft.name).toBe('改名')
      expect(result.current.dirty).toBe(true)
    })

    it('mutationBlocked 不影响 reconcileConsumedEffects', () => {
      const fresh: AgentEffectsData = { effects_revision: 3, effects: [effA, effB, effC] }
      const onEffectsRefreshed = vi.fn()
      const { result } = renderHook(
        () => useAgentEditorFlow(baseAgent, effectsData, onEffectsRefreshed, { mutationBlocked: true }),
        { wrapper },
      )
      act(() => result.current.beginEdit())
      act(() => result.current.updateDraft({ effects: [{ key: 'eff-b', effect_id: 'eff-b', weight: 2 }] }))

      let ok = false
      act(() => {
        ok = result.current.reconcileConsumedEffects('agent-1', fresh)
      })
      expect(ok).toBe(true)
      expect(result.current.draft.effects).toEqual([
        { key: 'eff-b', effect_id: 'eff-b', weight: 2 },
        { key: 'eff-c', effect_id: 'eff-c', weight: 3 },
      ])
      expect(onEffectsRefreshed).toHaveBeenCalledWith(fresh)
    })

    it('保存成功后基线 = updateEffects 返回值(而非重读 fresh)', async () => {
      updateAgent.mockResolvedValue({ ...baseAgent })
      getAgent.mockResolvedValue({ ...baseAgent })
      // 第三步重读:revision 3,无新能力(拿乐观锁)
      listEffects.mockResolvedValue({ effects_revision: 3, effects: [effA, effB] })
      // 服务端编排后的真相:revision 4,含新能力 C
      const updatedEffects: AgentEffectsData = { effects_revision: 4, effects: [effA, effB, effC] }
      updateEffects.mockResolvedValue(updatedEffects)
      const onEffectsRefreshed = vi.fn()
      const { result } = renderHook(
        () => useAgentEditorFlow(baseAgent, effectsData, onEffectsRefreshed),
        { wrapper },
      )
      act(() => result.current.beginEdit())

      let ok = false
      await act(async () => {
        ok = await result.current.save()
      })
      expect(ok).toBe(true)
      // 基线刷新必须来自 updateEffects 返回值,且只刷新一次
      expect(onEffectsRefreshed).toHaveBeenCalledTimes(1)
      expect(onEffectsRefreshed).toHaveBeenCalledWith(updatedEffects)
      // 草稿回源自 updateEffects 返回值
      expect(result.current.draft.effects).toEqual([
        { key: 'eff-a', effect_id: 'eff-a', weight: 2 },
        { key: 'eff-b', effect_id: 'eff-b', weight: 1 },
        { key: 'eff-c', effect_id: 'eff-c', weight: 3 },
      ])
      expect(result.current.dirty).toBe(false)
    })
  })
})
