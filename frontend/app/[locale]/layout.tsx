import { notFound } from 'next/navigation'
import { NextIntlClientProvider } from 'next-intl'
import { getMessages, setRequestLocale } from 'next-intl/server'
import type { ReactNode } from 'react'
import { Navbar } from '@/components/layout/navbar'
import { Footer } from '@/components/layout/footer'
import { locales } from '@/i18n'

/**
 * Locale-scoped layout.
 *
 * Owns the i18n provider AND the global chrome (Navbar / Footer). The
 * chrome must live here — not in the root layout — because the root
 * layout cannot read the active locale without `headers()`, which
 * breaks `output: 'export'`'s static prerender of `/_not-found`.
 *
 * `setRequestLocale` is required for static rendering under
 * `output: 'export'`; it tells next-intl which locale the page tree
 * should be rendered for.
 */
export function generateStaticParams() {
  return locales.map((locale) => ({ locale }))
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: ReactNode
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params

  if (!(locales as readonly string[]).includes(locale)) {
    notFound()
  }

  setRequestLocale(locale)
  const messages = await getMessages()

  return (
    <NextIntlClientProvider messages={messages} locale={locale}>
      <Navbar />
      <div className="flex-1">{children}</div>
      <Footer />
    </NextIntlClientProvider>
  )
}
