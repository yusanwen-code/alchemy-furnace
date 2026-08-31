'use client'

import { useSearchParams } from 'next/navigation'
import RecipeDetailPage from './recipe-detail'
import {
  parseEntityDetailId,
  RECIPE_EDIT_QUERY_KEY,
} from '@/lib/entity-detail-route'

/**
 * 丹方详情适配层:从查询参数解析实体 UUID 透传给详情页。
 * 缺失/占位/畸形 id 一律透传 undefined,由详情页展示「链接无效」且不请求 API。
 * ?edit=1 由丹方卡片「编辑新版本」入口带入,详情页就绪后直接进入编辑态。
 */
export function RecipeDetailPageClient() {
  const searchParams = useSearchParams()
  return (
    <RecipeDetailPage
      recipeId={parseEntityDetailId(searchParams)}
      initialEdit={searchParams.get(RECIPE_EDIT_QUERY_KEY) === '1'}
    />
  )
}
