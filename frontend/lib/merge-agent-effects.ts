import type { AgentEffect, AgentEffectDraftItem } from '@/services/types'

/**
 * 服用同步专用三方合并（道人编辑页「服用金丹」成功后调用）：
 * - kept:保留 draft 中仍在 fresh 活跃集内的项 —— 用户调过的权重、新增项原样保留；
 * - added:追加 fresh 中「不在同步前基线(previous)」且「未被 kept 覆盖」的项，
 *   即本次服用带来的新能力，按服务端 weight 落草稿；
 * - 被用户待删除(draft 中缺)且 fresh 也没有的项自然消失，不会复活。
 * 幂等:对同一 fresh 重复合并输出不变。
 */
export function mergeConsumedEffects(
  draft: AgentEffectDraftItem[],
  previous: AgentEffect[],
  fresh: AgentEffect[],
): AgentEffectDraftItem[] {
  const previousIds = new Set(previous.map((e) => e.id))
  const kept = draft.filter((item) => fresh.some((e) => e.id === item.effect_id))
  const keptIds = new Set(kept.map((item) => item.effect_id))
  const added = fresh
    .filter((e) => !previousIds.has(e.id) && !keptIds.has(e.id))
    .map((e) => ({ key: e.id, effect_id: e.id, weight: e.weight }))
  return [...kept, ...added]
}
