'use client'

/**
 * 金丹阁页面 - 消耗品库存（金丹消耗品重构任务 6 Phase D）
 * - 列表来自 GET /pill-items（真实库存实例，恒 available），按丹方分组显示数量；
 *   每组标题链接到丹方详情，行条目链接到具体实例详情（/pills/detail?id=<itemId>）。
 * - 没有库存时提示去丹方炼制；不显示任何"可无限服用的内置定义"。
 * - 分页加载（PAGE_SIZE=24），分组在客户端按 recipe_id 聚合当前页。
 * - 服用/弃置等写操作在实例详情（Phase E 服用对话框），本页只读展示库存。
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import Link from 'next/link'
import { useTranslations } from 'next-intl'
import {
  AlertCircle,
  ChevronRight,
  CircleDot,
  FlaskConical,
  Loader2,
  PackageOpen,
  RefreshCw,
} from 'lucide-react'
import { listPillItems } from '@/services/pillInventoryService'
import { pillItemDetailHref, recipeDetailHref } from '@/lib/entity-detail-route'
import { formatDateTime } from '@/utils/format'
import type { PillItemListItem } from '@/services/types'

const PAGE_SIZE = 24

type LoadState = 'loading' | 'ready' | 'error'

/** 客户端按丹方聚合的分组 */
interface RecipeGroup {
  recipeId: string
  name: string
  items: PillItemListItem[]
}

export default function PillsPage() {
  const t = useTranslations('pills')
  const [items, setItems] = useState<PillItemListItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [load, setLoad] = useState<LoadState>('loading')
  const [errorMessage, setErrorMessage] = useState('')
  const [loadingMore, setLoadingMore] = useState(false)
  const inFlightRef = useRef(false)

  /** 加载一页；append 为 true 时追加到现有列表（加载更多），否则首屏替换 */
  const loadPage = useCallback(async (pageNo: number, append: boolean) => {
    if (inFlightRef.current) return
    inFlightRef.current = true
    if (append) setLoadingMore(true)
    else setLoad('loading')
    try {
      const data = await listPillItems({ page: pageNo, size: PAGE_SIZE })
      setItems((prev) => (append ? [...prev, ...data.items] : data.items))
      setTotal(data.total)
      setPage(pageNo)
      setLoad('ready')
      setErrorMessage('')
    } catch (err) {
      if (!append) {
        setLoad('error')
        setErrorMessage(err instanceof Error ? err.message : String(err))
      }
    } finally {
      inFlightRef.current = false
      setLoadingMore(false)
    }
  }, [])

  useEffect(() => {
    void loadPage(1, false)
  }, [loadPage])

  // 按丹方分组（客户端聚合当前已加载页；组内保持后端顺序）
  const groups = useMemo<RecipeGroup[]>(() => {
    const map = new Map<string, RecipeGroup>()
    for (const item of items) {
      const group = map.get(item.recipe_id)
      if (group) {
        group.items.push(item)
      } else {
        map.set(item.recipe_id, { recipeId: item.recipe_id, name: item.name, items: [item] })
      }
    }
    return [...map.values()]
  }, [items])

  const hasMore = load === 'ready' && items.length < total

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      {/* 页面头部 */}
      <div className="mb-6 flex items-center gap-3">
        <CircleDot className="h-6 w-6 text-gold" />
        <div>
          <h1 className="page-title">{t('title')}</h1>
          <p className="page-subtitle">{t('subtitle')}</p>
        </div>
      </div>

      {/* 加载状态 */}
      {load === 'loading' && items.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16">
          <Loader2 className="mb-3 h-8 w-8 animate-spin text-gold" />
          <p className="text-sm text-muted-foreground">{t('loading')}</p>
        </div>
      )}

      {/* 加载失败 */}
      {load === 'error' && items.length === 0 && (
        <div role="alert" className="dao-card flex flex-col items-center px-6 py-10 text-center">
          <AlertCircle className="mb-3 h-10 w-10 text-primary" />
          <h3 className="mb-1 font-medium text-foreground">{t('loadErrorTitle')}</h3>
          <p className="mb-4 max-w-xl break-words text-sm text-muted-foreground">{errorMessage}</p>
          <button
            type="button"
            onClick={() => void loadPage(1, false)}
            className="dao-btn-ghost"
          >
            <RefreshCw className="h-4 w-4" />
            {t('retry')}
          </button>
        </div>
      )}

      {/* 空库存 → 去丹方炼制 */}
      {load === 'ready' && items.length === 0 && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <PackageOpen className="mb-3 h-12 w-12 text-sage/50" />
          <h3 className="mb-1 text-base font-medium text-muted-foreground">{t('emptyTitle')}</h3>
          <p className="mb-4 text-sm text-sage">{t('emptyDesc')}</p>
          <Link href="/recipes" className="dao-btn-primary whitespace-nowrap">
            <FlaskConical className="h-4 w-4" />
            {t('goToRecipes')}
          </Link>
        </div>
      )}

      {/* 按丹方分组的库存 */}
      {groups.length > 0 && (
        <div className="space-y-5">
          {groups.map((group) => (
            <section key={group.recipeId} className="dao-card p-5">
              <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                <Link
                  href={recipeDetailHref(group.recipeId)}
                  className="inline-flex min-w-0 items-center gap-1 font-serif text-base font-bold text-gold transition-colors hover:text-gold/80"
                >
                  <span className="truncate">{group.name}</span>
                  <ChevronRight className="h-4 w-4 shrink-0" />
                </Link>
                <span className="rounded-full border border-sage/30 bg-sage/15 px-2 py-0.5 text-[11px] text-sage">
                  {t('availableCount', { count: group.items.length })}
                </span>
              </div>
              <ul className="divide-y divide-border/60">
                {group.items.map((item) => (
                  <li key={item.id}>
                    <Link
                      href={pillItemDetailHref(item.id)}
                      className="flex items-center justify-between gap-3 py-2.5 text-sm transition-colors hover:text-gold"
                    >
                      <span className="text-muted-foreground">
                        {t('itemRevision', { revision: item.revision })}
                      </span>
                      <span className="flex items-center gap-1 text-xs text-muted-foreground">
                        {formatDateTime(item.created_at)}
                        <ChevronRight className="h-3.5 w-3.5" />
                      </span>
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      )}

      {/* 分页 */}
      {hasMore && (
        <div className="mt-6 flex justify-center">
          <button
            type="button"
            onClick={() => void loadPage(page + 1, true)}
            disabled={loadingMore}
            className="dao-btn-ghost whitespace-nowrap disabled:opacity-50"
          >
            {loadingMore ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            {t('loadMore')}
          </button>
        </div>
      )}
    </div>
  )
}
