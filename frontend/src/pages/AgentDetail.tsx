/**
 * 道人详情页面 - Agent 配置
 * 道人信息编辑 + 已服用金丹列表 + 服用金丹 + 开始对话
 * H5 优化: 纵向排列
 */
import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import {
  ArrowLeft,
  Users,
  Cpu,
  Pill,
  Sparkles,
  Plus,
  Trash2,
  Loader2,
  AlertCircle,
  MessageSquare,
  Edit3,
  X,
  Check,
  FlaskConical,
} from 'lucide-react'
import { useAgent } from '@/contexts/AgentContext'
import { usePill } from '@/contexts/PillContext'
import { useChat } from '@/contexts/ChatContext'
import Layout from '@/components/Layout'
import { mockModels } from '@/services/mockData'
import type { Pill as PillType } from '@/services/types'

export default function AgentDetail() {
  const { id } = useParams<{ id: string }>()
  const agentId = Number(id)

  const { state: agentState, fetchAgent, fetchAgentPills, bindPill, unbindPill, updateAgent } = useAgent()
  const { state: pillState, fetchPills } = usePill()
  const { createSession } = useChat()

  const [showBindPill, setShowBindPill] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [editName, setEditName] = useState('')
  const [editPersonality, setEditPersonality] = useState('')
  const [editModel, setEditModel] = useState('')
  const [isCreatingSession, setIsCreatingSession] = useState(false)

  const agent = agentState.currentAgent
  const agentPills = agentState.currentAgentPills

  // 加载数据
  useEffect(() => {
    if (agentId) {
      fetchAgent(agentId)
      fetchAgentPills(agentId)
      fetchPills()
    }
  }, [agentId, fetchAgent, fetchAgentPills, fetchPills])

  // 初始化编辑表单
  useEffect(() => {
    if (agent) {
      setEditName(agent.name)
      setEditPersonality(agent.personality || '')
      setEditModel(agent.model_name)
    }
  }, [agent])

  /** 保存编辑 */
  const handleSaveEdit = async () => {
    if (!editName.trim()) return
    await updateAgent(agentId, {
      name: editName.trim(),
      personality: editPersonality.trim(),
      model_name: editModel,
    })
    setIsEditing(false)
  }

  /** 服用金丹 */
  const handleBindPill = async (pillId: number) => {
    await bindPill(agentId, pillId)
  }

  /** 解除金丹 */
  const handleUnbindPill = async (pillId: number) => {
    await unbindPill(agentId, pillId)
  }

  /** 开始对话 */
  const handleStartChat = async () => {
    setIsCreatingSession(true)
    await createSession(agentId, `与${agent?.name}的对话`)
    setIsCreatingSession(false)
  }

  /** 获取头像颜色 */
  function getAvatarColor(name: string): string {
    const colors = [
      'from-cinnabar-500 to-cinnabar-700',
      'from-jade-500 to-jade-700',
      'from-gold-500 to-gold-700',
      'from-blue-500 to-blue-700',
    ]
    let hash = 0
    for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
    return colors[Math.abs(hash) % colors.length]
  }

  /** 可服用金丹列表（未绑定的） */
  const availablePills = pillState.pills.filter(
    p => !agentPills.some(ap => ap.id === p.id) && p.status === 'refined'
  )

  if (!agent && agentState.loading) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center py-16">
          <Loader2 className="w-8 h-8 text-gold-400 animate-spin mb-3" />
          <p className="text-sm text-ink-400">加载中...</p>
        </div>
      </Layout>
    )
  }

  if (!agent) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center py-16">
          <AlertCircle className="w-12 h-12 text-cinnabar-400 mb-3" />
          <p className="text-sm text-ink-400">道人不存在</p>
          <Link to="/agents" className="dao-btn-primary mt-4">
            <ArrowLeft className="w-4 h-4" />
            返回道人府
          </Link>
        </div>
      </Layout>
    )
  }

  return (
    <Layout>
      {/* 返回按钮 */}
      <Link
        to="/agents"
        className="inline-flex items-center gap-1.5 text-sm text-ink-400 hover:text-gold-300 transition-colors mb-4"
      >
        <ArrowLeft className="w-4 h-4" />
        返回道人府
      </Link>

      {/* 道人信息头部 */}
      <div className="dao-card p-5 md:p-6 mb-6">
        <div className="flex flex-col md:flex-row gap-5">
          {/* 头像 */}
          <div className={`
            flex-shrink-0 w-20 h-20 md:w-24 md:h-24 rounded-2xl
            bg-gradient-to-br ${getAvatarColor(agent.name)}
            flex items-center justify-center text-white font-serif font-bold text-3xl md:text-4xl
            shadow-lg
          `}>
            {agent.name.charAt(0)}
          </div>

          {/* 信息 */}
          <div className="flex-1 min-w-0">
            {isEditing ? (
              <div className="space-y-3">
                <input
                  value={editName}
                  onChange={e => setEditName(e.target.value)}
                  className="dao-input text-lg font-serif"
                />
                <textarea
                  value={editPersonality}
                  onChange={e => setEditPersonality(e.target.value)}
                  className="dao-textarea"
                  rows={3}
                  placeholder="性格描述..."
                />
                <select
                  value={editModel}
                  onChange={e => setEditModel(e.target.value)}
                  className="dao-input"
                >
                  {mockModels.map(m => (
                    <option key={m.id} value={m.id}>{m.name}</option>
                  ))}
                </select>
                <div className="flex items-center gap-2">
                  <button onClick={handleSaveEdit} className="dao-btn-primary text-sm">
                    <Check className="w-4 h-4" /> 保存
                  </button>
                  <button onClick={() => setIsEditing(false)} className="dao-btn-ghost text-sm">
                    <X className="w-4 h-4" /> 取消
                  </button>
                </div>
              </div>
            ) : (
              <>
                <div className="flex items-center gap-3 mb-2">
                  <h1 className="text-xl md:text-2xl font-serif font-bold text-rice-paper-100">
                    {agent.name}
                  </h1>
                  <span className={`
                    text-[10px] px-2 py-0.5 rounded-full border
                    ${agent.status === 'active'
                      ? 'bg-jade-500/20 text-jade-300 border-jade-500/30'
                      : 'bg-ink-500/30 text-ink-400 border-ink-400/20'
                    }
                  `}>
                    {agent.status === 'active' ? '活跃' : '沉睡'}
                  </span>
                </div>

                {agent.personality && (
                  <p className="text-sm text-ink-400 mb-3 leading-relaxed">
                    {agent.personality}
                  </p>
                )}

                <div className="flex flex-wrap items-center gap-3 text-xs text-ink-400">
                  <span className="flex items-center gap-1">
                    <Cpu className="w-3.5 h-3.5" />
                    {agent.model_name}
                  </span>
                  <span className="flex items-center gap-1">
                    <Pill className="w-3.5 h-3.5" />
                    {agentPills.length} 金丹
                  </span>
                </div>

                <div className="flex items-center gap-2 mt-4">
                  <Link to="/chat" onClick={handleStartChat} className="dao-btn-primary text-sm">
                    {isCreatingSession ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <MessageSquare className="w-4 h-4" />
                    )}
                    开始对话
                  </Link>
                  <button onClick={() => setIsEditing(true)} className="dao-btn-ghost text-sm">
                    <Edit3 className="w-4 h-4" />
                    编辑
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* 已服用金丹 */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-serif font-bold text-gold-300 flex items-center gap-2">
            <FlaskConical className="w-5 h-5" />
            已服用金丹
          </h2>
          <button
            onClick={() => setShowBindPill(!showBindPill)}
            className="dao-btn-gold text-sm"
          >
            <Plus className="w-4 h-4" />
            服用金丹
          </button>
        </div>

        {/* 服用金丹选择面板 */}
        {showBindPill && (
          <div className="dao-card p-4 mb-4 animate-fade-in">
            <h3 className="text-sm font-medium text-gold-300 mb-3">从金丹阁选择</h3>
            {availablePills.length === 0 ? (
              <p className="text-sm text-ink-400 text-center py-4">
                暂无可服用的金丹（所有已成丹的金丹已绑定）
              </p>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {availablePills.map(pill => (
                  <button
                    key={pill.id}
                    onClick={() => handleBindPill(pill.id)}
                    className="flex items-center gap-3 p-3 rounded-lg bg-ink-800/50 border border-bronze-600/20 hover:border-gold-400/40 hover:bg-gold-400/5 transition-all text-left"
                  >
                    <div className="w-8 h-8 rounded-lg bg-gold-500/15 flex items-center justify-center flex-shrink-0">
                      <FlaskConical className="w-4 h-4 text-gold-400" />
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm text-rice-paper-100 truncate">{pill.name}</p>
                      <p className="text-[10px] text-ink-400">{pill.vector_count} 向量</p>
                    </div>
                    <Plus className="w-4 h-4 text-gold-400 ml-auto flex-shrink-0" />
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* 已绑定金丹列表 */}
        {agentPills.length === 0 ? (
          <div className="dao-card flex flex-col items-center py-8 text-center">
            <Pill className="w-10 h-10 text-ink-600 mb-2" />
            <p className="text-sm text-ink-400">尚未服用任何金丹</p>
            <p className="text-xs text-ink-500 mt-1">服用金丹后，道人才能获得对应知识</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {agentPills.map((pill: PillType) => (
              <div
                key={pill.id}
                className="dao-card p-4 flex items-center gap-3"
              >
                <div className="w-10 h-10 rounded-xl bg-gold-500/15 flex items-center justify-center flex-shrink-0">
                  <FlaskConical className="w-5 h-5 text-gold-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <Link
                    to={`/pills/${pill.id}`}
                    className="text-sm font-medium text-rice-paper-100 hover:text-gold-300 transition-colors"
                  >
                    {pill.name}
                  </Link>
                  <p className="text-[10px] text-ink-400">{pill.vector_count} 向量</p>
                </div>
                <button
                  onClick={() => handleUnbindPill(pill.id)}
                  className="p-1.5 rounded hover:bg-cinnabar-500/20 text-ink-400 hover:text-cinnabar-400 transition-colors"
                  title="解除绑定"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </Layout>
  )
}
