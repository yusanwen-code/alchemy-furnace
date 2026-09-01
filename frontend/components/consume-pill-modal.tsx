'use client'

/**
 * 服用金丹对话框（金丹消耗品重构任务 6 Phase E）
 * 选择一位道人，消耗 1 枚库存实例（available→consumed_by_agent）并生成能力快照。
 * - 提交成功才移除库存并更新能力；失败保持原状（对话框保留可重试）。
 * - Idempotency-Key 由 pending-operations 按「动作+目标」生成/复用：网络重试、
 *   窗口隐藏再显示沿用原 key；成功后清除，换道人即新动作。
 * - 断线恢复先 GET pill-operations/:id；404 说明未提交，仍用原 key 重试，不换 key。
 * 调用 POST /api/v1/agents/:id/consume
 */
import { useState, useEffect } from 'react'
import { useTranslations } from 'next-intl'
import { X, Users, Loader2, FlaskConical, AlertCircle } from 'lucide-react'
import { EntityAvatar } from '@/components/avatar/entity-avatar'
import { useConsumePillOperation } from '@/hooks/use-consume-pill-operation'
import { listAgents } from '@/services/agentService'
import type { Agent } from '@/services/types'

interface ConsumePillModalProps {
  /** 库存实例 UUID（不是丹方 recipeId / 能力 effectId） */
  itemId: string
  /** 实例展示名（标题与幂等 label 用） */
  itemName: string
  onClose: () => void
  /** 服用成功后回调（父组件刷新库存/能力缓存） */
  onConsumed?: () => void
}

export function ConsumePillModal({ itemId, itemName, onClose, onConsumed }: ConsumePillModalProps) {
  const t = useTranslations('consumeDialog')
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null)
  const [weight, setWeight] = useState(1)
  const [sortOrder, setSortOrder] = useState(0)
  /** 道人列表加载失败（与服用结果错误分开展示；关弹窗重开即重置） */
  const [loadError, setLoadError] = useState<string | null>(null)

  // 共享服用执行器：提交/断线恢复/幂等 key 语义与道人编辑页「服用金丹」一致
  const { status, error, submit } = useConsumePillOperation({
    onCommitted: () => {
      onConsumed?.()
      setTimeout(onClose, 800)
    },
  })
  const submitting = status === 'submitting'
  const success = status === 'committed'
  // 结果未知（recover 查询失败）时展示通用失败文案，重试同 key 可安全收敛
  const errorText = error ?? (status === 'uncertain' ? t('errorConsume') : null)

  // 加载道人列表
  useEffect(() => {
    let cancelled = false
    listAgents({ page_size: 100 })
      .then(data => {
        if (!cancelled) setAgents(data.list || [])
      })
      .catch(err => {
        if (!cancelled) setLoadError(err instanceof Error ? err.message : t('errorLoadAgents'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [t])

  /** 执行服用（幂等）：成功才回调/关闭，失败保留对话框原状；重试沿用同一幂等 key */
  const handleConsume = () => {
    if (!selectedAgentId || submitting || success) return
    const agent = agents.find(a => a.id === selectedAgentId)
    void submit({
      agentId: selectedAgentId,
      itemId,
      weight,
      sortOrder,
      label: `${itemName}→${agent?.name ?? itemName}`,
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/30 backdrop-blur-sm">
      <div className="dao-card w-full max-w-md p-6 animate-in fade-in duration-300 max-h-[80vh] overflow-y-auto">
        <div className="flex items-center justify-between gap-2 mb-5">
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <FlaskConical className="w-5 h-5 text-gold shrink-0" />
            <h2 className="text-lg font-serif font-bold text-foreground truncate">
              {t('title')}
            </h2>
          </div>
          <button
            aria-label={t('closeModal')}
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors shrink-0"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <p className="text-xs text-muted-foreground mb-4 leading-relaxed">
          {t('prompt')}
        </p>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-6 h-6 text-gold animate-spin" />
          </div>
        ) : agents.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-6">
            {t('noAgents')}
          </p>
        ) : (
          <div className="space-y-2 mb-4 max-h-60 overflow-y-auto">
            {agents.map(agent => (
              <button
                key={agent.id}
                onClick={() => setSelectedAgentId(agent.id)}
                className={`
                  w-full flex items-center gap-3 p-3 rounded-xl border transition-all text-left min-w-0
                  ${selectedAgentId === agent.id
                    ? 'bg-gold/10 border-gold/40'
                    : 'bg-secondary/70 border-border/70 hover:border-gold/30'
                  }
                `}
              >
                <EntityAvatar name={agent.name} src={agent.avatar} size="sm" shape="square" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground truncate">{agent.name}</p>
                  <p className="text-[10px] text-muted-foreground truncate">{agent.model_name}</p>
                </div>
                {selectedAgentId === agent.id && (
                  <span className="w-2 h-2 rounded-full bg-gold shrink-0" />
                )}
              </button>
            ))}
          </div>
        )}

        {agents.length > 0 && (
          <div className="grid grid-cols-2 gap-3 mb-4">
            <div className="min-w-0">
              <label className="dao-label">{t('weightLabel')}</label>
              <input
                type="number"
                min={0}
                max={10}
                step={0.5}
                value={weight}
                onChange={e => setWeight(Math.min(10, Math.max(0, Number(e.target.value))))}
                className="dao-input"
              />
            </div>
            <div className="min-w-0">
              <label className="dao-label">{t('sortOrder')}</label>
              <input
                type="number"
                min={0}
                step={1}
                value={sortOrder}
                onChange={e => setSortOrder(Math.max(0, Math.floor(Number(e.target.value))))}
                className="dao-input"
              />
            </div>
          </div>
        )}

        {(errorText ?? loadError) && (
          <div className="flex items-center gap-2 text-xs text-primary mb-3" role="alert">
            <AlertCircle className="w-3.5 h-3.5 shrink-0" />
            <span>{errorText ?? loadError}</span>
          </div>
        )}

        <div className="flex items-center gap-3 flex-wrap">
          <button onClick={onClose} className="dao-btn-ghost flex-1 whitespace-nowrap">
            {t('cancel')}
          </button>
          <button
            onClick={handleConsume}
            disabled={!selectedAgentId || submitting || success || agents.length === 0}
            className="dao-btn-primary flex-1 disabled:opacity-50 whitespace-nowrap"
          >
            {submitting ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : success ? (
              <Users className="w-4 h-4" />
            ) : (
              <FlaskConical className="w-4 h-4" />
            )}
            {submitting ? t('submitting') : success ? t('success') : t('submit')}
          </button>
        </div>
      </div>
    </div>
  )
}
