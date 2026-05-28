/**
 * 金丹卡片组件 - 道教丹药瓶视觉风格
 * 显示金丹名称、状态、丹方数量、创建时间
 * 响应式：桌面端横向卡片，H5 纵向卡片
 */
import { Link } from 'react-router-dom'
import { CircleDot, FileText, Clock, ChevronRight, FlaskConical } from 'lucide-react'
import type { Pill } from '@/services/types'
import { PILL_STATUS_MAP, formatDateTime } from '@/utils/format'

interface PillCardProps {
  pill: Pill
  /** 紧凑模式（列表中使用） */
  compact?: boolean
}

export default function PillCard({ pill, compact = false }: PillCardProps) {
  const statusInfo = PILL_STATUS_MAP[pill.status] || PILL_STATUS_MAP.refining

  if (compact) {
    // 紧凑模式：横向小卡片
    return (
      <Link
        to={`/pills/${pill.id}`}
        className="dao-card flex items-center gap-4 p-4 group"
      >
        {/* 丹药瓶图标 */}
        <div className={`
          flex-shrink-0 w-12 h-12 rounded-xl flex items-center justify-center
          ${pill.status === 'refined'
            ? 'bg-gold-500/15 text-gold-400'
            : pill.status === 'refining'
              ? 'bg-jade-500/15 text-jade-400'
              : 'bg-cinnabar-500/15 text-cinnabar-400'
          }
        `}>
          <FlaskConical className="w-6 h-6" />
        </div>

        {/* 信息 */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="font-serif font-semibold text-rice-paper-100 truncate group-hover:text-gold-300 transition-colors">
              {pill.name}
            </h3>
            <span className={statusInfo.badgeClass}>{statusInfo.label}</span>
          </div>
          <p className="text-xs text-ink-400 mt-0.5 truncate">{pill.description || '暂无描述'}</p>
        </div>

        {/* 统计 */}
        <div className="hidden sm:flex items-center gap-4 text-xs text-ink-400">
          <span className="flex items-center gap-1">
            <FileText className="w-3.5 h-3.5" />
            {pill.vector_count} 向量
          </span>
          <span className="flex items-center gap-1">
            <Clock className="w-3.5 h-3.5" />
            {formatDateTime(pill.created_at)}
          </span>
        </div>

        <ChevronRight className="w-5 h-5 text-ink-500 group-hover:text-gold-400 transition-colors" />
      </Link>
    )
  }

  // 默认模式：纵向卡片（网格中使用）
  return (
    <Link
      to={`/pills/${pill.id}`}
      className="dao-card flex flex-col p-5 group h-full"
    >
      {/* 顶部：图标 + 状态 */}
      <div className="flex items-start justify-between mb-3">
        <div className={`
          w-14 h-14 rounded-2xl flex items-center justify-center
          ${pill.status === 'refined'
            ? 'bg-gold-500/15 text-gold-400 glow-gold'
            : pill.status === 'refining'
              ? 'bg-jade-500/15 text-jade-400'
              : 'bg-cinnabar-500/15 text-cinnabar-400'
          }
          transition-all duration-300 group-hover:scale-110
        `}>
          <FlaskConical className="w-7 h-7" />
        </div>
        <span className={statusInfo.badgeClass}>{statusInfo.label}</span>
      </div>

      {/* 名称 */}
      <h3 className="font-serif font-bold text-lg text-rice-paper-100 group-hover:text-gold-300 transition-colors mb-1.5">
        {pill.name}
      </h3>

      {/* 描述 */}
      {pill.description && (
        <p className="text-sm text-ink-400 line-clamp-2 mb-4 flex-1">
          {pill.description}
        </p>
      )}

      {/* 底部分隔 */}
      <div className="dao-divider my-3 text-[10px]">
        <CircleDot className="w-3 h-3" />
      </div>

      {/* 底部信息 */}
      <div className="flex items-center justify-between text-xs text-ink-400">
        <span className="flex items-center gap-1">
          <FileText className="w-3.5 h-3.5" />
          {pill.vector_count} 向量
        </span>
        <span className="flex items-center gap-1">
          <Clock className="w-3.5 h-3.5" />
          {formatDateTime(pill.created_at)}
        </span>
      </div>
    </Link>
  )
}
