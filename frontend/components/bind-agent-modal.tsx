'use client'

/**
 * 从金丹到道人 - 快捷绑定弹窗
 * 选择一位道人，将金丹绑定给它（可设置权重与服用顺序）
 * 调用 POST /api/v1/agents/:id/pills
 */
import { useState, useEffect } from 'react'
import { X, Users, Loader2, Gift, AlertCircle } from 'lucide-react'
import * as agentService from '@/services/agentService'
import type { Agent, Pill } from '@/services/types'

interface BindAgentModalProps {
  pill: Pill
  onClose: () => void
}

export function BindAgentModal({ pill, onClose }: BindAgentModalProps) {
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [selectedAgentId, setSelectedAgentId] = useState<number | null>(null)
  const [weight, setWeight] = useState(1)
  const [sortOrder, setSortOrder] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  // 加载道人列表
  useEffect(() => {
    let cancelled = false
    agentService.listAgents({ page_size: 100 })
      .then(data => {
        if (!cancelled) setAgents(data.list || [])
      })
      .catch(err => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取道人列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [])

  /** 执行绑定 */
  const handleBind = async () => {
    if (!selectedAgentId) return
    setSubmitting(true)
    setError(null)
    try {
      await agentService.bindPill(selectedAgentId, pill.id, weight, sortOrder)
      setSuccess(true)
      setTimeout(onClose, 800)
    } catch (err) {
      setError(err instanceof Error ? err.message : '绑定失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/30 backdrop-blur-sm">
      <div className="dao-card w-full max-w-md p-6 animate-in fade-in duration-300 max-h-[80vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2">
            <Gift className="w-5 h-5 text-gold" />
            <h2 className="text-lg font-serif font-bold text-foreground">赠予道人</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <p className="text-xs text-muted-foreground mb-4">
          将「<span className="text-gold">{pill.name}</span>」赠予一位道人服用
        </p>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-6 h-6 text-gold animate-spin" />
          </div>
        ) : agents.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-6">
            暂无道人，请先在道人府招募一位
          </p>
        ) : (
          <div className="space-y-2 mb-4 max-h-60 overflow-y-auto">
            {agents.map(agent => (
              <button
                key={agent.id}
                onClick={() => setSelectedAgentId(agent.id)}
                className={`
                  w-full flex items-center gap-3 p-3 rounded-xl border transition-all text-left
                  ${selectedAgentId === agent.id
                    ? 'bg-gold/10 border-gold/40'
                    : 'bg-secondary/70 border-border/70 hover:border-gold/30'
                  }
                `}
              >
                <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-sage to-sage/70 flex items-center justify-center text-primary-foreground font-serif font-bold flex-shrink-0">
                  {agent.name.charAt(0)}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-foreground">{agent.name}</p>
                  <p className="text-[10px] text-muted-foreground truncate">{agent.model_name}</p>
                </div>
                {selectedAgentId === agent.id && (
                  <span className="w-2 h-2 rounded-full bg-gold flex-shrink-0" />
                )}
              </button>
            ))}
          </div>
        )}

        {agents.length > 0 && (
          <div className="grid grid-cols-2 gap-3 mb-4">
            <div>
              <label className="dao-label">权重（0-10）</label>
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
            <div>
              <label className="dao-label">服用顺序</label>
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

        {error && (
          <div className="flex items-center gap-2 text-xs text-primary mb-3">
            <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <div className="flex items-center gap-3">
          <button onClick={onClose} className="dao-btn-ghost flex-1">
            取消
          </button>
          <button
            onClick={handleBind}
            disabled={!selectedAgentId || submitting || success || agents.length === 0}
            className="dao-btn-primary flex-1 disabled:opacity-50"
          >
            {submitting ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : success ? (
              <Users className="w-4 h-4" />
            ) : (
              <Gift className="w-4 h-4" />
            )}
            {success ? '已赠予' : '确认赠予'}
          </button>
        </div>
      </div>
    </div>
  )
}
