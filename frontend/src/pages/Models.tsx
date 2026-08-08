/**
 * 模型管理页面 - LLM 模型配置
 * 模型列表表格（名称/显示名/服务商/密钥/温度/默认/合成/启用/引用数/操作）
 * 创建/编辑弹窗、连接测试（显示延迟或中文错误）、删除（409 引用保护）
 * API Key 仅用于写入，编辑时以掩码占位，留空表示不修改
 */
import { useState, useEffect, useCallback } from 'react'
import {
  Cpu,
  Plus,
  Loader2,
  X,
  Pencil,
  Trash2,
  Zap,
  Check,
  AlertCircle,
  Star,
  Wand2,
} from 'lucide-react'
import Layout from '@/components/Layout'
import * as modelService from '@/services/modelService'
import { ApiError } from '@/services/api'
import type { LLMModel, ModelProvider, CreateModelRequest } from '@/services/modelService'

/** 服务商选项与展示名 */
const PROVIDERS: { value: ModelProvider; label: string }[] = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'deepseek', label: 'DeepSeek' },
  { value: 'aliyun', label: '阿里云（通义）' },
  { value: 'ollama', label: 'Ollama（本地）' },
  { value: 'other', label: '其他' },
]

/** 服务商徽标颜色 */
const PROVIDER_BADGE: Record<string, string> = {
  openai: 'bg-blue-500/10 text-blue-400 border-blue-500/20',
  deepseek: 'bg-purple-500/10 text-purple-400 border-purple-500/20',
  aliyun: 'bg-orange-500/10 text-orange-400 border-orange-500/20',
  ollama: 'bg-jade-500/10 text-jade-400 border-jade-500/20',
  other: 'bg-ink-500/30 text-ink-300 border-ink-400/20',
}

/** 表单状态 */
interface ModelForm {
  name: string
  display_name: string
  provider: ModelProvider
  base_url: string
  api_key: string
  temperature: number
  max_tokens: number
  is_enabled: boolean
  is_default: boolean
  is_synthesis: boolean
  sort_order: number
}

const EMPTY_FORM: ModelForm = {
  name: '',
  display_name: '',
  provider: 'openai',
  base_url: '',
  api_key: '',
  temperature: 0.7,
  max_tokens: 4096,
  is_enabled: true,
  is_default: false,
  is_synthesis: false,
  sort_order: 0,
}

/** 单行连接测试状态 */
interface TestState {
  loading: boolean
  result: { success: boolean; latency_ms: number; error: string } | null
}

export default function Models() {
  const [models, setModels] = useState<LLMModel[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // 弹窗表单
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<LLMModel | null>(null)
  const [form, setForm] = useState<ModelForm>(EMPTY_FORM)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  // 删除确认
  const [deleting, setDeleting] = useState<LLMModel | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  // 每行连接测试状态
  const [tests, setTests] = useState<Record<number, TestState>>({})

  /** 加载模型列表 */
  const fetchModels = useCallback(async () => {
    setLoading(true)
    try {
      const data = await modelService.list({ page_size: 100 })
      setModels(data.list || [])
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : '获取模型列表失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchModels()
  }, [fetchModels])

  /** 打开创建弹窗 */
  const openCreate = () => {
    setEditing(null)
    setForm(EMPTY_FORM)
    setFormError(null)
    setShowForm(true)
  }

  /** 打开编辑弹窗（api_key 留空 = 不修改，占位符展示掩码） */
  const openEdit = (model: LLMModel) => {
    setEditing(model)
    setForm({
      name: model.name,
      display_name: model.display_name,
      provider: (PROVIDERS.some(p => p.value === model.provider) ? model.provider : 'other') as ModelProvider,
      base_url: model.base_url || '',
      api_key: '',
      temperature: model.temperature,
      max_tokens: model.max_tokens,
      is_enabled: model.is_enabled,
      is_default: model.is_default,
      is_synthesis: model.is_synthesis,
      sort_order: model.sort_order,
    })
    setFormError(null)
    setShowForm(true)
  }

  /** 提交创建/编辑 */
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.name.trim()) return
    setSaving(true)
    setFormError(null)

    const payload: CreateModelRequest = {
      name: form.name.trim(),
      display_name: form.display_name.trim() || undefined,
      provider: form.provider,
      base_url: form.base_url.trim() || undefined,
      temperature: form.temperature,
      max_tokens: form.max_tokens,
      is_enabled: form.is_enabled,
      is_default: form.is_default,
      is_synthesis: form.is_synthesis,
      sort_order: form.sort_order,
    }
    // api_key 仅在填写时提交：创建时写入，编辑时留空表示不修改
    if (form.api_key) {
      payload.api_key = form.api_key
    }

    try {
      if (editing) {
        await modelService.update(editing.id, payload)
      } else {
        await modelService.create(payload)
      }
      setShowForm(false)
      setEditing(null)
      await fetchModels()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  /** 删除模型（409：仍被道人引用） */
  const handleDelete = async () => {
    if (!deleting) return
    setDeleteLoading(true)
    try {
      await modelService.remove(deleting.id)
      setDeleting(null)
      setError(null)
      await fetchModels()
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        const referencedBy = (err.data?.data as { referenced_by?: number } | undefined)?.referenced_by
        setError(
          err.message ||
          `该模型仍被 ${referencedBy ?? deleting.referenced_by} 个道人引用，无法删除`
        )
      } else {
        setError(err instanceof Error ? err.message : '删除失败')
      }
      setDeleting(null)
    } finally {
      setDeleteLoading(false)
    }
  }

  /** 测试连接 */
  const handleTest = async (model: LLMModel) => {
    setTests(prev => ({ ...prev, [model.id]: { loading: true, result: null } }))
    try {
      const result = await modelService.testConnection(model.id)
      setTests(prev => ({ ...prev, [model.id]: { loading: false, result } }))
    } catch (err) {
      setTests(prev => ({
        ...prev,
        [model.id]: {
          loading: false,
          result: {
            success: false,
            latency_ms: 0,
            error: err instanceof Error ? err.message : '连接测试失败',
          },
        },
      }))
    }
  }

  return (
    <Layout>
      {/* 页面头部 */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <div>
          <div className="flex items-center gap-3">
            <Cpu className="w-6 h-6 text-gold-400" />
            <h1 className="page-title">模型管理</h1>
          </div>
          <p className="page-subtitle">配置论道与丹性合成所用的语言模型</p>
        </div>

        <button onClick={openCreate} className="dao-btn-primary self-start">
          <Plus className="w-4 h-4" />
          新增模型
        </button>
      </div>

      {/* 加载状态 */}
      {loading && models.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16">
          <Loader2 className="w-8 h-8 text-gold-400 animate-spin mb-3" />
          <p className="text-sm text-ink-400">正在加载模型...</p>
        </div>
      )}

      {/* 空状态 */}
      {!loading && models.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Cpu className="w-12 h-12 text-ink-600 mb-3" />
          <h3 className="text-base font-medium text-ink-400 mb-1">暂无模型</h3>
          <p className="text-sm text-ink-500 mb-4">点击上方按钮添加第一个语言模型</p>
          <button onClick={openCreate} className="dao-btn-primary">
            <Plus className="w-4 h-4" />
            新增模型
          </button>
        </div>
      )}

      {/* 模型表格 */}
      {models.length > 0 && (
        <div className="dao-card overflow-x-auto">
          <table className="w-full text-sm min-w-[900px]">
            <thead>
              <tr className="border-b border-bronze-600/20 text-left">
                <th className="px-4 py-3 text-xs font-medium text-ink-400">名称</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400">显示名</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400">服务商</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400">API Key</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400">温度</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400 text-center">默认</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400 text-center">合成</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400 text-center">启用</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400 text-center">引用数</th>
                <th className="px-4 py-3 text-xs font-medium text-ink-400 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              {models.map(model => {
                const test = tests[model.id]
                return (
                  <tr
                    key={model.id}
                    className="border-b border-bronze-600/10 last:border-0 hover:bg-gold-400/5 transition-colors"
                  >
                    <td className="px-4 py-3 text-rice-paper-100 font-medium whitespace-nowrap">
                      {model.name}
                    </td>
                    <td className="px-4 py-3 text-ink-300 whitespace-nowrap">
                      {model.display_name || '-'}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`
                        text-[10px] px-2 py-0.5 rounded-full border whitespace-nowrap
                        ${PROVIDER_BADGE[model.provider] || PROVIDER_BADGE.other}
                      `}>
                        {model.provider}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-ink-400 font-mono text-xs whitespace-nowrap">
                      {model.has_api_key ? model.api_key_masked || '已配置' : '未配置'}
                    </td>
                    <td className="px-4 py-3 text-ink-300">{model.temperature}</td>
                    <td className="px-4 py-3 text-center">
                      {model.is_default && <Star className="w-4 h-4 text-gold-400 inline" />}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {model.is_synthesis && <Wand2 className="w-4 h-4 text-jade-400 inline" />}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span className={`
                        inline-block w-2 h-2 rounded-full
                        ${model.is_enabled ? 'bg-jade-400' : 'bg-ink-500'}
                      `} />
                    </td>
                    <td className="px-4 py-3 text-center text-ink-300">{model.referenced_by}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-1">
                        {/* 连接测试 */}
                        <button
                          onClick={() => handleTest(model)}
                          disabled={test?.loading}
                          className="p-1.5 rounded hover:bg-jade-500/15 text-ink-400 hover:text-jade-400 transition-colors disabled:opacity-40"
                          title="测试连接"
                        >
                          {test?.loading ? (
                            <Loader2 className="w-4 h-4 animate-spin" />
                          ) : (
                            <Zap className="w-4 h-4" />
                          )}
                        </button>
                        <button
                          onClick={() => openEdit(model)}
                          className="p-1.5 rounded hover:bg-gold-400/10 text-ink-400 hover:text-gold-300 transition-colors"
                          title="编辑"
                        >
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => setDeleting(model)}
                          className="p-1.5 rounded hover:bg-cinnabar-500/20 text-ink-400 hover:text-cinnabar-400 transition-colors"
                          title="删除"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                      {/* 连接测试结果 */}
                      {test?.result && (
                        <p className={`
                          text-[10px] mt-1 text-right flex items-center justify-end gap-1
                          ${test.result.success ? 'text-jade-400' : 'text-cinnabar-400'}
                        `}>
                          {test.result.success ? (
                            <>
                              <Check className="w-3 h-3" />
                              连接成功 · {test.result.latency_ms}ms
                            </>
                          ) : (
                            <>
                              <AlertCircle className="w-3 h-3" />
                              {test.result.error || '连接失败'}
                            </>
                          )}
                        </p>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* 创建/编辑弹窗 */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="dao-card w-full max-w-lg p-6 animate-fade-in max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-5">
              <div className="flex items-center gap-2">
                <Cpu className="w-5 h-5 text-jade-400" />
                <h2 className="text-lg font-serif font-bold text-gold-300">
                  {editing ? '编辑模型' : '新增模型'}
                </h2>
              </div>
              <button
                onClick={() => setShowForm(false)}
                className="p-1.5 rounded-lg hover:bg-ink-700 text-ink-400 hover:text-rice-paper-100 transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="dao-label">模型名称 *</label>
                  <input
                    type="text"
                    value={form.name}
                    onChange={e => setForm({ ...form, name: e.target.value })}
                    placeholder="如：gpt-4o"
                    className="dao-input"
                    required
                  />
                </div>
                <div>
                  <label className="dao-label">显示名</label>
                  <input
                    type="text"
                    value={form.display_name}
                    onChange={e => setForm({ ...form, display_name: e.target.value })}
                    placeholder="如：GPT-4o"
                    className="dao-input"
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="dao-label">服务商 *</label>
                  <select
                    value={form.provider}
                    onChange={e => setForm({ ...form, provider: e.target.value as ModelProvider })}
                    className="dao-input"
                  >
                    {PROVIDERS.map(p => (
                      <option key={p.value} value={p.value}>{p.label}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="dao-label">Base URL</label>
                  <input
                    type="text"
                    value={form.base_url}
                    onChange={e => setForm({ ...form, base_url: e.target.value })}
                    placeholder="https://api.openai.com/v1"
                    className="dao-input"
                  />
                </div>
              </div>

              <div>
                <label className="dao-label">API Key</label>
                <input
                  type="password"
                  value={form.api_key}
                  onChange={e => setForm({ ...form, api_key: e.target.value })}
                  placeholder={
                    editing
                      ? editing.has_api_key
                        ? editing.api_key_masked || '已配置密钥（留空不修改）'
                        : '未配置密钥'
                      : 'sk-xxxxxxxxxxxxxxxxxxxxxxxx'
                  }
                  className="dao-input"
                  autoComplete="new-password"
                />
                {editing && (
                  <p className="text-[10px] text-ink-500 mt-1">
                    留空表示不修改密钥；填写新密钥将替换原密钥
                  </p>
                )}
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div>
                  <label className="dao-label">温度（0-2）</label>
                  <input
                    type="number"
                    min={0}
                    max={2}
                    step={0.1}
                    value={form.temperature}
                    onChange={e => setForm({ ...form, temperature: Number(e.target.value) })}
                    className="dao-input"
                  />
                </div>
                <div>
                  <label className="dao-label">最大 Token</label>
                  <input
                    type="number"
                    min={1}
                    step={1}
                    value={form.max_tokens}
                    onChange={e => setForm({ ...form, max_tokens: Math.max(1, Math.floor(Number(e.target.value))) })}
                    className="dao-input"
                  />
                </div>
                <div>
                  <label className="dao-label">排序</label>
                  <input
                    type="number"
                    min={0}
                    step={1}
                    value={form.sort_order}
                    onChange={e => setForm({ ...form, sort_order: Math.max(0, Math.floor(Number(e.target.value))) })}
                    className="dao-input"
                  />
                </div>
              </div>

              <div className="flex flex-wrap gap-x-6 gap-y-2">
                <label className="flex items-center gap-2 text-sm text-rice-paper-200 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.is_enabled}
                    onChange={e => setForm({ ...form, is_enabled: e.target.checked })}
                    className="accent-jade-500 w-4 h-4"
                  />
                  启用
                </label>
                <label className="flex items-center gap-2 text-sm text-rice-paper-200 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.is_default}
                    onChange={e => setForm({ ...form, is_default: e.target.checked })}
                    className="accent-gold-500 w-4 h-4"
                  />
                  设为默认模型
                </label>
                <label className="flex items-center gap-2 text-sm text-rice-paper-200 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.is_synthesis}
                    onChange={e => setForm({ ...form, is_synthesis: e.target.checked })}
                    className="accent-jade-500 w-4 h-4"
                  />
                  用于丹性合成
                </label>
              </div>

              {formError && (
                <div className="flex items-center gap-2 text-sm text-cinnabar-400 bg-cinnabar-500/10 border border-cinnabar-500/20 rounded-lg px-3 py-2">
                  <AlertCircle className="w-4 h-4 flex-shrink-0" />
                  <span>{formError}</span>
                </div>
              )}

              <div className="flex items-center gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setShowForm(false)}
                  className="dao-btn-ghost flex-1"
                >
                  取消
                </button>
                <button
                  type="submit"
                  disabled={!form.name.trim() || saving}
                  className="dao-btn-primary flex-1 disabled:opacity-50"
                >
                  {saving ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Check className="w-4 h-4" />
                  )}
                  {editing ? '保存' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* 删除确认弹窗 */}
      {deleting && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="dao-card w-full max-w-sm p-6 animate-fade-in">
            <div className="flex items-center gap-2 mb-4">
              <AlertCircle className="w-5 h-5 text-cinnabar-400" />
              <h2 className="text-lg font-serif font-bold text-gold-300">删除模型</h2>
            </div>
            <p className="text-sm text-ink-300 mb-2">
              确定要删除模型「{deleting.display_name || deleting.name}」吗？此操作不可撤销。
            </p>
            {deleting.referenced_by > 0 && (
              <p className="text-xs text-gold-300/90 bg-gold-500/10 border border-gold-500/20 rounded-lg px-3 py-2 mb-2">
                该模型正被 {deleting.referenced_by} 个道人引用，删除将被拒绝
              </p>
            )}
            <div className="flex items-center gap-3 mt-5">
              <button
                onClick={() => setDeleting(null)}
                className="dao-btn-ghost flex-1"
              >
                取消
              </button>
              <button
                onClick={handleDelete}
                disabled={deleteLoading}
                className="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-cinnabar-500/20 border border-cinnabar-500/40 text-cinnabar-300 hover:bg-cinnabar-500/30 transition-colors text-sm font-medium disabled:opacity-50"
              >
                {deleteLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Trash2 className="w-4 h-4" />
                )}
                删除
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 错误提示 */}
      {error && (
        <div className="fixed bottom-20 md:bottom-6 right-4 dao-card p-3 flex items-center gap-2 text-sm text-cinnabar-400 animate-fade-in z-50">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          <span>{error}</span>
          <button
            onClick={() => setError(null)}
            className="p-1 rounded hover:bg-ink-700 text-ink-400"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      )}
    </Layout>
  )
}
