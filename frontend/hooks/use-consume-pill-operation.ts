'use client'

/**
 * 共享服用执行器（金丹消耗品重构任务 6 Phase E 复用层）
 * 旧 ConsumePillModal（选道人）与道人编辑页「服用金丹」共用同一套提交/断线恢复/
 * 幂等语义，本 hook 把「执行」与「UI」解耦：
 * - submit：以 ConsumeInput 发起服用，幂等 key 由 pending-operations 按
 *   action+target 生成/复用；双击与 committed 后拒绝重复 POST
 * - 失败（网络层）：先 GET pill-operations/:id 恢复；命中且校验通过按成功处理；
 *   404/结果不符展示错误（保留 pending，同 key 重试，不自动换 key）
 * - recover 查询本身失败：结果未知（uncertain），保留 pending 与同 key，
 *   重试可安全收敛（服务端幂等去重）
 * - 成功检查：operation_id===key && effect_id 存在 && consumed_item_ids 含 itemId
 *   （后端 idempotencyKey 回显，见 pill_inventory/consume_effects.go）
 */
import { useCallback, useRef, useState } from 'react'

import {
  clearPendingOperation,
  consumeTarget,
  recoverOperation,
  startPendingOperation,
  type ConsumeInput,
} from '@/lib/pending-operations'
import { consumePill } from '@/services/pillInventoryService'
import type { PillOperationResult } from '@/services/types'

export type ConsumeStatus = 'idle' | 'submitting' | 'committed' | 'error' | 'uncertain'

export interface ConsumePillOperation {
  status: ConsumeStatus
  /** 最近一次错误信息（uncertain 时为 null，UI 用自己的文案） */
  error: string | null
  /** 发起/重试服用（同 target 期间复用幂等 key） */
  submit(input: ConsumeInput): Promise<void>
  /** 用最近一次输入原样重试（沿用同幂等 key） */
  retry(): Promise<void>
}

/**
 * 成功检查：operation_id 必须等于幂等 key（后端 idempotencyKey 回显），
 * 且结果确实包含本次服用产生的能力快照与实例消耗；否则视为未提交。
 */
export function isConsumedResult(
  result: PillOperationResult,
  key: string,
  input: ConsumeInput,
): boolean {
  return (
    result.operation_id === key &&
    !!result.effect_id &&
    (result.consumed_item_ids ?? []).includes(input.itemId)
  )
}

export function useConsumePillOperation(options?: {
  /** 提交成功后回调（服用页面用它同步库存/能力；result 为幂等操作的终局响应） */
  onCommitted?: (input: ConsumeInput, result: PillOperationResult) => void | Promise<void>
}): ConsumePillOperation {
  const onCommitted = options?.onCommitted
  const [status, setStatus] = useState<ConsumeStatus>('idle')
  const [error, setError] = useState<string | null>(null)
  /** 提交中去重（双击/重复触发只产生一次写请求） */
  const submittingRef = useRef(false)
  /** 已提交后拒绝一切再次 POST（含换目标的新输入） */
  const committedRef = useRef(false)
  const lastInputRef = useRef<ConsumeInput | null>(null)

  const run = useCallback(
    async (incoming: ConsumeInput) => {
      if (committedRef.current || submittingRef.current) return
      submittingRef.current = true
      lastInputRef.current = incoming
      setStatus('submitting')
      setError(null)
      const key = startPendingOperation(
        'consume',
        incoming.label ?? `${incoming.agentId}→${incoming.itemId}`,
        consumeTarget(incoming),
        incoming,
      )
      try {
        const result = await consumePill(key, incoming.agentId, incoming.itemId, {
          weight: incoming.weight,
          sortOrder: incoming.sortOrder,
        })
        if (!isConsumedResult(result, key, incoming)) {
          // 响应异常：走恢复查询自愈（同 key 已提交则按成功收敛）
          throw new Error('consume result mismatch')
        }
        clearPendingOperation(key)
        committedRef.current = true
        setStatus('committed')
        await onCommitted?.(incoming, result)
      } catch (err) {
        try {
          const recovered = await recoverOperation(key)
          if (recovered && isConsumedResult(recovered, key, incoming)) {
            // 断线恢复命中：服务端已提交，按成功处理（不重复服用）
            clearPendingOperation(key)
            committedRef.current = true
            setStatus('committed')
            await onCommitted?.(incoming, recovered)
            return
          }
          // 404（未提交）或结果与本次输入不符：保留 pending，同 key 重试
          setStatus('error')
          setError(err instanceof Error ? err.message : String(err))
        } catch {
          // 恢复查询本身失败：结果未知；保留 pending，同 key 重试可安全收敛
          setStatus('uncertain')
          setError(null)
        }
      } finally {
        submittingRef.current = false
      }
    },
    [onCommitted],
  )

  const submit = useCallback((incoming: ConsumeInput) => run(incoming), [run])

  const retry = useCallback(() => {
    if (!lastInputRef.current) return Promise.resolve()
    return run(lastInputRef.current)
  }, [run])

  return { status, error, submit, retry }
}
