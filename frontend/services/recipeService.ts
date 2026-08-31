/**
 * 丹方服务（金丹消耗品重构任务 6）
 * 丹方永久保留；编辑生成不可变新版本，不影响旧金丹与已吸收能力。
 * 写操作（保存/编辑/归档/炼制）必须携带 Idempotency-Key（UUID），
 * 断线恢复先 GET /pill-operations/:id 再决定是否用原 key 重试。
 */
import { request } from './api'
import type {
  PillOperationResult,
  RecipeDetail,
  RecipeDraft,
  RecipeListItem,
  RecipeListParams,
  RecipeRevision,
} from './types'

/** Idempotency-Key 请求头（每个明确用户动作一个 key；重试沿用原 key） */
function idempotencyHeaders(key: string): Record<string, string> {
  return { 'Idempotency-Key': key }
}

/** 丹方分页列表（含各丹方可用库存计数） */
export function listRecipes(params: RecipeListParams = {}): Promise<{ total: number; items: RecipeListItem[] }> {
  return request('/recipes', {
    method: 'GET',
    params: {
      page: params.page,
      size: params.size,
      keyword: params.keyword,
      include_archived: params.include_archived,
    },
  })
}

/** 创建丹方；craftOne=true 同事务炼制一枚（幂等写操作） */
export function saveRecipe(key: string, draft: RecipeDraft, craftOne = false): Promise<PillOperationResult> {
  return request('/recipes', {
    method: 'POST',
    headers: idempotencyHeaders(key),
    body: JSON.stringify({ ...draft, craft_one: craftOne }),
  })
}

/** 丹方详情（含当前版本内容；任意状态可读） */
export function getRecipe(recipeId: string): Promise<RecipeDetail> {
  return request(`/recipes/${encodeURIComponent(recipeId)}`, { method: 'GET' })
}

/** 读指定不可变版本（归属校验：版本必须属于该丹方） */
export function getRecipeRevision(recipeId: string, revisionId: string): Promise<RecipeRevision> {
  return request(
    `/recipes/${encodeURIComponent(recipeId)}/revisions/${encodeURIComponent(revisionId)}`,
    { method: 'GET' }
  )
}

/**
 * 编辑丹方生成新版本（expectedRevisionId 必须匹配当前版本，冲突 409）
 * 产物为新版本；旧版本金丹/能力不受影响（幂等写操作）
 */
export function updateRecipe(
  key: string,
  recipeId: string,
  expectedRevisionId: string,
  draft: RecipeDraft
): Promise<PillOperationResult> {
  return request(`/recipes/${encodeURIComponent(recipeId)}/revisions`, {
    method: 'POST',
    headers: idempotencyHeaders(key),
    body: JSON.stringify({ expected_revision_id: expectedRevisionId, ...draft }),
  })
}

/** 归档丹方（停止新炼制，不删历史；幂等） */
export function archiveRecipe(key: string, recipeId: string): Promise<PillOperationResult> {
  return request(`/recipes/${encodeURIComponent(recipeId)}/archive`, {
    method: 'POST',
    headers: idempotencyHeaders(key),
    body: JSON.stringify({}),
  })
}

/** 按不可变版本炼制 1 枚金丹入库（归档丹方拒绝；幂等写操作） */
export function craftPill(key: string, recipeId: string, revisionId: string): Promise<PillOperationResult> {
  return request(`/recipes/${encodeURIComponent(recipeId)}/craft`, {
    method: 'POST',
    headers: idempotencyHeaders(key),
    body: JSON.stringify({ revision_id: revisionId }),
  })
}
