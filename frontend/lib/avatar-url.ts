/**
 * 头像 URL 规范化：只放行可直接进入 <img src> 的值。
 * - 空/空白 → undefined
 * - `data:image/` 前缀直接接受（trim 后原样返回，不重建 data URI）
 * - 其余仅接受 http:/https: 协议（new URL 解析，返回 url.toString()）
 * - javascript:/vbscript:/blob:/相对路径等 → undefined
 *
 * 任何非法值返回 undefined，调用方必须渲染 fallback，不得进入 <img src>。
 */
export function normalizeAvatarUrl(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  if (!trimmed) return undefined
  if (trimmed.startsWith('data:image/')) return trimmed
  try {
    const url = new URL(trimmed)
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.toString() : undefined
  } catch {
    return undefined
  }
}
