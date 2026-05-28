/**
 * 金丹阁页面 - 知识库管理
 * 金丹列表卡片（道教丹药风格）
 * 创建金丹按钮 + 表单
 * H5 优化: 单列卡片布局
 */
import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import {
  Plus,
  Search,
  CircleDot,
  Loader2,
  FlaskConical,
  X,
} from 'lucide-react'
import { usePill } from '@/contexts/PillContext'
import PillCard from '@/components/PillCard'
import Layout from '@/components/Layout'

export default function Pills() {
  const { state, fetchPills, addPill } = usePill()
  const [showCreate, setShowCreate] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [searchQuery, setSearchQuery] = useState('')

  // 初始化加载
  useEffect(() => {
    fetchPills()
  }, [fetchPills])

  /** 创建金丹 */
  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    await addPill(name.trim(), description.trim() || undefined)
    setShowCreate(false)
    setName('')
    setDescription('')
  }

  /** 过滤金丹 */
  const filteredPills = state.pills.filter(pill =>
    pill.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (pill.description?.toLowerCase() || '').includes(searchQuery.toLowerCase())
  )

  return (
    <Layout>
      {/* 页面头部 */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <div className="flex items-center gap-3">
            <CircleDot className="w-6 h-6 text-gold-400" />
            <h1 className="page-title">金丹阁</h1>
          </div>
          <p className="page-subtitle">管理你的知识金丹</p>
        </div>

        <button
          onClick={() => setShowCreate(true)}
          className="dao-btn-primary self-start"
        >
          <Plus className="w-4 h-4" />
          炼制新金丹
        </button>
      </div>

      {/* 搜索栏 */}
      <div className="relative mb-6">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-400" />
        <input
          type="text"
          placeholder="搜索金丹..."
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          className="dao-input pl-10"
        />
      </div>

      {/* 创建金丹弹窗 */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="dao-card w-full max-w-md p-6 animate-fade-in">
            <div className="flex items-center justify-between mb-5">
              <div className="flex items-center gap-2">
                <FlaskConical className="w-5 h-5 text-cinnabar-400" />
                <h2 className="text-lg font-serif font-bold text-gold-300">炼制新金丹</h2>
              </div>
              <button
                onClick={() => setShowCreate(false)}
                className="p-1.5 rounded-lg hover:bg-ink-700 text-ink-400 hover:text-rice-paper-100 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="dao-label">金丹名称 *</label>
                <input
                  type="text"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="如：九转还魂丹"
                  className="dao-input"
                  autoFocus
                  required
                />
              </div>

              <div>
                <label className="dao-label">描述</label>
                <textarea
                  value={description}
                  onChange={e => setDescription(e.target.value)}
                  placeholder="描述这颗金丹的用途..."
                  className="dao-textarea"
                  rows={3}
                />
              </div>

              <div className="flex items-center gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreate(false)}
                  className="dao-btn-ghost flex-1"
                >
                  取消
                </button>
                <button
                  type="submit"
                  disabled={!name.trim() || state.loading}
                  className="dao-btn-primary flex-1 disabled:opacity-50"
                >
                  {state.loading ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <FlaskConical className="w-4 h-4" />
                  )}
                  开始炼制
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* 加载状态 */}
      {state.loading && state.pills.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16">
          <Loader2 className="w-8 h-8 text-gold-400 animate-spin mb-3" />
          <p className="text-sm text-ink-400">正在搜寻金丹...</p>
        </div>
      )}

      {/* 空状态 */}
      {!state.loading && filteredPills.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <CircleDot className="w-12 h-12 text-ink-600 mb-3" />
          <h3 className="text-base font-medium text-ink-400 mb-1">
            {searchQuery ? '未找到匹配的金丹' : '暂无金丹'}
          </h3>
          <p className="text-sm text-ink-500 mb-4">
            {searchQuery ? '尝试其他关键词' : '点击上方按钮开始炼制你的第一颗金丹'}
          </p>
          {!searchQuery && (
            <button onClick={() => setShowCreate(true)} className="dao-btn-primary">
              <Plus className="w-4 h-4" />
              炼制新金丹
            </button>
          )}
        </div>
      )}

      {/* 金丹列表 - 桌面端网格，H5 单列 */}
      {filteredPills.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredPills.map(pill => (
            <PillCard key={pill.id} pill={pill} />
          ))}
        </div>
      )}
    </Layout>
  )
}
