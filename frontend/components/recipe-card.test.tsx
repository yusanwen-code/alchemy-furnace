import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { RecipeCard } from '@/components/recipe-card'
import { ApiError } from '@/services/api'
import type { RecipeListItem } from '@/services/types'

const push = vi.hoisted(() => vi.fn())
const craftPill = vi.hoisted(() => vi.fn())
const getOperation = vi.hoisted(() => vi.fn())

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push }),
}))

// key 透传：卡片文案断言只关心键/角色，不依赖具体语言
vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock('@/services/recipeService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/recipeService')>()
  return { ...actual, craftPill }
})

vi.mock('@/services/pillInventoryService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/pillInventoryService')>()
  return { ...actual, getOperation }
})

const RECIPE_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const REVISION_ID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'

const recipe: RecipeListItem = {
  id: RECIPE_ID,
  name: '文言文丹方',
  current_revision_id: REVISION_ID,
  archived_at: null,
  created_at: '2026-08-20T00:00:00Z',
  available_count: 2,
  revision: 3,
}

/** 延迟可控的 promise（双击竞态用） */
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('RecipeCard 炼制 1 枚（幂等契约）', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    sessionStorage.clear()
  })

  afterEach(() => cleanup())

  it('双击「炼制 1 枚」只产生一次 craftPill（提交中按钮禁用）', async () => {
    const d = deferred<{ operation_id: string; recipe_id: string }>()
    craftPill.mockReturnValue(d.promise)

    render(<RecipeCard recipe={recipe} />)
    const craftBtn = screen.getByText('craftCta')
    await userEvent.click(craftBtn)
    await userEvent.click(craftBtn)

    expect(craftPill).toHaveBeenCalledTimes(1)
    const [key, recipeId, revisionId] = craftPill.mock.calls[0]
    expect(recipeId).toBe(RECIPE_ID)
    expect(revisionId).toBe(REVISION_ID)
    expect(key).toBeTruthy()

    d.resolve({ operation_id: key, recipe_id: RECIPE_ID })
    expect(await screen.findByText('crafted')).toBeInTheDocument()
  })

  it('炼制失败未提交：显示错误，重试沿用同一幂等 key', async () => {
    craftPill.mockRejectedValueOnce(new ApiError('网络请求失败', 0))
    getOperation.mockRejectedValueOnce(new ApiError('not found', 404))

    const { rerender } = render(<RecipeCard recipe={recipe} />)
    await userEvent.click(screen.getByText('craftCta'))

    expect(await screen.findByRole('alert')).toHaveTextContent('craftFailed')
    const firstKey = craftPill.mock.calls[0][0]
    expect(firstKey).toBeTruthy()

    // 同 key 重试成功（保留 pending 记录，action+label 去重）
    craftPill.mockResolvedValueOnce({ operation_id: firstKey, recipe_id: RECIPE_ID })
    rerender(<RecipeCard recipe={recipe} />)
    await userEvent.click(screen.getByText('craftCta'))

    expect(craftPill.mock.calls[1][0]).toBe(firstKey)
    expect(await screen.findByText('crafted')).toBeInTheDocument()
  })

  it('网络断但已提交：恢复查询命中即按成功处理并回调 onCrafted', async () => {
    const onCrafted = vi.fn()
    craftPill.mockRejectedValueOnce(new ApiError('网络请求失败', 0))
    getOperation.mockResolvedValueOnce({ operation_id: 'op-1', recipe_id: RECIPE_ID })

    render(<RecipeCard recipe={recipe} onCrafted={onCrafted} />)
    await userEvent.click(screen.getByText('craftCta'))

    expect(await screen.findByText('crafted')).toBeInTheDocument()
    expect(onCrafted).toHaveBeenCalledTimes(1)
  })

  it('归档丹方禁用炼制', () => {
    render(<RecipeCard recipe={{ ...recipe, archived_at: '2026-08-25T00:00:00Z' }} />)
    expect(screen.getByText('craftCta')).toBeDisabled()
  })

  it('「导出 Skill」打开丹方导出弹窗（不消耗库存）', async () => {
    render(<RecipeCard recipe={recipe} />)
    await userEvent.click(screen.getByText('exportSkillCta'))
    expect(screen.getByLabelText('closeModal')).toBeInTheDocument()
    expect(craftPill).not.toHaveBeenCalled()
  })

  it('「编辑新版本」直达详情编辑态地址', async () => {
    render(<RecipeCard recipe={recipe} />)
    fireEvent.click(screen.getByText('editCta'))
    expect(push).toHaveBeenCalledWith(`/recipes/detail?id=${RECIPE_ID}&edit=1`)
  })
})
