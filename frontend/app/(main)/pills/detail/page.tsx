import { Suspense } from 'react'
import { PillDetailPageClient } from './pill-detail-page-client'

/**
 * 金丹详情页。
 *
 * 桌面端为 Next output:export 静态站点，实体 id 通过查询参数 ?id=<UUID> 传递
 * （动态路由 /pills/[id] 只预渲染占位 _，无法恢复真实 UUID）。
 * useSearchParams() 必须包在 Suspense 中才能静态导出。
 */
export default function Page() {
  return (
    <Suspense fallback={<main aria-busy="true" className="min-h-screen" />}>
      <PillDetailPageClient />
    </Suspense>
  )
}
