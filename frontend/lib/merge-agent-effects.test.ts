import { describe, expect, it } from 'vitest'

import { mergeConsumedEffects } from './merge-agent-effects'
import type { AgentEffect } from '@/services/types'

/** 本地 fixture:满足 AgentEffect 全字段(与 use-agent-editor-flow 测试同构,不依赖真实库) */
const eff = (id: string, weight = 1): AgentEffect => ({
  id,
  name: `能力${id}`,
  schema: {},
  weight,
  sort_order: 1,
  item_id: `item-${id}`,
  revision_id: `rev-${id}`,
  created_at: '2026-08-20T00:00:00Z',
})
const a = eff('a')
const b = eff('b', 1)
const c = eff('c', 3)

describe('mergeConsumedEffects 三方合并(服用同步专用)', () => {
  it('核心:用户待删除项不复活,新能力追加末尾,保留用户权重', () => {
    expect(
      mergeConsumedEffects(
        [{ key: b.id, effect_id: b.id, weight: 2 }], // 草稿:仅保留 B(用户权重 2),A 待删除
        [a, b], // 同步前编辑器认知基线
        [a, b, c], // 服用后的服务端活跃集
      ),
    ).toEqual([
      { key: b.id, effect_id: b.id, weight: 2 },
      { key: c.id, effect_id: c.id, weight: c.weight },
    ])
  })

  it('并发移除:服务端已无 B,输出不含 B', () => {
    expect(
      mergeConsumedEffects([{ key: b.id, effect_id: b.id, weight: 2 }], [a, b], [a, c]),
    ).toEqual([{ key: c.id, effect_id: c.id, weight: c.weight }])
  })

  it('重复同步不重复追加:对同一 fresh 二次合并输出不变', () => {
    const first = mergeConsumedEffects([{ key: b.id, effect_id: b.id, weight: 2 }], [a, b], [a, b, c])
    const second = mergeConsumedEffects(first, [a, b, c], [a, b, c])
    expect(second).toEqual(first)
  })

  it('previous 为空(首次同步):全部 fresh 追加', () => {
    expect(mergeConsumedEffects([], [], [a, c])).toEqual([
      { key: a.id, effect_id: a.id, weight: a.weight },
      { key: c.id, effect_id: c.id, weight: c.weight },
    ])
  })
})
