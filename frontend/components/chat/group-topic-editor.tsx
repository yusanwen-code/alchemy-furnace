'use client'

/**
 * 群聊页头主题编辑器:默认展示标题 + 编辑按钮;进入编辑态复制 title 到草稿。
 * 保存时 trim 并按 Unicode 字符数校验(≤200);onRename 返回非 null 才退出编辑态,
 * 返回 null 视为失败:保留草稿与错误,不覆盖页头/目录标题。
 */
import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { Check, Loader2, Pencil, X } from 'lucide-react'

import type { ChatSession } from '@/services/types'

const TOPIC_MAX = 200

export interface GroupTopicEditorProps {
  sessionId: string
  title: string
  onRename: (sessionId: string, title: string) => Promise<ChatSession | null>
}

export function GroupTopicEditor({ sessionId, title, onRename }: GroupTopicEditorProps) {
  const t = useTranslations('chatView.topic')
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const startEdit = () => {
    setDraft(title)
    setError(null)
    setEditing(true)
  }

  const cancelEdit = () => {
    if (saving) return
    setError(null)
    setEditing(false)
  }

  const handleSave = async () => {
    if (saving) return
    const trimmed = draft.trim()
    if (!trimmed) {
      setError(t('emptyError'))
      return
    }
    if (Array.from(trimmed).length > TOPIC_MAX) {
      setError(t('tooLongError'))
      return
    }
    setSaving(true)
    setError(null)
    try {
      const updated = await onRename(sessionId, trimmed)
      if (updated) {
        setEditing(false)
      } else {
        // 失败保留草稿,标题不被覆盖
        setError(t('renameError'))
      }
    } finally {
      setSaving(false)
    }
  }

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (saving) return
    if (event.key === 'Enter') {
      event.preventDefault()
      void handleSave()
    } else if (event.key === 'Escape') {
      event.preventDefault()
      cancelEdit()
    }
  }

  if (!editing) {
    return (
      <div className="flex items-center gap-1.5 min-w-0">
        <p className="text-sm font-medium text-foreground truncate">{title}</p>
        <button
          type="button"
          onClick={startEdit}
          aria-label={t('rename')}
          title={t('rename')}
          className="p-1 rounded hover:bg-secondary text-muted-foreground hover:text-gold transition-colors shrink-0"
        >
          <Pencil className="w-3.5 h-3.5" />
        </button>
      </div>
    )
  }

  return (
    <div className="min-w-0">
      <div className="flex items-center gap-1.5">
        <input
          aria-label={t('renameLabel')}
          value={draft}
          onChange={e => {
            setError(null)
            setDraft(e.target.value)
          }}
          onKeyDown={handleKeyDown}
          disabled={saving}
          className="dao-input w-full min-w-0 text-sm py-1 px-2"
        />
        <button
          type="button"
          onClick={() => { void handleSave() }}
          disabled={saving}
          aria-label={t('saveRename')}
          title={t('saveRename')}
          className="p-1.5 rounded hover:bg-sage/15 text-sage hover:text-sage transition-colors shrink-0 disabled:cursor-wait disabled:opacity-50"
        >
          {saving ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Check className="w-3.5 h-3.5" />}
        </button>
        <button
          type="button"
          onClick={cancelEdit}
          disabled={saving}
          aria-label={t('cancelRename')}
          title={t('cancelRename')}
          className="p-1.5 rounded hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors shrink-0 disabled:cursor-wait disabled:opacity-50"
        >
          <X className="w-3.5 h-3.5" />
        </button>
      </div>
      {error && <p role="alert" className="text-[10px] text-primary mt-1">{error}</p>}
    </div>
  )
}
