'use client'

import { useSearchParams } from 'next/navigation'
import PillDetailPage from './pill-detail'
import { parseEntityDetailId } from '@/lib/entity-detail-route'

/**
 * 金丹详情适配层:从查询参数解析实体 UUID 透传给详情页。
 * 缺失/占位/畸形 id 一律透传 undefined,由详情页展示「链接无效」且不请求 API。
 */
export function PillDetailPageClient() {
  const searchParams = useSearchParams()
  return <PillDetailPage pillId={parseEntityDetailId(searchParams)} />
}
