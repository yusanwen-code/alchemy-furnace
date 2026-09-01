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

/** 服用动作输入快照（label 仅作展示，不参与幂等签名） */
export interface ConsumeInput {
  agentId: string
  itemId: string
  weight: number
  sortOrder?: number
  /** pending 记录展示名（断线恢复提示用；不参与签名） */
  label?: string
}

/** 恢复所需的最小记录（不存密钥/请求体） */
export interface PendingOperationRecord {
  key: string
  action: PendingAction
  /** 目标展示名（断线恢复时提示用） */
  label: string
  createdAt: number
  /** 幂等目标签名（consume:JSON.stringify([agentId,itemId,weight,sortOrder??0])）；无 target 的动作不存 */
  target?: string
  /** 服用动作输入快照（只存 UUID/权重/顺序，供恢复重试；非 consume 动作不存） */
  consumeInput?: ConsumeInput
}

const STORAGE_KEY = 'alchemy_pending_operations'

/**
 * 服用幂等目标签名：同 action+target 视为同一逻辑操作；
 * 参数变化（换道人/金丹/权重/顺序）即新动作、新 key。
 */
export function consumeTarget(input: ConsumeInput): string {
  return JSON.stringify([input.agentId, input.itemId, input.weight, input.sortOrder ?? 0])
}

/**
 * 模块级内存后备：sessionStorage 不可用/被清空时仍保证会话内幂等语义
 * （SSR、隐私模式、桌面 webview 存储被禁）。
 */
let memoryStore: PendingOperationRecord[] = []

/** sessionStorage 读写包装：不可用时静默降级为内存（SSR/隐私模式） */
function readAll(): PendingOperationRecord[] {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (raw === null) return memoryStore
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? (parsed as PendingOperationRecord[]) : memoryStore
  } catch {
    return memoryStore
  }
}

function writeAll(records: PendingOperationRecord[]): void {
  memoryStore = records
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(records))
  } catch {
    // sessionStorage 不可用：内存后备已接管，不阻断业务
  }
}

/** 快照化：只保留 UUID/权重/顺序，剥离 label 等展示字段（记录不存名称/密钥/请求体） */
function toConsumeSnapshot(input: ConsumeInput): ConsumeInput {
  return {
    agentId: input.agentId,
    itemId: input.itemId,
    weight: input.weight,
    ...(input.sortOrder !== undefined ? { sortOrder: input.sortOrder } : {}),
  }
}

/**
 * 为明确用户动作生成/复用幂等 key 并记录。
 * - 无 target（非 consume 动作）：同 action+label 已有 pending 记录时返回原 key；
 * - 有 target（consume 动作）：同 action+target 复用 key（参数签名见 consumeTarget）。
 */
export function startPendingOperation(
  action: PendingAction,
  label: string,
  target?: string,
  consumeInput?: ConsumeInput,
): string {
  const records = readAll()
  const existing =
    target !== undefined
      ? records.find((r) => r.action === action && r.target === target)
      : records.find((r) => r.action === action && r.label === label)
  if (existing) return existing.key
  const record: PendingOperationRecord = {
    key: crypto.randomUUID(),
    action,
    label,
    createdAt: Date.now(),
    ...(target !== undefined ? { target } : {}),
    ...(consumeInput !== undefined ? { consumeInput: toConsumeSnapshot(consumeInput) } : {}),
  }
  writeAll([...records, record])
  return record.key
}

/** 当前全部 pending 记录（服用中/结果未知提示、窗口隐藏再显示用） */
export function listPendingOperations(): PendingOperationRecord[] {
  return readAll()
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
