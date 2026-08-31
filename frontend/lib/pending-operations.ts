/**
 * 桌面会话级 pending 操作记录（金丹消耗品重构任务 6）
 *
 * 每个明确用户动作生成一个 UUID 幂等 key，写入当前会话记录；网络重试、
 * 窗口隐藏再显示使用原 key；操作完成后清除，用户再次点击才创建新 key。
 * 同动作同目标 pending 期间复用同一 key（防重复点击产生多份逻辑操作）。
 *
 * 断线恢复契约：先 GET /api/v1/pill-operations/:id；404 仅说明没有已提交
 * 结果，仍可用同 key 重试，不能自动换 key。
 *
 * 记录只存恢复所需 ID/动作/目标名，绝不存 LLM 密钥或请求体。
 */
import type { PillOperationResult } from '@/services/types'
import { getOperation } from '@/services/pillInventoryService'

/** 写操作动作种类（与后端幂等键语义一一对应） */
export type PendingAction =
  | 'craft'
  | 'save_recipe'
  | 'update_recipe'
  | 'archive_recipe'
  | 'consume'
  | 'discard'
  | 'remove_effect'
  | 'confirm_fusion'

/** 恢复所需的最小记录（不存密钥/请求体） */
export interface PendingOperationRecord {
  key: string
  action: PendingAction
  /** 目标展示名（断线恢复时提示用） */
  label: string
  createdAt: number
}

const STORAGE_KEY = 'alchemy_pending_operations'

/** sessionStorage 读写包装：不可用时静默降级为内存（SSR/隐私模式） */
function readAll(): PendingOperationRecord[] {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? (parsed as PendingOperationRecord[]) : []
  } catch {
    return []
  }
}

function writeAll(records: PendingOperationRecord[]): void {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(records))
  } catch {
    // 写失败不阻断业务：key 仍在内存调用方手中，可继续重试
  }
}

/**
 * 为明确用户动作生成/复用幂等 key 并记录。
 * 同 action+label 已有 pending 记录时返回原 key（重复点击只产生一项逻辑操作）。
 */
export function startPendingOperation(action: PendingAction, label: string): string {
  const existing = readAll().find((r) => r.action === action && r.label === label)
  if (existing) return existing.key
  const record: PendingOperationRecord = {
    key: crypto.randomUUID(),
    action,
    label,
    createdAt: Date.now(),
  }
  writeAll([...readAll(), record])
  return record.key
}

/** 读 pending 记录（断线恢复时查它决定用原 key 重试） */
export function getPendingOperation(key: string): PendingOperationRecord | null {
  return readAll().find((r) => r.key === key) ?? null
}

/** 操作成功后清除记录；之后再触发同动作才创建新 key */
export function clearPendingOperation(key: string): void {
  writeAll(readAll().filter((r) => r.key !== key))
}

/**
 * 断线恢复：网络失败后先查已提交结果。
 * 已提交 → 返回结果（调用方按成功处理并清除记录）；
 * 404/未提交 → 返回 null，调用方仍用原 key 重试（不自动换 key）。
 */
export async function recoverOperation(key: string): Promise<PillOperationResult | null> {
  try {
    return await getOperation(key)
  } catch (error) {
    if (error instanceof Error && 'status' in error && (error as { status: number }).status === 404) {
      return null
    }
    throw error
  }
}
