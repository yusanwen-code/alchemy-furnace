import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import { defaultLocale } from '@/i18n'

/**
 * Minimal locale-prefix middleware.
 *
 * We deliberately do NOT use next-intl's built-in middleware here. Reason:
 * the MVP only moves the home page into `app/[locale]/` — the other
 * top-level routes (`/pills`, `/chat`, `/agents`, `/models`, `/settings`)
 * stay at their original paths and will be migrated to the locale segment
 * in a follow-up. Running next-intl's full middleware would 404 those
 * pages by trying to redirect them to `/<locale>/...`.
 *
 * All this middleware does is redirect bare `/` to the default locale so
 * the home page always has a locale-prefixed URL. Everything else passes
 * through untouched.
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  if (pathname === '/') {
    return NextResponse.redirect(new URL(`/${defaultLocale}`, request.url))
  }

  return NextResponse.next()
}

export const config = {
  // Only run on the root path. Other paths are handled by their own
  // route segments and don't need any locale rewriting for the MVP.
  matcher: ['/'],
}
