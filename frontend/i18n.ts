import { getRequestConfig } from 'next-intl/server'
import { notFound } from 'next/navigation'

/**
 * next-intl request config.
 *
 * Locale codes are intentionally short so they fit naturally as URL segments:
 *   - `/zh-CN/...`  Chinese (default)
 *   - `/en/...`     English
 *
 * The locale is resolved from the `[locale]` route segment via `requestLocale`,
 * so we don't need next-intl's middleware just to detect the URL. A small
 * dedicated middleware (see `frontend/middleware.ts`) only handles the root
 * path redirect to keep the home page's URL stable.
 */
export const locales = ['zh-CN', 'en'] as const
export type Locale = (typeof locales)[number]
export const defaultLocale: Locale = 'zh-CN'

export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale
  const locale = (locales as readonly string[]).includes(requested ?? '')
    ? (requested as Locale)
    : null

  if (!locale) {
    notFound()
  }

  return {
    locale,
    messages: (await import(`./messages/${locale}.json`)).default,
  }
})
