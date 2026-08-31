'use client'

/**
 * 道人编辑器流程 hook（金丹消耗品重构任务 6：agent_pills → effects）
 * 只读/编辑双态 + 独立草稿 + 写后重读回源：
 * - 草稿为深拷贝，与源道人/能力列表零引用共享
 * - 保存四步：PUT 基础资料 → 移除草稿中删除的能力（幂等 key + 404 容忍）→
 *   重读能力列表取最新乐观锁 → PUT 全量编排 → GET 回读；
 *   任一步失败保留草稿与编辑态
 * - 409 agent.effects_conflict（乐观锁过期/提交集与活跃集不一致）：保留草稿与
 *   用户编辑，重新拉取能力列表刷新基线并把新能力并入草稿（仍在集合中的行保留
 *   用户权重/顺序），提示用户重新确认；不把服务端结果覆盖用户编辑
 * - 移除能力不返还金丹（原实例保持 consumed_by_agent）
 */
import { useCallback, useMemo, useRef, useState } from 'react'

import { useAgent } from '@/contexts/AgentContext'
import { validateAvatarField } from '@/lib/avatar-validation'
import { clearPendingOperation, startPendingOperation } from '@/lib/pending-operations'
import { getAgent, updateAgent } from '@/services/agentService'
import { ApiError } from '@/services/api'
import { listEffects, removeEffect, updateEffects } from '@/services/pillInventoryService'
import type {
  AgentDetail,
  AgentEditorDraft,
  AgentEffect,
  AgentEffectDraftItem,
  DistillationDraft,
} from '@/services/types'

export type AgentEditorMode = 'readonly' | 'editing'
export type AgentSaveStatus = 'idle' | 'submitting' | 'error' | 'conflict'

/** 能力列表基线（effects_revision 供 PUT 乐观锁；与 GET /agents/:id/effects 响应同形） */
export interface AgentEffectsData {
  effects: AgentEffect[]
  effects_revision: number
}

export interface AgentEditorFlow {
  mode: AgentEditorMode
  draft: AgentEditorDraft
  dirty: boolean
  saveStatus: AgentSaveStatus
  /** 字段级校验错误（如 name / effects.<key>.weight），保存前客户端校验填充 */
  fieldErrors: Record<string, string>
  /** 进入编辑态 */
  beginEdit(): void
  /** 局部更新草稿 */
  updateDraft(patch: Partial<AgentEditorDraft>): void
  /** 保存草稿：基础资料 → 移除缺失能力 → 全量编排 → GET 回读；任何一步失败保留草稿与编辑态 */
  save(): Promise<boolean>
  /** 放弃修改并退出到只读 */
  discard(): void
  /** 女娲蒸馏草稿显式落表（仅调用此方法才写入 name/personality；不触碰能力编排） */
  applyNuwaDraft(draft: DistillationDraft): void
}

/** 从道人详情 + 能力列表构造独立草稿（能力行重建，与源对象零引用共享） */
function buildDraft(agent: AgentDetail | null, effectsData: AgentEffectsData | null): AgentEditorDraft {
  if (!agent) return emptyDraft()
  return {
    name: agent.name,
    avatar: agent.avatar ?? '',
    personality: agent.personality ?? '',
    model_name: agent.model_name,
    proactivity: agent.proactivity,
    status: agent.status,
    // 能力行 key 与 effect_id 同值；顺序 = listEffects 返回顺序（sort_order 升序）
    effects: (effectsData?.effects ?? []).map(ef => ({
      key: ef.id,
      effect_id: ef.id,
      weight: ef.weight,
    })),
  }
}

function emptyDraft(): AgentEditorDraft {
  return {
    name: '',
    avatar: '',
    personality: '',
    model_name: '',
    proactivity: 50,
    status: 'active',
    effects: [],
  }
}

/** 键序无关的稳定序列化（dirty 比较用，避免对象重建导致误报） */
function stableStringify(value: unknown): string {
  if (value === null || typeof value !== 'object') return JSON.stringify(value) ?? 'undefined'
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(',')}]`
  const entries = Object.keys(value as Record<string, unknown>)
    .sort()
    .map(key => `${JSON.stringify(key)}:${stableStringify((value as Record<string, unknown>)[key])}`)
  return `{${entries.join(',')}}`
}

/** 客户端字段级校验：返回 fieldErrors（空对象 = 通过） */
function validateDraft(draft: AgentEditorDraft): Record<string, string> {
  const errors: Record<string, string> = {}
  if (!draft.name.trim()) errors.name = 'required'
  const avatarError = validateAvatarField(draft.avatar)
  if (avatarError) errors.avatar = avatarError
  for (const effect of draft.effects) {
    if (effect.weight <= 0 || effect.weight > 10) errors[`effects.${effect.key}.weight`] = 'range'
  }
  return errors
}

/**
 * 409 冲突后把新活跃集并入草稿：
 * 仍在集合中的行保留用户权重/顺序；已被并发移除的行丢弃；
 * 新吸收的能力追加到末尾（权重沿用快照）。
 * 保证返回的提交集与活跃集一致，重新确认不会再因集合差异 409。
 */
function mergeFreshEffects(
  draftEffects: AgentEffectDraftItem[],
  fresh: AgentEffect[],
): AgentEffectDraftItem[] {
  const freshIds = new Set(fresh.map(e => e.id))
  const kept = draftEffects.filter(item => freshIds.has(item.effect_id))
  const keptIds = new Set(kept.map(i => i.effect_id))
  const appended = fresh
    .filter(e => !keptIds.has(e.id))
    .map(e => ({ key: e.id, effect_id: e.id, weight: e.weight }))
  return [...kept, ...appended]
}

export function useAgentEditorFlow(
  agent: AgentDetail | null,
  effectsData: AgentEffectsData | null,
  onEffectsRefreshed?: (fresh: AgentEffectsData) => void,
): AgentEditorFlow {
  const { dispatch } = useAgent()
  const [mode, setMode] = useState<AgentEditorMode>('readonly')
  const [saveStatus, setSaveStatus] = useState<AgentSaveStatus>('idle')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [draft, setDraft] = useState<AgentEditorDraft>(() =>
    agent ? buildDraft(agent, effectsData) : emptyDraft(),
  )
  /** 提交中去重（双击/重复触发只产生一次写请求） */
  const submittingRef = useRef(false)

  // 切换到另一位道人时整体复位（渲染期间派生调整，React 官方推荐模式）
  const [prevAgentId, setPrevAgentId] = useState(agent?.id)
  if (agent?.id !== prevAgentId) {
    setPrevAgentId(agent?.id)
    setMode('readonly')
    setSaveStatus('idle')
    setFieldErrors({})
    setDraft(agent ? buildDraft(agent, effectsData) : emptyDraft())
  }

  const baseline = useMemo(
    () => (agent ? buildDraft(agent, effectsData) : emptyDraft()),
    [agent, effectsData],
  )
  const dirty = useMemo(
    () => mode === 'editing' && stableStringify(draft) !== stableStringify(baseline),
    [mode, draft, baseline],
  )

  const beginEdit = useCallback(() => {
    if (!agent) return
    setDraft(buildDraft(agent, effectsData))
    setFieldErrors({})
    setSaveStatus('idle')
    setMode('editing')
  }, [agent, effectsData])

  const updateDraft = useCallback((patch: Partial<AgentEditorDraft>) => {
    setDraft(prev => ({ ...prev, ...patch }))
    setFieldErrors({})
  }, [])

  const discard = useCallback(() => {
    setDraft(agent ? buildDraft(agent, effectsData) : emptyDraft())
    setFieldErrors({})
    setSaveStatus('idle')
    setMode('readonly')
  }, [agent, effectsData])

  const save = useCallback(async (): Promise<boolean> => {
    if (!agent || mode !== 'editing' || submittingRef.current) return false
    // 字段级校验：失败不发起任何 API
    const errors = validateDraft(draft)
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors)
      return false
    }
    submittingRef.current = true
    setSaveStatus('submitting')
    setFieldErrors({})
    try {
      // 第一步：基础资料
      await updateAgent(agent.id, {
        name: draft.name,
        avatar: draft.avatar.trim(),
        personality: draft.personality,
        model_name: draft.model_name,
        proactivity: draft.proactivity,
        status: draft.status,
      })

      // 第二步：移除草稿中删除的能力。
      // 每个移除是明确用户动作，走幂等 key + pending 记录（重试沿用原 key）；
      // 已移除（并发窗口删除 → 404 dao.agent.remove_effect_uuid）视为成功。
      const baselineIds = new Set((effectsData?.effects ?? []).map(e => e.id))
      for (const effectId of baselineIds) {
        if (draft.effects.some(item => item.effect_id === effectId)) continue
        const key = startPendingOperation('remove_effect', `${agent.id}→${effectId}`)
        try {
          await removeEffect(key, agent.id, effectId)
        } catch (error) {
          if (error instanceof ApiError && error.errorCode === 'dao.agent.remove_effect_uuid') {
            // 已移除：视为成功，继续
          } else {
            throw error
          }
        }
        clearPendingOperation(key)
      }

      // 第三步：重读能力列表（remove 会递增 effects_revision，乐观锁必须以最新值为准）
      const fresh = await listEffects(agent.id)
      onEffectsRefreshed?.(fresh)

      // 第四步：全量编排（乐观锁；提交集必须等于活跃集，sortOrder = 草稿数组下标）
      await updateEffects(
        agent.id,
        fresh.effects_revision,
        draft.effects.map((item, index) => ({
          effectId: item.effect_id,
          weight: item.weight,
          sortOrder: index,
        })),
      )

      // 第五步：写后重读；只有 GET 成功才算保存完成，最终状态以 GET 对象为准
      const freshAgent = await getAgent(agent.id)
      dispatch({ type: 'UPDATE_AGENT', payload: freshAgent })
      dispatch({ type: 'SET_CURRENT_AGENT', payload: freshAgent })
      setDraft(buildDraft(freshAgent, fresh))
      setSaveStatus('idle')
      setMode('readonly')
      return true
    } catch (error) {
      if (error instanceof ApiError && error.errorCode === 'service.agent.effects_conflict') {
        // 冲突：保留草稿与用户编辑；刷新能力基线并合并新能力，提示用户重新确认
        try {
          const fresh = await listEffects(agent.id)
          onEffectsRefreshed?.(fresh)
          setDraft(prev => ({ ...prev, effects: mergeFreshEffects(prev.effects, fresh.effects) }))
          setSaveStatus('conflict')
        } catch {
          setSaveStatus('error')
        }
        return false
      }
      // 失败保留草稿与编辑态，不丢用户输入
      setSaveStatus('error')
      return false
    } finally {
      submittingRef.current = false
    }
  }, [agent, mode, draft, effectsData, dispatch, onEffectsRefreshed])

  const applyNuwaDraft = useCallback((incoming: DistillationDraft) => {
    if (!agent) return
    setDraft(prev => ({ ...prev, name: incoming.name, personality: incoming.persona_summary }))
    setFieldErrors({})
    setSaveStatus('idle')
    setMode('editing')
  }, [agent])

  return {
    mode,
    draft,
    dirty,
    saveStatus,
    fieldErrors,
    beginEdit,
    updateDraft,
    save,
    discard,
    applyNuwaDraft,
  }
}
