'use client'

/**
 * 客户端 locale Provider:让语言选择在非 locale 路由下也能全局生效。
 *
 * 生产环境 output:'export' 下页面在构建期静态渲染,服务端无法读取
 * localStorage。此组件在浏览器端挂载后读取 `alchemy-locale`,如果与
 * 服务端初始 locale 不同,则切换 locale 并重新渲染整棵子树。
 *
 * - `(main)/layout.tsx` 传 `preferStored={true}`:非 locale 路由(
 *   /pills /chat 等)以 localStorage 为准,覆盖默认中文。
 * - `[locale]/layout.tsx` 传 `preferStored={false}`:locale 前缀路由以
 *   URL 为准,尊重 `/zh-CN` 或 `/en` 的直接访问。
 */

import { useEffect, useState } from 'react'
import { NextIntlClientProvider } from 'next-intl'
import { defaultLocale, type Locale, locales } from '@/i18n'
import zhCN from '@/messages/zh-CN.json'
import en from '@/messages/en.json'

const messages = {
  'zh-CN': zhCN,
  en,
} as const

export const LOCALE_STORAGE_KEY = 'alchemy-locale'
export const LOCALE_COOKIE_NAME = 'NEXT_LOCALE'

function getStoredLocale(): Locale | null {
  if (typeof window === 'undefined') return null
  const raw = localStorage.getItem(LOCALE_STORAGE_KEY) as Locale | null
  return raw && locales.includes(raw) ? raw : null
}

export function LocaleProvider({
  children,
  initialLocale,
  preferStored = false,
}: {
  children: React.ReactNode
  initialLocale: Locale
  preferStored?: boolean
}) {
  const [locale, setLocale] = useState<Locale>(initialLocale)

  useEffect(() => {
    if (!preferStored) return
    const stored = getStoredLocale()
    if (stored && stored !== locale) {
      setLocale(stored)
    }
  }, [preferStored, locale])

  return (
    <NextIntlClientProvider locale={locale} messages={messages[locale]}>
      {children}
    </NextIntlClientProvider>
  )
}

/**
 * 持久化并应用新的 locale。
 * 同时写入 localStorage 与 cookie,让 middleware 在访问 `/` 时也能识别偏好。
 */
export function persistLocale(next: Locale) {
  if (typeof window === 'undefined') return
  localStorage.setItem(LOCALE_STORAGE_KEY, next)
  document.cookie = `${LOCALE_COOKIE_NAME}=${next};path=/;max-age=${60 * 60 * 24 * 365};SameSite=Lax`
}
