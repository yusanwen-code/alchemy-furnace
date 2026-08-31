/**
 * 道人/金丹/丹方详情页地址的唯一事实来源。
 *
 * 桌面端使用 Next output:export，动态路由 `/agents/[id]`、`/pills/[id]` 只会预渲染
 * 占位 `_.html`，真实 UUID 无法通过 useParams() 恢复。因此实体详情地址统一为查询参数：
 *   道人详情: /agents/detail?id=<UUID>
 *   库存实例详情: /pills/detail?id=<itemId>（旧金丹 ID 经 legacy 解析跳丹方）
 *   丹方详情: /recipes/detail?id=<recipeId>
 * 所有入口都必须用这里的函数生成/解析，禁止散落 `/agents/${id}` 模板字符串。
 * 历史动态路径 `/agents/<UUID>`、`/pills/<UUID>` 由 Go WebUI 307 重定向到规范地址。
 */

export const ENTITY_DETAIL_QUERY_KEY = 'id'

/** 丹方详情页「直接进入编辑新版本」的查询参数（recipe-card 的「编辑新版本」入口） */
export const RECIPE_EDIT_QUERY_KEY = 'edit'

// RFC 4122 通用 UUID（含 version/variant 位），大小写不敏感
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

type SearchParamsReader = Pick<URLSearchParams, 'get'>
type EntityKind = 'agents' | 'pills' | 'recipes'

function detailHref(kind: EntityKind, id: string): string {
  if (!UUID_RE.test(id)) throw new Error('Invalid entity id')
  return `/${kind}/detail?${ENTITY_DETAIL_QUERY_KEY}=${encodeURIComponent(id)}`
}

/** 道人详情规范地址；非法 id 直接抛错，避免把占位/脏值拼成可访问链接 */
export function agentDetailHref(id: string): string {
  return detailHref('agents', id)
}

/** 金丹库存实例详情规范地址；非法 id 直接抛错，避免把占位/脏值拼成可访问链接 */
export function pillItemDetailHref(id: string): string {
  return detailHref('pills', id)
}

/** 丹方详情规范地址；非法 id 直接抛错，避免把占位/脏值拼成可访问链接 */
export function recipeDetailHref(id: string): string {
  return detailHref('recipes', id)
}

/** 丹方详情「编辑新版本」直达地址；非法 id 直接抛错 */
export function recipeEditHref(id: string): string {
  return `${detailHref('recipes', id)}&${RECIPE_EDIT_QUERY_KEY}=1`
}

/** 从查询参数解析出合法的实体 UUID；缺失/占位/畸形一律返回 undefined */
export function parseEntityDetailId(searchParams: SearchParamsReader): string | undefined {
  const value = searchParams.get(ENTITY_DETAIL_QUERY_KEY)?.trim()
  return value && UUID_RE.test(value) ? value : undefined
}

/**
 * 识别历史动态路径 `/agents/<UUID>`、`/pills/<UUID>`（可带尾斜杠），
 * 仅用于一次性兼容重定向。不匹配详情页本身、其它实体路径、占位 `_` 或非法 UUID。
 */
export function parseLegacyEntityDetailPath(
  pathname: string,
): { kind: EntityKind; id: string } | undefined {
  const match = pathname.match(/^\/(agents|pills)\/([^/]+)\/?$/)
  if (!match) return undefined
  const id = decodeURIComponent(match[2])
  return UUID_RE.test(id) ? { kind: match[1] as EntityKind, id } : undefined
}
