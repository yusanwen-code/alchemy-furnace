'use client'

import { useTransition } from 'react'
import { useLocale, useTranslations } from 'next-intl'
import { Languages } from 'lucide-react'
import { locales, type Locale } from '@/i18n'
import { cn } from '@/lib/utils'

/**
 * Compact language switcher.
 *
 * Lives in the navbar on every page. Clicking a locale navigates to the
 * home page of the target locale (`/<locale>`). For the MVP we only ship
 * the home page under `[locale]/`; the other top-level routes
 * (`/pills`, `/chat`, …) don't have locale-prefixed versions yet, so
 * switching from one of those pages always lands on the new locale's
 * home. Once those routes are migrated, this component can be extended
 * to preserve the current sub-path.
 */
export function LanguageSwitcher({ className }: { className?: string }) {
  const currentLocale = useLocale() as Locale
  const t = useTranslations('languageSwitcher')
  const [pending, startTransition] = useTransition()

  const handleSelect = (next: Locale) => {
    if (next === currentLocale) return
    // Hard navigation so the root layout re-runs and the new locale's
    // metadata / fonts apply cleanly. Using `next/navigation`'s router
    // would only re-render the [locale] subtree, which is also fine —
    // but a hard nav mirrors what a fresh page load looks like, which
    // is what users expect from a language switcher.
    startTransition(() => {
      window.location.href = `/${next}`
    })
  }

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1 rounded-full border border-border/70 bg-card/80 p-0.5 text-xs shadow-[0_4px_12px_-6px_rgba(60,40,20,0.12)]',
        pending && 'opacity-60',
        className,
      )}
      role="group"
      aria-label={t('label')}
    >
      <Languages className="ml-1.5 size-3.5 text-muted-foreground" aria-hidden />
      {locales.map((loc) => {
        const active = loc === currentLocale
        return (
          <button
            key={loc}
            type="button"
            onClick={() => handleSelect(loc)}
            disabled={pending}
            aria-pressed={active}
            className={cn(
              'rounded-full px-2.5 py-1 font-medium transition-colors duration-200',
              active
                ? 'bg-primary text-primary-foreground shadow-[0_4px_10px_-4px_rgba(181,74,63,0.5)]'
                : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
            )}
          >
            {t(loc)}
          </button>
        )
      })}
    </div>
  )
}
