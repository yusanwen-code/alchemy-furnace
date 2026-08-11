import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import { defaultLocale, locales } from '@/i18n'

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
 * 新增:访问裸 `/` 时,优先读取 `NEXT_LOCALE` cookie(由语言切换器写入),
 * 回退到默认 locale。这样用户切换语言后,再次访问首页仍会进入上次
 * 选择的语言版本。
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  if (pathname === '/') {
    const cookieLocale = request.cookies.get('NEXT_LOCALE')?.value
    const locale =
      cookieLocale && locales.includes(cookieLocale as typeof locales[number])
        ? cookieLocale
        : defaultLocale
    return NextResponse.redirect(new URL(`/${locale}`, request.url))
  }

  return NextResponse.next()
}

export const config = {
  // Only run on the root path. Other paths are handled by their own
  // route segments and don't need any locale rewriting for the MVP.
  matcher: ['/'],
}
