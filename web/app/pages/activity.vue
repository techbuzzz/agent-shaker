<script setup lang="ts">
import type { Project } from '~/types/api'
import type { AuditEvent } from '~/types/activity'

/**
 * The append-only journal.
 *
 * A read-only surface, like the journal it reads. There is nothing here that
 * edits a row, and the anchor is shown because it is the value an operator
 * pins outside the database — the point of the chain is that a trail
 * rewritten inside the database cannot be rewritten with it.
 */
useHead({ title: 'Activity · Agent Shaker' })

const api = useApi()
const settings = useSettingsStore()
const { relativeTime } = useFormatters()

const activity = useActivity()
const { events, head, loading, error, load } = activity

const { data: projects } = await useFetch<Project[]>('/projects', {
  baseURL: api.apiBase,
  key: 'activity:projects',
  default: () => []
})

const selectedProjectId = ref<string | null>(settings.lastProjectId)

const projectId = computed(() => {
  const list = projects.value ?? []
  if (selectedProjectId.value && list.some((p) => p.id === selectedProjectId.value)) {
    return selectedProjectId.value
  }
  return list[0]?.id ?? null
})

const projectItems = computed(() => (projects.value ?? []).map((p) => ({ label: p.name, value: p.id })))
const currentProjectName = computed(
  () => (projects.value ?? []).find((p) => p.id === projectId.value)?.name ?? 'this project'
)

const entityType = ref('')
const entityId = ref('')
const action = ref('')
const limit = ref(50)

function selectProject(value: unknown): void {
  const next = typeof value === 'string' ? value : null
  selectedProjectId.value = next
  settings.rememberProject(next)
}

/**
 * Reads the journal for the current project and filters.
 *
 * Client-only: the endpoints are relative to the app's own origin, and a
 * server-side render cannot resolve them.
 */
async function refresh(): Promise<void> {
  if (!import.meta.client || !projectId.value) return

  await load(projectId.value, {
    entityType: entityType.value.trim(),
    entityId: entityId.value.trim(),
    action: action.value.trim(),
    limit: limit.value
  })
}

// The first read runs after hydration rather than during setup, so the
// server's HTML and the client's first paint agree: loading during setup
// would leave the server showing the empty state while the client shows the
// skeleton.
onMounted(() => { void refresh() })

watch([projectId, entityType, entityId, action, limit], () => { void refresh() })

/** The event's payload, rendered as key and value rather than as JSON. */
function pairs(event: AuditEvent): { key: string; value: string }[] {
  const source = event.after ?? event.before ?? {}
  return Object.entries(source).map(([key, value]) => ({
    key,
    value: typeof value === 'string' ? value : JSON.stringify(value)
  }))
}
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
      <div>
        <h1 class="text-2xl font-semibold">Activity</h1>
        <p class="text-muted mt-1">
          What happened in
          <span class="font-medium text-default">{{ currentProjectName }}</span>, in the order it happened.
        </p>
      </div>
      <USelectMenu
        :model-value="projectId ?? undefined"
        :items="projectItems"
        value-key="value"
        size="sm"
        placeholder="Choose a project"
        class="w-56"
        aria-label="Project"
        @update:model-value="selectProject"
      />
    </div>

    <!--
      The anchor. It is the value an operator pins outside the database: the
      chain head, so that a trail rewritten inside the database cannot be
      rewritten with it.
    -->
    <UCard v-if="head" class="mb-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="min-w-0">
          <p class="font-medium">Chain head</p>
          <p class="text-muted text-sm mt-1">
            The hash of the newest entry. Pin it somewhere you control — the chain breaks if any earlier
            entry is changed or removed.
          </p>
        </div>
        <code class="font-mono text-xs text-muted break-all">{{ head.head_hash }}</code>
      </div>
    </UCard>

    <!-- FILTERS -->
    <div class="grid gap-3 sm:grid-cols-4 mb-6">
      <UInput
        v-model="entityType"
        placeholder="Entity type — task, agent…"
        aria-label="Entity type"
        class="w-full"
      />
      <UInput
        v-model="entityId"
        placeholder="Entity id"
        aria-label="Entity id"
        class="w-full"
      />
      <UInput
        v-model="action"
        placeholder="Action — task.claimed…"
        aria-label="Action"
        class="w-full"
      />
      <UInput
        v-model.number="limit"
        type="number"
        min="1"
        max="500"
        placeholder="How many"
        aria-label="How many entries"
        class="w-full"
      />
    </div>

    <div v-if="loading" class="space-y-3" aria-busy="true">
      <USkeleton v-for="n in 3" :key="n" class="h-24 w-full rounded-xl" />
    </div>

    <div
      v-else-if="error"
      role="alert"
      class="flex items-start gap-3 rounded-lg border border-error/40 bg-error/10 px-4 py-3 text-sm text-error"
    >
      <UIcon name="i-lucide-circle-alert" class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <p class="leading-snug">{{ error }}</p>
    </div>

    <EmptyState
      v-else-if="!events.length"
      icon="i-lucide-scroll-text"
      title="Nothing recorded yet"
      description="The journal records the changes that affect coordination — a task claimed, a context published, an agent going stale."
    />

    <div v-else class="space-y-2">
      <UCard v-for="event in events" :key="event.id" :ui="{ body: 'p-4' }">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <code class="font-mono text-xs text-muted">{{ event.action }}</code>
              <span v-if="event.entity_type" class="text-xs text-muted">
                {{ event.entity_type }}<template v-if="event.entity_id"> {{ event.entity_id }}</template>
              </span>
              <span v-if="event.actor_type" class="text-xs text-muted">by {{ event.actor_type }}</span>
            </div>
            <p class="mt-1 text-xs text-muted">{{ relativeTime(event.occurred_at) }}</p>
          </div>
          <code v-if="event.hash" class="font-mono text-[10px] text-muted break-all max-w-40">{{ event.hash }}</code>
        </div>

        <dl v-if="pairs(event).length" class="mt-2 grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-1 text-xs">
          <div v-for="pair in pairs(event)" :key="pair.key" class="flex gap-2">
            <dt class="font-medium text-default shrink-0">{{ pair.key }}</dt>
            <dd class="text-muted min-w-0 break-all">{{ pair.value }}</dd>
          </div>
        </dl>
      </UCard>
    </div>
  </div>
</template>
