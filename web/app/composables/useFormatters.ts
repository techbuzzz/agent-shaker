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
  /** Returns an absolute date string, rendered in UTC for SSR stability. */
  absoluteDate: (iso: string | undefined | null) => string
  /** The instant as ISO-8601 UTC, for datetime attributes and tooltips. */
  absoluteIso: (iso: string | undefined | null) => string
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

  /**
   * Absolute timestamp, rendered deterministically.
   *
   * This used to be a bare `toLocaleString()`. That takes its locale AND time
   * zone from the runtime, so the server rendered the container's UTC while the
   * browser rendered the user's local zone — the same instant, two different
   * strings, and Vue reported a hydration mismatch on every card that shows a
   * date. It only reproduced in production: dev is slow enough that the timing
   * differed, which is exactly why this survived a green test run.
   *
   * Pinning both is the only way to keep server and client byte-identical.
   * UTC is chosen over the visitor's zone because a coordinator comparing two
   * agents' timestamps needs one shared clock, not two local ones; the exact
   * instant is also emitted as ISO in `absoluteIso` for callers that need to
   * put it in a `datetime` attribute or a tooltip.
   *
   * When a real i18n layer lands, this becomes a locale lookup plus a
   * user-configured zone — at which point the value must either stay server-
   * rendered and consistent, or move behind <ClientOnly>. It cannot go back to
   * bare toLocaleString().
   */
  const ABSOLUTE_LOCALE = 'en-GB'
  const ABSOLUTE_TIME_ZONE = 'UTC'

  function absoluteDate(iso: string | undefined | null): string {
    if (!iso) return '—'
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return '—'
    return d.toLocaleString(ABSOLUTE_LOCALE, {
      timeZone: ABSOLUTE_TIME_ZONE,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    })
  }

  /** The instant as ISO-8601 UTC, for datetime attributes and tooltips. */
  function absoluteIso(iso: string | undefined | null): string {
    if (!iso) return ''
    const d = new Date(iso)
    if (Number.isNaN(d.getTime())) return ''
    return d.toISOString()
  }

  // Aliased names so the pre-existing ProjectCard / StandupCard /
  // ContextViewer components type-check against this composable.
  return {
    relativeTime,
    truncate,
    absoluteDate,
    absoluteIso,
    formatDate: absoluteDate,
    formatDateTime: absoluteDate
  }
}
