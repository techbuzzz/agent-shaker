/**
 * The journal reader.
 *
 * The state machine is a factory with the transport injected, exactly like
 * `useHooks`: it holds no I/O of its own, so it can be unit-tested without
 * booting Nuxt.
 */
import { ref, type Ref } from 'vue'
import type { AuditChainHead, AuditEvent, AuditEventsResponse } from '../types/activity'
import { ApiConnectionError } from './useApi'

/** The filter the page applies to a journal read. */
export interface ActivityFilter {
  entityType: string
  entityId: string
  action: string
  limit: number
}

export interface ActivityTransport {
  listEvents: (projectId: string, filter: ActivityFilter) => Promise<AuditEvent[]>
  chainHead: (projectId: string) => Promise<AuditChainHead | null>
}

export interface ActivityStore {
  events: Ref<AuditEvent[]>
  head: Ref<AuditChainHead | null>
  loading: Ref<boolean>
  error: Ref<string | null>
  load: (projectId: string, filter: ActivityFilter) => Promise<void>
}

export function createActivityStore(transport: ActivityTransport): ActivityStore {
  const events = ref<AuditEvent[]>([])
  const head = ref<AuditChainHead | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function load(projectId: string, filter: ActivityFilter): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const [list, chain] = await Promise.all([
        transport.listEvents(projectId, filter),
        transport.chainHead(projectId)
      ])
      events.value = list
      head.value = chain
    } catch (err: unknown) {
      error.value = describeActivityError(err, 'Could not load the journal')
    } finally {
      loading.value = false
    }
  }

  return { events, head, loading, error, load }
}

/**
 * Turns a failed request into one sentence an operator can act on.
 *
 * The Go service answers every non-2xx with an envelope —
 * `{"error":{"code","message","request_id"}}` — so the server's own words live
 * at `data.error.message`.
 */
export function describeActivityError(err: unknown, fallback: string): string {
  const candidate = err as {
    data?: { error?: { message?: string }; message?: string }
    message?: string
    status?: number
    statusCode?: number
  }
  const structured = candidate?.data?.error?.message ?? candidate?.data?.message
  if (structured) return structured
  if (candidate?.statusCode) return `${fallback} (HTTP ${candidate.statusCode})`
  return fallback
}

/** The real transport: the journal endpoints over the app's usual base URL. */
export function createActivityTransport(baseURL: () => string): ActivityTransport {
  async function request<T>(path: string): Promise<T> {
    try {
      return await $fetch(path, { baseURL: baseURL() }) as T
    } catch (err: unknown) {
      const fe = err as { status?: number; statusCode?: number; message?: string }
      if (!fe?.status && !fe?.statusCode) {
        throw new ApiConnectionError(fe?.message || 'Cannot reach MCP server', baseURL())
      }
      throw err
    }
  }

  return {
    listEvents: async (projectId, filter) => {
      const query = new URLSearchParams({ project_id: projectId, limit: String(filter.limit) })
      if (filter.entityType) query.set('entity_type', filter.entityType)
      if (filter.entityId) query.set('entity_id', filter.entityId)
      if (filter.action) query.set('action', filter.action)
      return (await request<AuditEventsResponse>(`/activity?${query.toString()}`)).events
    },
    chainHead: async (projectId) => {
      const head = await request<AuditChainHead>(`/activity/anchor?project_id=${projectId}`)
      return head.head_hash ? head : null
    }
  }
}

export function useActivity(): ActivityStore {
  const api = useApi()
  const transport = createActivityTransport(() => api.apiBase.value)
  return createActivityStore(transport)
}
