import { computed, ref, type ComputedRef, type Ref } from 'vue'
import type { Context } from '~/types/api'

/**
 * Context search state for a project's Contexts tab.
 *
 * The factory is exported separately from the Nuxt composable so the state
 * machine can be tested without booting Nuxt. It holds no I/O of its own: the
 * caller injects the fetch, which is the same shape the real API wrapper
 * already has.
 */
export interface ContextSearchState {
  query: Ref<string>
  results: Ref<Context[] | null>
  searching: Ref<boolean>
  /** True while a search result set is on screen, as opposed to the full list. */
  isActive: ComputedRef<boolean>
  /** What the list should render: search hits when searching, everything otherwise. */
  displayed: ComputedRef<Context[]>
  run: () => Promise<void>
  clear: () => void
}

export interface ContextSearchOptions {
  /** Resolved per call rather than captured, so a route change re-scopes the search. */
  projectId: () => string
  search: (filters: { project_id: string; q: string; limit?: number }) => Promise<Context[]>
  /** The full list already loaded by the page, used whenever search is inactive. */
  all?: Ref<Context[] | null | undefined>
  limit?: number
  onError?: (message: string) => void
}

/** DEFAULT_SEARCH_LIMIT mirrors the server's default; the server caps it regardless. */
export const DEFAULT_SEARCH_LIMIT = 20

/**
 * extractErrorMessage pulls a human-readable string out of a $fetch failure.
 * The two shapes seen in practice are a structured API error body and a plain
 * transport message, and showing the user neither leaves them with a spinner
 * that never resolves into an explanation.
 */
export function extractErrorMessage(err: unknown, fallback: string): string {
  const e = err as { data?: { message?: string }; message?: string } | null
  return e?.data?.message ?? e?.message ?? fallback
}

export function createContextSearch(options: ContextSearchOptions): ContextSearchState {
  const query = ref('')
  const results = ref<Context[] | null>(null)
  const searching = ref(false)

  const isActive = computed(() => results.value !== null)
  const displayed = computed(() => results.value ?? options.all?.value ?? [])

  function clear(): void {
    query.value = ''
    results.value = null
  }

  async function run(): Promise<void> {
    const q = query.value.trim()

    // An empty query means "no search", not "match everything". The server
    // short-circuits on this too, but clearing here keeps the UI from
    // rendering an empty result set that looks like a project with no notes.
    if (!q) {
      clear()
      return
    }

    searching.value = true
    try {
      results.value = await options.search({
        project_id: options.projectId(),
        q,
        limit: options.limit ?? DEFAULT_SEARCH_LIMIT
      })
    } catch (err: unknown) {
      // Fall back to the full list rather than leaving a stale or partial
      // result set on screen: a user who cannot tell "no match" from "the
      // search broke" will just run the same query again.
      results.value = null
      options.onError?.(extractErrorMessage(err, 'Context search failed'))
    } finally {
      searching.value = false
    }
  }

  return { query, results, searching, isActive, displayed, run, clear }
}
