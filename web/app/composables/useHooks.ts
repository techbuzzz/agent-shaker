/**
 * Hook layer client — the browser's half of the server-side hook runner.
 *
 * A hook is a command the server runs when a domain event happens. This file
 * owns the two halves of that: the transport that talks to the hook endpoints,
 * and the state machine the page renders.
 *
 * Why the transport lives here rather than in `useApi`: the hook endpoints are
 * new and `useApi` is not part of this change. What is reused is the pattern —
 * the same `$fetch` over the same base URL, the same connection-error
 * handling, and the same error envelope — so the two halves of the app cannot
 * drift apart in how they talk to the server or how they report a failure.
 *
 * The state machine is a factory with the transport injected, exactly like
 * `createContextSearch`: it holds no I/O of its own, so it can be unit-tested
 * without booting Nuxt. `useHooks()` is the thin binding that wires the real
 * transport in.
 */
import { ref, type Ref } from 'vue'
import type {
  CreateHookInput,
  Hook,
  HookRun,
  HookRunsResponse,
  HookRunResponse,
  HookResponse,
  HooksResponse,
  HookSettingsResponse,
  UpdateHookInput
} from '../types/hooks'
import { ApiConnectionError } from './useApi'

/**
 * The endpoints this client speaks to.
 *
 * Injected into the store so the store can be tested against a fake, and so
 * the endpoint list is written down in one place instead of being spread over
 * the page.
 */
export interface HooksTransport {
  listHooks: (projectId: string) => Promise<Hook[]>
  createHook: (projectId: string, input: CreateHookInput) => Promise<Hook>
  updateHook: (id: string, input: UpdateHookInput) => Promise<Hook>
  deleteHook: (id: string) => Promise<void>
  listHookRuns: (projectId: string) => Promise<HookRun[]>
  /** A dry run against a synthetic payload. It does not touch real work. */
  testHook: (id: string) => Promise<HookRun>
  /** Whether the server runs this project's hooks. */
  getHookSettings: (projectId: string) => Promise<boolean>
  /** Turns the project's hook layer on or off. */
  setHookSettings: (projectId: string, enabled: boolean) => Promise<boolean>
}

/**
 * Turns a failed request into one sentence an operator can act on.
 *
 * The Go service answers every non-2xx with an envelope —
 * `{"error":{"code","message","request_id"}}` (internal/httpx/errors.go) — so
 * the server's own words live at `data.error.message`. The flat `data.message`
 * shape is also accepted because that is what `useContextSearch` sees from the
 * same API, and ofetch puts its own transport noise in `message`, which is
 * stripped the way `describeSubmitError` strips it.
 */
export function describeHookError(err: unknown, fallback: string): string {
  const candidate = err as {
    data?: { error?: { message?: string }; message?: string }
    message?: string
    status?: number
    statusCode?: number
  }
  const structured = candidate?.data?.error?.message ?? candidate?.data?.message
  if (structured) return structured
  const transport = candidate?.message
  if (transport && !/^\[?(GET|POST|PUT|PATCH|DELETE)/i.test(transport)) return transport
  const status = candidate?.statusCode ?? candidate?.status
  if (status) return `${fallback} (HTTP ${status})`
  return fallback
}

/**
 * Newest run first.
 *
 * The server orders by `started_at DESC`, but the list is rendered from
 * whatever arrived plus whatever the operator just triggered, so the order is
 * re-established here rather than assumed. A run with an unreadable timestamp
 * sinks to the bottom instead of jumping to the top.
 */
export function sortRuns(runs: HookRun[]): HookRun[] {
  return [...runs].sort((a, b) => {
    const left = new Date(a.started_at).getTime()
    const right = new Date(b.started_at).getTime()
    if (Number.isNaN(left) && Number.isNaN(right)) return 0
    if (Number.isNaN(left)) return 1
    if (Number.isNaN(right)) return -1
    return right - left
  })
}

export interface HooksStore {
  hooks: Ref<Hook[]>
  runs: Ref<HookRun[]>
  /**
   * Whether the server runs this project's hooks.
   *
   * It is a separate switch from each hook's own `enabled`: the project
   * decides whether commands may run at all, and each hook decides whether it
   * runs when they do.
   */
  projectEnabled: Ref<boolean>
  loading: Ref<boolean>
  /** A write is in flight. */
  saving: Ref<boolean>
  /** The hook whose test run is in flight, so only its button waits. */
  testingId: Ref<string | null>
  /** The list could not be loaded. */
  loadError: Ref<string | null>
  /** The run log could not be loaded. */
  runError: Ref<string | null>
  /** The last write that failed, whatever it was. */
  actionError: Ref<string | null>
  /** The result of the most recent test run. */
  lastTestRun: Ref<HookRun | null>
  load: (projectId: string) => Promise<void>
  create: (projectId: string, input: CreateHookInput) => Promise<Hook>
  update: (id: string, input: UpdateHookInput) => Promise<Hook>
  /** Flips `enabled` without deleting the hook. */
  toggle: (hook: Hook) => Promise<Hook>
  remove: (id: string) => Promise<void>
  test: (id: string) => Promise<HookRun>
  /** Turns the project's hook layer on or off. */
  setProjectEnabled: (projectId: string, enabled: boolean) => Promise<void>
}

export interface HooksStoreOptions {
  transport: HooksTransport
}

/**
 * The hook state machine.
 *
 * Writes re-throw after recording the message, because the form dialog needs
 * the rejection to keep its own error banner on screen — a failed create that
 * silently closed the dialog would leave the operator with nothing to act on.
 */
export function createHooksStore(options: HooksStoreOptions): HooksStore {
  const { transport } = options

  const hooks = ref<Hook[]>([])
  const runs = ref<HookRun[]>([])
  const projectEnabled = ref(false)
  const loading = ref(false)
  const saving = ref(false)
  const testingId = ref<string | null>(null)
  const loadError = ref<string | null>(null)
  const runError = ref<string | null>(null)
  const actionError = ref<string | null>(null)
  const lastTestRun = ref<HookRun | null>(null)

  /**
   * The hook list and the run log are fetched separately and fail separately.
   *
   * They are separate resources with separate endpoints; a server that answers
   * the hook list but not the run log should still show the hooks. One failed
   * promise must not empty the other list.
   */
  async function load(projectId: string): Promise<void> {
    loading.value = true
    loadError.value = null
    runError.value = null
    try {
      const [hooksResult, runsResult, settingsResult] = await Promise.allSettled([
        transport.listHooks(projectId),
        transport.listHookRuns(projectId),
        transport.getHookSettings(projectId)
      ])

      if (hooksResult.status === 'fulfilled') {
        hooks.value = hooksResult.value
      } else {
        loadError.value = describeHookError(hooksResult.reason, 'Could not load the hooks')
      }

      if (runsResult.status === 'fulfilled') {
        runs.value = sortRuns(runsResult.value)
      } else {
        runError.value = describeHookError(runsResult.reason, 'Could not load the run log')
      }

      if (settingsResult.status === 'fulfilled') {
        projectEnabled.value = settingsResult.value
      }
      // A failed settings read is not an error worth a banner: the switch's
      // value is only advisory until it is written, and the list of hooks is
      // what the operator came for.
    } finally {
      loading.value = false
    }
  }

  function replaceHook(hook: Hook): void {
    const index = hooks.value.findIndex((h) => h.id === hook.id)
    if (index === -1) {
      hooks.value = [hook, ...hooks.value]
      return
    }
    hooks.value.splice(index, 1, hook)
  }

  async function create(projectId: string, input: CreateHookInput): Promise<Hook> {
    saving.value = true
    actionError.value = null
    try {
      const hook = await transport.createHook(projectId, input)
      hooks.value = [hook, ...hooks.value]
      return hook
    } catch (err: unknown) {
      actionError.value = describeHookError(err, 'Could not create the hook')
      throw err
    } finally {
      saving.value = false
    }
  }

  async function update(id: string, input: UpdateHookInput): Promise<Hook> {
    saving.value = true
    actionError.value = null
    try {
      const hook = await transport.updateHook(id, input)
      replaceHook(hook)
      return hook
    } catch (err: unknown) {
      actionError.value = describeHookError(err, 'Could not update the hook')
      throw err
    } finally {
      saving.value = false
    }
  }

  async function toggle(hook: Hook): Promise<Hook> {
    return update(hook.id, { enabled: !hook.enabled })
  }

  async function remove(id: string): Promise<void> {
    saving.value = true
    actionError.value = null
    try {
      await transport.deleteHook(id)
      hooks.value = hooks.value.filter((h) => h.id !== id)
      // The runs are deliberately left alone: the log outlives the hook it
      // belongs to, because "this hook ran twice and was then deleted" is
      // exactly the row an operator needs.
    } catch (err: unknown) {
      actionError.value = describeHookError(err, 'Could not delete the hook')
      throw err
    } finally {
      saving.value = false
    }
  }

  async function test(id: string): Promise<HookRun> {
    testingId.value = id
    actionError.value = null
    try {
      const run = await transport.testHook(id)
      lastTestRun.value = run
      runs.value = sortRuns([run, ...runs.value])
      return run
    } catch (err: unknown) {
      actionError.value = describeHookError(err, 'Could not run the test')
      throw err
    } finally {
      testingId.value = null
    }
  }

  async function setProjectEnabled(projectId: string, enabled: boolean): Promise<void> {
    saving.value = true
    actionError.value = null
    try {
      projectEnabled.value = await transport.setHookSettings(projectId, enabled)
    } catch (err: unknown) {
      actionError.value = describeHookError(err, 'Could not change the project hook setting')
      throw err
    } finally {
      saving.value = false
    }
  }

  return {
    hooks,
    runs,
    projectEnabled,
    loading,
    saving,
    testingId,
    loadError,
    runError,
    actionError,
    lastTestRun,
    load,
    create,
    update,
    toggle,
    remove,
    test,
    setProjectEnabled
  }
}

/**
 * The real transport: the hook endpoints over the app's usual base URL.
 *
 * The base URL is read per call rather than captured, because it comes from a
 * cookie and a captured string would go stale the moment the operator points
 * the app at a different backend from Settings.
 */
export function createHooksTransport(baseURL: () => string): HooksTransport {
  async function request<T>(path: string, opts: Record<string, unknown> = {}): Promise<T> {
    try {
      return await $fetch(path, { baseURL: baseURL(), ...opts }) as T
    } catch (err: unknown) {
      const fe = err as { status?: number; statusCode?: number; message?: string }
      // No status means the request never reached the server. The same
      // distinction `useApi` makes: a refused request keeps its own message,
      // an unreachable server becomes a connection error.
      if (!fe?.status && !fe?.statusCode) {
        throw new ApiConnectionError(fe?.message || 'Cannot reach MCP server', baseURL())
      }
      throw err
    }
  }

  return {
    listHooks: async (projectId) =>
      (await request<HooksResponse>(`/projects/${projectId}/hooks`)).hooks,
    createHook: async (projectId, input) =>
      (await request<HookResponse>(`/projects/${projectId}/hooks`, { method: 'POST', body: input })).hook,
    updateHook: async (id, input) =>
      (await request<HookResponse>(`/hooks/${id}`, { method: 'PATCH', body: input })).hook,
    deleteHook: async (id) => {
      await request<void>(`/hooks/${id}`, { method: 'DELETE' })
    },
    listHookRuns: async (projectId) =>
      (await request<HookRunsResponse>(`/projects/${projectId}/hook-runs`)).runs,
    testHook: async (id) =>
      (await request<HookRunResponse>(`/hooks/${id}/test`, { method: 'POST' })).run,
    getHookSettings: async (projectId) =>
      (await request<HookSettingsResponse>(`/projects/${projectId}/hook-settings`)).enabled,
    setHookSettings: async (projectId, enabled) =>
      (await request<HookSettingsResponse>(`/projects/${projectId}/hook-settings`, {
        method: 'PUT',
        body: { enabled }
      })).enabled
  }
}

export type UseHooks = HooksStore

/**
 * The page-facing composable.
 *
 * State is per call rather than shared: a hook list belongs to one project on
 * one page, and a module-level singleton would leak one project's hooks into
 * the next.
 */
export function useHooks(): UseHooks {
  const api = useApi()
  const transport = createHooksTransport(() => api.apiBase.value)
  return createHooksStore({ transport })
}
