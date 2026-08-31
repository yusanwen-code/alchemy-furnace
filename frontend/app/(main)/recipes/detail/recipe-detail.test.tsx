import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import RecipeDetailPage from '@/app/(main)/recipes/detail/recipe-detail'
import { ApiError } from '@/services/api'
import type { RecipeDetail } from '@/services/types'

const getRecipe = vi.hoisted(() => vi.fn())
const updateRecipe = vi.hoisted(() => vi.fn())
const craftPill = vi.hoisted(() => vi.fn())
const getOperation = vi.hoisted(() => vi.fn())

vi.mock('@/services/recipeService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/recipeService')>()
  return { ...actual, getRecipe, updateRecipe, craftPill }
})

vi.mock('@/services/pillInventoryService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/pillInventoryService')>()
  return { ...actual, getOperation }
})

// key 透传：文案断言只关心键/角色，不依赖具体语言
vi.mock('next-intl', () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

const RECIPE_ID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const REVISION_ID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'

const recipe: RecipeDetail = {
  id: RECIPE_ID,
  name: '文言文丹方',
  description: '以文言为骨',
  skill_schema: {
    identity_card: '一位书院山长',
    expression_dna: { formality: 0.8, vocabulary: ['之乎者也'] },
    mental_models: [{ name: '经义', application: '以经解事' }],
    values: ['守正'],
  },
  tags: ['古风'],
  author: '太上老君',
  version_label: '2.1.0',
  revision: 3,
  current_revision_id: REVISION_ID,
  archived_at: null,
  created_at: '2026-08-20T00:00:00Z',
}

describe('RecipeDetailPage 丹方详情（读/编辑双态）', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    sessionStorage.clear()
    getRecipe.mockResolvedValue(recipe)
  })

  afterEach(() => cleanup())

  it('无 id 展示「链接无效」，不请求 API', () => {
    render(<RecipeDetailPage />)
    expect(screen.getByText('invalidLink')).toBeInTheDocument()
    expect(getRecipe).not.toHaveBeenCalled()
  })

  it('只读态展示丹方信息与动作列（炼制/导出/编辑）', async () => {
    render(<RecipeDetailPage recipeId={RECIPE_ID} />)
    expect(await screen.findByText('文言文丹方')).toBeInTheDocument()
    expect(screen.getByText('craftCta')).toBeInTheDocument()
    expect(screen.getByText('exportSkillCta')).toBeInTheDocument()
    expect(screen.getByText('editCta')).toBeInTheDocument()
  })

  it('「炼制 1 枚」携带当前不可变版本与幂等 key，成功后展示反馈', async () => {
    craftPill.mockResolvedValue({ operation_id: 'op-1', recipe_id: RECIPE_ID })
    render(<RecipeDetailPage recipeId={RECIPE_ID} />)
    await screen.findByText('文言文丹方')

    await userEvent.click(screen.getByText('craftCta'))
    expect(await screen.findByText('crafted')).toBeInTheDocument()
    const [key, recipeId, revisionId] = craftPill.mock.calls[0]
    expect(recipeId).toBe(RECIPE_ID)
    expect(revisionId).toBe(REVISION_ID)
    expect(key).toBeTruthy()
  })

  it('initialEdit=true 就绪后直接进入编辑态（新版本草稿）', async () => {
    render(<RecipeDetailPage recipeId={RECIPE_ID} initialEdit />)
    expect(await screen.findByText('editSave')).toBeInTheDocument()
    expect(screen.queryByText('editCta')).not.toBeInTheDocument()
    // 草稿回填源丹方
    expect(screen.getByDisplayValue('文言文丹方')).toBeInTheDocument()
  })

  it('归档丹方忽略 initialEdit，保持只读并提示', async () => {
    getRecipe.mockResolvedValue({ ...recipe, archived_at: '2026-08-25T00:00:00Z' })
    render(<RecipeDetailPage recipeId={RECIPE_ID} initialEdit />)
    expect(await screen.findByText('archivedNotice')).toBeInTheDocument()
    expect(screen.queryByText('editSave')).not.toBeInTheDocument()
    expect(screen.getByText('craftCta')).toBeDisabled()
  })

  it('404 展示「不存在」', async () => {
    getRecipe.mockRejectedValue(new ApiError('not found', 404))
    render(<RecipeDetailPage recipeId={RECIPE_ID} />)
    expect(await screen.findByText('notFound')).toBeInTheDocument()
  })
})
