'use client'

/**
 * 丹方编辑器流程 hook（金丹消耗品重构任务 6）
 * 编辑丹方 = 生成不可变新版本（revision+1），旧版本金丹/能力不受影响：
 * - 草稿为深拷贝，与源丹方零引用共享
 * - 保存 = updateRecipe(幂等 key, expected_revision_id=当前版本)，写后重读回源
 * - 提交中去重：连续触发只产生一次写请求；断线恢复先 GET operation，同 key 重试
 * - 409 版本冲突单独标记 conflict，提示刷新后重试（不自动覆盖他人修改）
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import {
  clearPendingOperation,
  recoverOperation,
  startPendingOperation,
} from '@/lib/pending-operations'
import { getRecipe, updateRecipe } from '@/services/recipeService'
import type { RecipeDetail } from '@/services/types'

export type RecipeEditorMode = 'readonly' | 'editing'
export type RecipeSaveStatus = 'idle' | 'submitting' | 'error'

/** 编辑器草稿：所有字段具象化（skill_schema 必存在，供结构化编辑器读写） */
export interface RecipeEditorDraft {
  name: string
  description: string
  author: string
  version_label: string
  tags: string[]
  skill_schema: NonNullable<RecipeDetail['skill_schema']>
}

export interface RecipeEditorFlow {
  mode: RecipeEditorMode
  draft: RecipeEditorDraft
  dirty: boolean
  saveStatus: RecipeSaveStatus
  /** 409 乐观锁冲突：丹方已被其他会话更新 */
  conflict: boolean
  /** 进入编辑态（生成新版本草稿） */
  beginEdit(): void
  /** 保存新版本：写后重读回源；成功返回 true 并触发 onSaved */
  save(): Promise<boolean>
  /** 放弃修改，回到只读 */
  discard(): void
  /** 局部更新草稿 */
  updateDraft(patch: Partial<RecipeEditorDraft>): void
}

export interface RecipeEditorFlowOptions {
  /** 保存成功并重读后的回调（携带最新丹方，供页面刷新显示与计数） */
  onSaved?: (recipe: RecipeDetail) => void
}

/** 从丹方详情构造独立草稿（JSON 深拷贝，与源对象零引用共享） */
function buildDraft(recipe: RecipeDetail): RecipeEditorDraft {
  return {
    name: recipe.name,
    description: recipe.description ?? '',
    author: recipe.author ?? '',
    version_label: recipe.version_label ?? '',
    tags: [...(recipe.tags ?? [])],
    skill_schema: JSON.parse(JSON.stringify(recipe.skill_schema ?? {})),
  }
}

function emptyDraft(): RecipeEditorDraft {
  return { name: '', description: '', author: '', version_label: '', tags: [], skill_schema: {} }
}

/** 键序无关的稳定序列化（dirty 比较用，避免对象重建导致误报） */
function stableStringify(value: unknown): string {
  if (value === null || typeof value !== 'object') return JSON.stringify(value) ?? 'undefined'
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(',')}]`
  const entries = Object.keys(value as Record<string, unknown>)
    .sort()
    .map((key) => `${JSON.stringify(key)}:${stableStringify((value as Record<string, unknown>)[key])}`)
  return `{${entries.join(',')}}`
}

export function useRecipeEditorFlow(
  recipe: RecipeDetail | null,
  options?: RecipeEditorFlowOptions,
): RecipeEditorFlow {
  const [mode, setMode] = useState<RecipeEditorMode>('readonly')
  const [saveStatus, setSaveStatus] = useState<RecipeSaveStatus>('idle')
  const [conflict, setConflict] = useState(false)
  const [draft, setDraft] = useState<RecipeEditorDraft>(() =>
    recipe ? buildDraft(recipe) : emptyDraft(),
  )
  /** 提交中去重（双击/重复触发只产生一次写请求） */
  const submittingRef = useRef(false)
  const onSavedRef = useRef(options?.onSaved)
  // 最新回调存入 ref（在 effect 中更新，避免渲染期写 ref）
  useEffect(() => {
    onSavedRef.current = options?.onSaved
  })

  // 切换到另一份丹方时整体复位（渲染期间派生调整，React 官方推荐模式）
  const [prevRecipeId, setPrevRecipeId] = useState(recipe?.id)
  if (recipe?.id !== prevRecipeId) {
    setPrevRecipeId(recipe?.id)
    setMode('readonly')
    setSaveStatus('idle')
    setConflict(false)
    setDraft(recipe ? buildDraft(recipe) : emptyDraft())
  }

  const baseline = useMemo(() => (recipe ? buildDraft(recipe) : emptyDraft()), [recipe])
  const dirty = useMemo(
    () => mode === 'editing' && stableStringify(draft) !== stableStringify(baseline),
    [mode, draft, baseline],
  )

  const beginEdit = useCallback(() => {
    if (!recipe) return
    setDraft(buildDraft(recipe))
    setConflict(false)
    setSaveStatus('idle')
    setMode('editing')
  }, [recipe])

  const discard = useCallback(() => {
    setMode('readonly')
    setSaveStatus('idle')
    setConflict(false)
    if (recipe) setDraft(buildDraft(recipe))
  }, [recipe])

  const updateDraft = useCallback((patch: Partial<RecipeEditorDraft>) => {
    setDraft((prev) => ({ ...prev, ...patch }))
  }, [])

  const save = useCallback(async (): Promise<boolean> => {
    if (!recipe || submittingRef.current) return false
    submittingRef.current = true
    setSaveStatus('submitting')
    setConflict(false)
    // 每个明确保存动作一个幂等 key；pending 期间同目标复用（重试不换 key）
    const key = startPendingOperation('update_recipe', recipe.name)
    let committed = false
    try {
      await updateRecipe(key, recipe.id, recipe.current_revision_id, {
        name: draft.name.trim(),
        description: draft.description,
        author: draft.author,
        version_label: draft.version_label,
        tags: draft.tags,
        skill_schema: draft.skill_schema,
      })
      committed = true
    } catch (error) {
      // 409 乐观锁冲突最先判断：不触发幂等恢复，单独标记冲突
      if (error instanceof Error && 'status' in error && (error as { status: number }).status === 409) {
        setConflict(true)
        setSaveStatus('error')
        submittingRef.current = false
        return false
      }
      // 断线恢复：先查已提交结果；404 = 未提交，保留原 key 供重试
      try {
        committed = (await recoverOperation(key)) !== null
      } catch {
        setSaveStatus('error')
        submittingRef.current = false
        return false
      }
      if (!committed) {
        setSaveStatus('error')
        submittingRef.current = false
        return false
      }
    }

    // 写后重读：只有 GET 成功才算保存完成
    try {
      const updated = await getRecipe(recipe.id)
      clearPendingOperation(key)
      setMode('readonly')
      setSaveStatus('idle')
      setDraft(buildDraft(updated))
      onSavedRef.current?.(updated)
      return true
    } catch {
      setSaveStatus('error')
      return false
    } finally {
      submittingRef.current = false
    }
  }, [recipe, draft])

  return { mode, draft, dirty, saveStatus, conflict, beginEdit, save, discard, updateDraft }
}
