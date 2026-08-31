/**
 * 金丹库存实例状态机（金丹消耗品重构任务 6）。
 *
 * 状态只允许 available 单向流转：
 *   available → consumed_by_agent（服用，能力保留）
 *   available → consumed_by_fusion（融合材料消耗）
 *   available → discarded（弃置，显式确认）
 * 终态不能回到 available。操作前必须调用 canUsePill 校验，
 * 任何依赖"无限服用内置定义"的旧 UI 逻辑都必须改走真实库存。
 */

export type PillItemState =
  | 'available'
  | 'consumed_by_agent'
  | 'consumed_by_fusion'
  | 'discarded'

/** 该实例是否仍可使用（服用/融合/弃置都只能作用于 available） */
export function canUsePill(state: PillItemState): boolean {
  return state === 'available'
}

/** 已消耗/已弃置的状态是否为终态 */
export function isTerminalPillState(state: PillItemState): boolean {
  return state !== 'available'
}
