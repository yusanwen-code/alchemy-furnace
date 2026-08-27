'use client'

/**
 * 金丹详情 - 导出 Skill 弹窗
 * 只对已保存金丹(详情页只读态)开放;生成 Codex/Claude 可用的 Skill 包下载:
 * - 选择目标平台,展示文件名(展示用 slug 近似,实际下载名以服务端 Content-Disposition 为准)
 * - 展示包含内容清单、来源说明与「不会包含 API Key 与网页全文」确认文案
 * - 成功触发浏览器下载(Blob + a[download]);失败保留弹窗并提供重试
 * 导出走只读接口 POST /api/v1/distillation/skill-export(pill_id 模式),不删除、不修改金丹。
 */
import { useState } from 'react'
import { useTranslations } from 'next-intl'
import {
  AlertCircle,
  Download,
  FileArchive,
  Loader2,
  ShieldCheck,
  X,
} from 'lucide-react'
import { exportSkill } from '@/services/distillationService'
import type { ExportFormat, Pill } from '@/services/types'

interface SkillExportDialogProps {
  pill: Pill
  onClose: () => void
}

/**
 * 展示用 slug:与服务端规则同方向近似(小写 ASCII、非字母数字转短横、去首尾短横、截断 48)。
 * 仅用于弹窗内文件名预览;真实文件名由服务端 Content-Disposition 权威下发。
 */
export function skillSlugForDisplay(name: string): string {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 48)
  return slug || 'skill'
}

/** 展示用下载文件名:alchemy-skill-<slug>-<format>.zip(plan §3.4) */
export function skillExportFilename(name: string, format: ExportFormat): string {
  return `alchemy-skill-${skillSlugForDisplay(name)}-${format}.zip`
}

/** 浏览器下载:Blob URL + a[download],下载后立即回收 */
function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

type ExportStatus = 'idle' | 'submitting' | 'success' | 'error'

export function SkillExportDialog({ pill, onClose }: SkillExportDialogProps) {
  const t = useTranslations('skillExport')
  const [format, setFormat] = useState<ExportFormat>('codex')
  const [status, setStatus] = useState<ExportStatus>('idle')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const handleExport = async () => {
    setStatus('submitting')
    setErrorMessage(null)
    try {
      const result = await exportSkill({ pill_id: pill.id, format })
      triggerDownload(result.blob, result.filename)
      setStatus('success')
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : t('error'))
      setStatus('error')
    }
  }

  const filename = skillExportFilename(pill.name, format)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4 backdrop-blur-sm">
      <div className="dao-card max-h-[80vh] w-full max-w-md overflow-y-auto p-6">
        <div className="mb-5 flex items-center justify-between gap-2">
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <FileArchive className="h-5 w-5 shrink-0 text-gold" />
            <h2 className="truncate font-serif text-lg font-bold text-foreground">{t('title')}</h2>
          </div>
          <button
            type="button"
            aria-label={t('closeModal')}
            onClick={onClose}
            className="shrink-0 rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* 目标平台选择 */}
        <label className="dao-label">{t('formatLabel')}</label>
        <div className="mb-4 grid grid-cols-2 gap-3">
          {(['codex', 'claude'] as const).map((option) => (
            <button
              key={option}
              type="button"
              aria-pressed={format === option}
              onClick={() => setFormat(option)}
              className={[
                'rounded-xl border px-3 py-2.5 text-sm font-medium transition-all',
                format === option
                  ? 'border-gold/40 bg-gold/10 text-gold'
                  : 'border-border/70 bg-secondary/70 text-muted-foreground hover:border-gold/30',
              ].join(' ')}
            >
              {t(option)}
            </button>
          ))}
        </div>

        {/* 文件名 */}
        <label className="dao-label">{t('filenameLabel')}</label>
        <p className="mb-4 truncate rounded-lg border border-border/70 bg-muted px-3 py-2 text-sm text-foreground">
          {filename}
        </p>

        {/* 包含内容清单 */}
        <label className="dao-label">{t('contentsLabel')}</label>
        <ul className="mb-4 space-y-1.5">
          {(['contents.skillMd', 'contents.sourcesMd', 'contents.readmeMd'] as const).map(
            (key) => (
              <li key={key} className="text-sm text-muted-foreground">
                {t(key)}
              </li>
            ),
          )}
          {format === 'claude' && (
            <li className="text-sm text-muted-foreground">{t('contents.platformJson')}</li>
          )}
        </ul>

        {/* 来源说明 */}
        <p className="mb-3 rounded-lg border border-sage/30 bg-sage/10 p-3 text-xs leading-relaxed text-muted-foreground">
          {t('sourceNote')}
        </p>

        {/* 导出前确认文案 */}
        <div className="mb-4 flex items-start gap-2 rounded-lg border border-gold/25 bg-gold/5 p-3">
          <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-gold" />
          <p className="text-xs font-medium leading-relaxed text-foreground">{t('privacyNote')}</p>
        </div>

        {/* 失败:保留弹窗并展示错误与重试 */}
        {status === 'error' && (
          <div role="alert" className="mb-4 rounded-lg border border-primary/30 bg-primary/5 p-3">
            <div className="flex items-center gap-2 text-xs text-primary">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" />
              <span className="min-w-0 break-words">{errorMessage}</span>
            </div>
            <button
              type="button"
              onClick={handleExport}
              className="mt-2 text-xs font-medium underline underline-offset-2"
            >
              {t('retry')}
            </button>
          </div>
        )}

        {/* 成功:留在弹窗内,可关闭 */}
        {status === 'success' && (
          <p role="status" className="mb-4 text-xs font-medium text-sage">
            {t('success')}
          </p>
        )}

        <div className="flex flex-wrap items-center gap-3">
          {status === 'success' ? (
            <button type="button" onClick={onClose} className="dao-btn-primary flex-1 whitespace-nowrap">
              {t('done')}
            </button>
          ) : (
            <>
              <button type="button" onClick={onClose} className="dao-btn-ghost flex-1 whitespace-nowrap">
                {t('cancel')}
              </button>
              <button
                type="button"
                onClick={handleExport}
                disabled={status === 'submitting'}
                className="dao-btn-primary flex-1 whitespace-nowrap disabled:opacity-50"
              >
                {status === 'submitting' ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Download className="h-4 w-4" />
                )}
                {status === 'submitting' ? t('exporting') : t('exportCta')}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
