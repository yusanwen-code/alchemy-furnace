import { NextIntlClientProvider } from 'next-intl'
import { setRequestLocale } from 'next-intl/server'
import type { ReactNode } from 'react'
import { Navbar } from '@/components/layout/navbar'
import { Footer } from '@/components/layout/footer'
import { defaultLocale } from '@/i18n'
import zhCN from '@/messages/zh-CN.json'

/**
 * Layout for non-locale MVP routes (`/agents`, `/chat`, `/pills`,
 * `/models`, `/settings`).
 *
 * Wraps the subtree in the chrome (Navbar / Footer) so these pages
 * have the same global navigation as the home page under
 * `app/[locale]/`. The parenthesized folder name `(main)` is a Next.js
 * route group — it does NOT appear in the URL.
 *
 * Why `setRequestLocale(defaultLocale)` is called before the provider:
 *   The next-intl plugin (configured in `next.config.mjs`) hooks into
 *   every layout in the app. When the request locale is undefined
 *   (non-locale routes), `i18n.ts`'s `getRequestConfig` calls
 *   `notFound()`, and the entire page prerenders as a 404 — even
 *   though the page itself is fine. Setting the request locale to the
 *   default here mirrors what `app/[locale]/layout.tsx` does for
 *   locale routes, and is what makes the chrome render at all.
 *
 * Messages are hard-coded to the default locale because these routes
 * don't carry a `[locale]` segment. The chrome falls back to Chinese
 * for them; on `/zh-CN` or `/en` the inner `app/[locale]/layout.tsx`
 * provider takes over and the chrome re-translates.
 */
export default function MainLayout({ children }: { children: ReactNode }) {
  setRequestLocale(defaultLocale)
  return (
    <NextIntlClientProvider locale={defaultLocale} messages={zhCN}>
      <Navbar />
      <div className="flex-1">{children}</div>
      <Footer />
    </NextIntlClientProvider>
  )
}
