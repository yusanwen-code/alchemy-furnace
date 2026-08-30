/**
 * 头像 URL 规范化：只放行可直接进入 <img src> 的值。
 * - 空/空白 → undefined
 * - `data:image/` 前缀直接接受（trim 后原样返回，不重建 data URI）
 * - 其余仅接受 http:/https: 协议（new URL 解析，返回 url.toString()）
 * - javascript:/vbscript:/blob:/相对路径/带用户名密码/超长等 → undefined
 *
 * 先经 validateAvatarField 共享校验（与后端契约一致），校验失败返回 undefined，
 * 调用方必须渲染 fallback，不得进入 <img src>。
 */
import { validateAvatarField } from '@/lib/avatar-validation'

export function normalizeAvatarUrl(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  if (!trimmed) return undefined
  if (validateAvatarField(trimmed)) return undefined
  if (trimmed.startsWith('data:image/')) return trimmed
  try {
    return new URL(trimmed).toString()
  } catch {
    return undefined
  }
}
