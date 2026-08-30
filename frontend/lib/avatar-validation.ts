/**
 * 头像字段校验（道人/用户共用，与后端 internal/util/avatar 校验口径一致）：
 * - 空/空白值合法（清空头像）
 * - 完整 http/https URL：长度 ≤2048，不允许内嵌凭据（user:pass@）
 * - data:image/(png|jpeg|webp|gif);base64, 数据 URI：总长 ≤1_500_000，
 *   payload 仅允许 Base64 字符（A-Za-z0-9+/=）
 * - 其余（相对路径 / javascript: / vbscript: / blob: / 其他 MIME / 非法 base64 / 超长）
 *   → 'invalid' | 'tooLong'
 *
 * 本模块为纯函数，直接使用 new URL 与 data URI 解析，不反向导入 avatar-url.ts。
 */

/** 完整 http/https URL 最大长度（字符） */
export const avatarMaxURLLen = 2048
/** data:image 数据 URI 最大总长（字符） */
export const avatarMaxDataURILen = 1_500_000

/** data URI 允许的图片 MIME 子类型 */
const AVATAR_ALLOWED_MIMES = new Set(['png', 'jpeg', 'webp', 'gif'])

const BASE64_CHARS = /^[A-Za-z0-9+/=]+$/

export type AvatarFieldError = 'invalid' | 'tooLong'

/** 头像字段校验：合法返回 undefined，否则返回 'invalid' | 'tooLong' */
export function validateAvatarField(value: string): AvatarFieldError | undefined {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  if (trimmed.startsWith('data:image/')) return validateDataURI(trimmed)
  return validateURL(trimmed)
}

function validateURL(raw: string): AvatarFieldError | undefined {
  if (raw.length > avatarMaxURLLen) return 'tooLong'
  let url: URL
  try {
    url = new URL(raw)
  } catch {
    return 'invalid'
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') return 'invalid'
  if (!url.hostname) return 'invalid'
  if (url.username || url.password) return 'invalid'
  return undefined
}

function validateDataURI(uri: string): AvatarFieldError | undefined {
  if (uri.length > avatarMaxDataURILen) return 'tooLong'
  const rest = uri.slice('data:image/'.length)
  const semi = rest.indexOf(';')
  if (semi < 0) return 'invalid'
  if (!AVATAR_ALLOWED_MIMES.has(rest.slice(0, semi))) return 'invalid'
  const tail = rest.slice(semi + 1)
  if (!tail.startsWith('base64,')) return 'invalid'
  const payload = tail.slice('base64,'.length)
  if (!payload || !BASE64_CHARS.test(payload)) return 'invalid'
  return undefined
}

/** 头像输入框动态 maxLength：data URI 按 1_500_000，其余（含空值）按 2048 */
export function avatarInputMaxLength(value: string): number {
  return value.startsWith('data:image/') ? avatarMaxDataURILen : avatarMaxURLLen
}
