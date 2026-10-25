/**
 * The hook layer's display helpers.
 *
 * They live here rather than in the page because three things render them: the
 * hook card, the run row and the test-result dialog. Three copies of a
 * duration formatter is three places for them to drift.
 */
import type { HookRunStatus } from '~/types/hooks'
import { HOOK_EVENT_LABELS } from '~/types/hooks'

/**
 * The palette is the app's semantic one: green for a run that did its job,
 * red for one that failed or ran out of time, amber for one that blocked on
 * purpose, neutral for one that was cancelled.
 *
 * The word is always next to the colour — the badge text is the status, the
 * colour only makes it scannable.
 */
export const HOOK_RUN_STATUS_COLORS: Record<HookRunStatus, 'success' | 'error' | 'warning' | 'neutral'> = {
  succeeded: 'success',
  failed: 'error',
  timeout: 'error',
  blocked: 'warning',
  cancelled: 'neutral'
}

/**
 * The label for an event name.
 *
 * Typed as a lookup that can miss, because the server is the authority on the
 * vocabulary: a run carrying an event this build does not know about still
 * renders, with the raw name, instead of rendering as a blank badge.
 */
export function hookEventLabel(event: string): string {
  const labels = HOOK_EVENT_LABELS as Partial<Record<string, string>>
  return labels[event] ?? event
}

/**
 * A run's duration, in the unit that reads best at that size.
 *
 * A missing duration is an em dash rather than zero: a run whose duration the
 * server did not record is not a run that took no time.
 */
export function formatHookDuration(ms: number | null | undefined): string {
  if (ms == null || !Number.isFinite(ms)) return '—'
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.round(ms / 60_000)}m`
}
