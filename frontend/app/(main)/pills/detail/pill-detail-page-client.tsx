'use client'

import { useSearchParams } from 'next/navigation'
import PillItemDetailPage from './pill-detail'
import { parseEntityDetailId } from '@/lib/entity-detail-route'

/**
 * 金丹库存实例详情适配层:从查询参数解析库存实例 UUID 透传给详情页。
 * 缺失/占位/畸形 id 一律透传 undefined,由详情页展示「链接无效」且不请求 API。
 * 注意:该 id 语义是库存实例 itemId(计划硬约束,与丹方 recipeId 严格区分)。
 */
export function PillDetailPageClient() {
  const searchParams = useSearchParams()
  return <PillItemDetailPage itemId={parseEntityDetailId(searchParams)} />
}
