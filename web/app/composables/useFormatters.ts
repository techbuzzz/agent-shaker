/**
 * useFormatters — small UI helpers used across domain cards.
 *
 * Kept dependency-free on purpose: every domain card (AgentCard, TaskCard,
 * ContextCard, MilestoneCard, GlobalContextCard, ProjectCard, StandupCard,
 * ContextViewer) calls `useFormatters()` for relative-time labels and a
 * safe string truncation. The composable auto-imports thanks to Nuxt's
 * `composables/` convention.
 *
 * If you find yourself reaching for a date / number helper that is not
 * here, prefer `@vueuse/core` over adding another inline helper.
 */
export interface Formatters {
  /** Returns a short human label like "2 minutes ago" / "in 3 days". */
  relativeTime: (iso: string | undefined | null) => string
  /** Truncates `s` to `max` characters, appending an ellipsis if needed. */
  truncate: (s: string | undefined | null, max?: number) => string
  /** Returns an absolute date string in the user's locale. */
  absoluteDate: (iso: string | undefined | null) => string
  /** Alias for absoluteDate — used by older ProjectCard / StandupCard. */
  formatDate: (iso: string | undefined | null) => string
  /** Alias for absoluteDate — used by ContextViewer. */
  formatDateTime: (iso: string | undefined | null) => string
}

export function useFormatters(): Formatters {
  function relativeTime(iso: string | undefined | null): string {
    if (!iso) return '—'
    const then = new Date(iso).getTime()
    if (Number.isNaN(then)) return '—'
    const diffSec = Math.round((then - Date.now()) / 1000)
    const abs = Math.abs(diffSec)
    const future = diffSec < 0
    const pick = (n: number, unit: Intl.RelativeTimeFormatUnit) =>
      new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' }).format(future ? n : -n, unit)
    if (abs < 60)            return pick(0, 'second')
    if (abs < 3600)          return pick(Math.round(abs / 60), 'minute')
    if (abs < 86_400)        return pick(Math.round(abs / 3600), 'hour')
    if (abs < 86_400 * 7)    return pick(Math.round(abs / 86_400), 'day')
    if (abs < 86_400 * 30)   return pick(Math.round(abs / (86_400 * 7)), 'week')
    if (abs < 86_400 * 365)  return pick(Math.round(abs / (86_400 * 30)), 'month')
    return pick(Math.round(abs / (86_400 * 365)), 'year')
  }

  function truncate(s: string | undefined | null, max = 200): string {
    if (!s) return ''
    if (s.length <= max) return s
    return s.slice(0, max - 1).trimEnd() + '…'
  }

  function absoluteDate(iso: string | undefined | null): string {
    if (!iso) return '—'
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return '—'
    return d.toLocaleString()
  }

  // Aliased names so the pre-existing ProjectCard / StandupCard /
  // ContextViewer components type-check against this composable.
  return {
    relativeTime,
    truncate,
    absoluteDate,
    formatDate: absoluteDate,
    formatDateTime: absoluteDate
  }
}
