'use client'

import { useSearchParams } from 'next/navigation'
import AgentDetailPage from './agent-detail'
import { parseEntityDetailId } from '@/lib/entity-detail-route'

/**
 * 道人详情适配层:从查询参数解析实体 UUID 透传给详情页。
 * 缺失/占位/畸形 id 一律透传 undefined,由详情页展示「链接无效」且不请求 API。
 */
export function AgentDetailPageClient() {
  const searchParams = useSearchParams()
  return <AgentDetailPage agentId={parseEntityDetailId(searchParams)} />
}
