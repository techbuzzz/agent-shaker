import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import type { Context } from '~/types/api'
import {
  createContextSearch,
  extractErrorMessage,
  DEFAULT_SEARCH_LIMIT
} from '../useContextSearch'

function makeContext(id: string, title: string): Context {
  return { id, title, content: '', tags: [], created_at: '', updated_at: '' } as unknown as Context
}

function makeSearch(results: Context[] = []) {
  return vi.fn().mockResolvedValue(results)
}

describe('createContextSearch', () => {
  it('shows the full list before any search runs', () => {
    const all = ref<Context[]>([makeContext('c1', 'Auth notes')])
    const state = createContextSearch({ projectId: () => 'p1', search: makeSearch(), all })

    expect(state.isActive.value).toBe(false)
    expect(state.displayed.value).toEqual(all.value)
  })

  it('treats a whitespace-only query as no search at all', async () => {
    // The server short-circuits on an empty query too. Hitting it anyway would
    // return every context in the project, which is the read search exists to
    // avoid, so the guard is duplicated on this side deliberately.
    const search = makeSearch()
    const all = ref<Context[]>([makeContext('c1', 'Auth notes')])
    const state = createContextSearch({ projectId: () => 'p1', search, all })

    state.query.value = '   \t  '
    await state.run()

    expect(search).not.toHaveBeenCalled()
    expect(state.isActive.value).toBe(false)
    expect(state.displayed.value).toEqual(all.value)
  })

  it('trims the query before sending it', async () => {
    const search = makeSearch()
    const state = createContextSearch({ projectId: () => 'p1', search })

    state.query.value = '  bcrypt tokens  '
    await state.run()

    expect(search).toHaveBeenCalledWith({ project_id: 'p1', q: 'bcrypt tokens', limit: DEFAULT_SEARCH_LIMIT })
  })

  it('replaces the displayed list with the hits', async () => {
    const hits = [makeContext('c9', 'Token rotation')]
    const all = ref<Context[]>([makeContext('c1', 'Auth notes'), makeContext('c2', 'Deploy runbook')])
    const state = createContextSearch({ projectId: () => 'p1', search: makeSearch(hits), all })

    state.query.value = 'rotation'
    await state.run()

    expect(state.isActive.value).toBe(true)
    expect(state.displayed.value).toEqual(hits)
    expect(state.displayed.value).not.toEqual(all.value)
  })

  it('reports an empty hit list as an active search with nothing in it', async () => {
    // Distinct from "the project has no contexts": both render an empty state,
    // but only one of them means the query was actually run.
    const all = ref<Context[]>([makeContext('c1', 'Auth notes')])
    const state = createContextSearch({ projectId: () => 'p1', search: makeSearch([]), all })

    state.query.value = 'nothing matches this'
    await state.run()

    expect(state.isActive.value).toBe(true)
    expect(state.displayed.value).toEqual([])
  })

  it('restores the full list on clear without a second request', async () => {
    const all = ref<Context[]>([makeContext('c1', 'Auth notes')])
    const search = makeSearch([makeContext('c9', 'Token rotation')])
    const state = createContextSearch({ projectId: () => 'p1', search, all })

    state.query.value = 'rotation'
    await state.run()
    expect(state.isActive.value).toBe(true)

    state.clear()

    expect(search).toHaveBeenCalledTimes(1)
    expect(state.query.value).toBe('')
    expect(state.isActive.value).toBe(false)
    expect(state.displayed.value).toEqual(all.value)
  })

  it('falls back to the full list and reports the failure', async () => {
    // A stale or half-filled result set is worse than no results: a user who
    // cannot tell "no match" from "the search broke" just re-runs the query.
    const all = ref<Context[]>([makeContext('c1', 'Auth notes')])
    const onError = vi.fn()
    const search = vi.fn().mockRejectedValue({ data: { message: 'relation does not exist' } })
    const state = createContextSearch({ projectId: () => 'p1', search, all, onError })

    state.query.value = 'rotation'
    await state.run()

    expect(state.isActive.value).toBe(false)
    expect(state.displayed.value).toEqual(all.value)
    expect(onError).toHaveBeenCalledWith('relation does not exist')
  })

  it('clears the in-flight flag whether the search succeeds or fails', async () => {
    const ok = createContextSearch({ projectId: () => 'p1', search: makeSearch([makeContext('c9', 'x')]) })
    ok.query.value = 'x'
    await ok.run()
    expect(ok.searching.value).toBe(false)

    const bad = createContextSearch({ projectId: () => 'p1', search: vi.fn().mockRejectedValue(new Error('boom')) })
    bad.query.value = 'x'
    await bad.run()
    expect(bad.searching.value).toBe(false)
  })

  it('resolves the project per call so a route change re-scopes the search', async () => {
    // projectId is a getter, not a captured value: the same page component
    // survives a route change, and a captured id would keep searching the
    // project the user just left.
    const search = makeSearch()
    let projectId = 'p1'
    const state = createContextSearch({ projectId: () => projectId, search })

    state.query.value = 'one'
    await state.run()

    projectId = 'p2'
    state.query.value = 'two'
    await state.run()

    expect(search).toHaveBeenNthCalledWith(1, { project_id: 'p1', q: 'one', limit: DEFAULT_SEARCH_LIMIT })
    expect(search).toHaveBeenNthCalledWith(2, { project_id: 'p2', q: 'two', limit: DEFAULT_SEARCH_LIMIT })
  })

  it('honours an explicit result limit', async () => {
    const search = makeSearch()
    const state = createContextSearch({ projectId: () => 'p1', search, limit: 50 })

    state.query.value = 'x'
    await state.run()

    expect(search).toHaveBeenCalledWith({ project_id: 'p1', q: 'x', limit: 50 })
  })

  it('survives a project with no contexts loaded yet', () => {
    // `all` is optional, and the page's fetch may not have resolved when the
    // tab is first opened. This must render empty rather than throw.
    const state = createContextSearch({ projectId: () => 'p1', search: makeSearch() })
    expect(state.displayed.value).toEqual([])
  })
})

describe('extractErrorMessage', () => {
  it('prefers a structured API message', () => {
    expect(extractErrorMessage({ data: { message: 'limit must be an integer' } }, 'fallback'))
      .toBe('limit must be an integer')
  })

  it('falls back to a transport message', () => {
    expect(extractErrorMessage(new Error('Failed to fetch'), 'fallback')).toBe('Failed to fetch')
  })

  it('uses the supplied default when there is nothing to show', () => {
    expect(extractErrorMessage(null, 'Context search failed')).toBe('Context search failed')
    expect(extractErrorMessage({}, 'Context search failed')).toBe('Context search failed')
  })

  it('does not prefer a top-level message over a structured one', () => {
    // $fetch sets both; the API's message is the one written for a user.
    expect(extractErrorMessage({ message: '500', data: { message: 'validation: q is required' } }, 'x'))
      .toBe('validation: q is required')
  })
})
