import type { ReactNode } from 'react'
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useRecipeEditorFlow } from '@/hooks/use-recipe-editor-flow'
import { ApiError } from '@/services/api'
import type { RecipeDetail } from '@/services/types'

const getRecipe = vi.hoisted(() => vi.fn())
const updateRecipe = vi.hoisted(() => vi.fn())

vi.mock('@/services/recipeService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/recipeService')>()
  return { ...actual, getRecipe, updateRecipe }
})

const wrapper = ({ children }: { children: ReactNode }) => <>{children}</>

const recipe: RecipeDetail = {
  id: '11111111-1111-4111-8111-111111111111',
  name: '文言文丹方',
  description: '以文言为骨',
  skill_schema: {
    identity_card: '一位书院山长',
    expression_dna: { formality: 0.8, vocabulary: ['之乎者也'] },
    mental_models: [{ name: '经义', application: '以经解事' }],
  },
  tags: ['古风'],
  author: '太上老君',
  version_label: '2.1.0',
  revision: 3,
  current_revision_id: '22222222-2222-4222-8222-222222222222',
  created_at: '2026-08-20T00:00:00Z',
}

describe('useRecipeEditorFlow 丹方编辑（生成不可变新版本）', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    sessionStorage.clear()
  })

  it('beginEdit 进入编辑态并生成独立草稿（与源零引用共享）', () => {
    const { result } = renderHook(() => useRecipeEditorFlow(recipe), { wrapper })
    expect(result.current.mode).toBe('readonly')
    expect(result.current.dirty).toBe(false)

    act(() => result.current.beginEdit())
    expect(result.current.mode).toBe('editing')
    expect(result.current.draft.name).toBe('文言文丹方')

    // 修改草稿不影响源对象
    act(() => result.current.updateDraft({ name: '新名' }))
    expect(result.current.draft.name).toBe('新名')
    expect(recipe.name).toBe('文言文丹方')
  })

  it('updateDraft 后 dirty 为 true，discard 回到只读', () => {
    const { result } = renderHook(() => useRecipeEditorFlow(recipe), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ description: '改了' }))
    expect(result.current.dirty).toBe(true)

    act(() => result.current.discard())
    expect(result.current.mode).toBe('readonly')
    expect(result.current.draft.description).toBe('以文言为骨')
    expect(result.current.dirty).toBe(false)
  })

  it('save 提交 expected_revision_id=当前版本；成功后写后重读回只读并清除 pending', async () => {
    const updated: RecipeDetail = { ...recipe, name: '文言文丹方 v4', revision: 4 }
    getRecipe.mockResolvedValue(updated)
    updateRecipe.mockResolvedValue({
      operation_id: 'op-1',
      recipe_id: recipe.id,
      revision_id: '33333333-3333-4333-8333-333333333333',
    })

    // 容器对象避免 TS 控制流把回调中赋值的变量收窄为 never
    const saved: { value: RecipeDetail | null } = { value: null }
    const { result } = renderHook(
      () => useRecipeEditorFlow(recipe, { onSaved: (r) => { saved.value = r } }),
      { wrapper },
    )
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '文言文丹方 v4' }))

    let ok = false
    await act(async () => {
      ok = await result.current.save()
    })

    expect(ok).toBe(true)
    const [, recipeId, expectedRevisionId, draft] = updateRecipe.mock.calls[0]
    expect(recipeId).toBe(recipe.id)
    expect(expectedRevisionId).toBe(recipe.current_revision_id)
    expect(draft.name).toBe('文言文丹方 v4')
    expect(getRecipe).toHaveBeenCalledWith(recipe.id)
    expect(result.current.mode).toBe('readonly')
    expect(result.current.saveStatus).toBe('idle')
    expect(saved.value?.name).toBe('文言文丹方 v4')
  })

  it('save 网络失败走断线恢复：已提交则按成功处理；未提交保留错误可重试', async () => {
    // 第一次：网络错误（无状态）
    updateRecipe.mockRejectedValueOnce(new ApiError('网络请求失败', 0))
    // 恢复查询 404 → 未提交
    const getOperation = vi.spyOn(
      await import('@/services/pillInventoryService'),
      'getOperation',
    )
    getOperation.mockRejectedValueOnce(new ApiError('not found', 404))
    getOperation.mockRejectedValueOnce(new ApiError('network', 0))

    const { result } = renderHook(() => useRecipeEditorFlow(recipe), { wrapper })
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '试试重试' }))

    await act(async () => {
      await result.current.save()
    })
    expect(result.current.mode).toBe('editing')
    expect(result.current.saveStatus).toBe('error')
    // pending 记录仍在：同动作同目标复用原 key
    const key = updateRecipe.mock.calls[0][0]
    expect(key).toBeTruthy()

    // 第二次调用 updateRecipe 必须使用同一 key
    updateRecipe.mockResolvedValueOnce({
      operation_id: key,
      recipe_id: recipe.id,
      revision_id: '44444444-4444-4444-8444-444444444444',
    })
    getRecipe.mockResolvedValueOnce({ ...recipe, name: '试试重试' })
    await act(async () => {
      await result.current.save()
    })
    expect(updateRecipe.mock.calls[1][0]).toBe(key)
    expect(result.current.mode).toBe('readonly')
  })

  it('409 版本冲突：conflict=true、停留编辑态、提示刷新后重试', async () => {
    updateRecipe.mockRejectedValueOnce(
      new ApiError('版本冲突', 409, { error_code: 'pill_inventory.revision_conflict' }),
    )
    const { result } = renderHook(() => useRecipeEditorFlow(recipe), { wrapper })
    act(() => result.current.beginEdit())

    let ok = true
    await act(async () => {
      ok = await result.current.save()
    })
    expect(ok).toBe(false)
    expect(result.current.conflict).toBe(true)
    expect(result.current.saveStatus).toBe('error')
    expect(result.current.mode).toBe('editing')
  })

  it('切换到另一丹方时整体复位', () => {
    const other: RecipeDetail = { ...recipe, id: '99999999-9999-4999-8999-999999999999', name: '另一丹方' }
    const { result, rerender } = renderHook(
      ({ r }: { r: RecipeDetail | null }) => useRecipeEditorFlow(r),
      { wrapper, initialProps: { r: recipe } },
    )
    act(() => result.current.beginEdit())
    act(() => result.current.updateDraft({ name: '改过的名' }))

    rerender({ r: other })
    expect(result.current.mode).toBe('readonly')
    expect(result.current.draft.name).toBe('另一丹方')
  })
})
