<script setup lang="ts">
import type { Context, Agent, GlobalContext } from '~/types/api'

useHead({ title: 'Documentation · Agent Shaker' })

const api = useApi()
const toast = useToastBus()
const search = ref('')

const { data: contexts, status, refresh } = await useFetch<Context[]>('/contexts', {
  baseURL: api.apiBase,
  key: 'contexts:all',
  default: () => []
})

const { data: globals, refresh: refreshGlobal } = await useFetch<GlobalContext[]>('/global_contexts', {
  baseURL: api.apiBase,
  key: 'gctx:global:all',
  params: { scope: 'global' },
  default: () => []
})

const { data: agents } = await useFetch<Agent[]>('/agents', {
  baseURL: api.apiBase,
  key: 'contexts:agents',
  default: () => []
})
const agentNames = computed(() => Object.fromEntries((agents.value ?? []).map((a) => [a.id, a.name])))

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return contexts.value ?? []
  return (contexts.value ?? []).filter((c) =>
    c.title.toLowerCase().includes(q) ||
    c.content.toLowerCase().includes(q) ||
    c.tags.some((t) => t.toLowerCase().includes(q))
  )
})

const viewing = ref<Context | null>(null)
const deleting = ref<Context | null>(null)
const viewingOpen = computed({
  get: () => viewing.value !== null,
  set: (v: boolean) => { if (!v) viewing.value = null }
})
const deletingOpen = computed({
  get: () => deleting.value !== null,
  set: (v: boolean) => { if (!v) deleting.value = null }
})

async function confirmDelete() {
  if (!deleting.value) return
  await api.deleteContext(deleting.value.id)
  await refresh()
  deleting.value = null
  toast.success('Context removed')
}
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
      <div>
        <h1 class="text-2xl font-semibold">Documentation</h1>
        <p class="text-muted mt-1">{{ contexts?.length ?? 0 }} project docs · {{ globals?.length ?? 0 }} global</p>
      </div>
      <UInput v-model="search" icon="i-lucide-search" placeholder="Search title, content, tags…" size="sm" class="w-full sm:w-64" />
    </div>

    <div v-if="globals?.length" class="mb-8">
      <div class="flex items-center gap-2 mb-2">
        <UIcon name="i-lucide-globe" class="text-primary w-4 h-4" />
        <h2 class="text-sm font-semibold uppercase tracking-wider text-muted">Global playbooks</h2>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <GlobalContextCard
          v-for="g in globals" :key="g.id" :context="g" :agent-name="agentNames[g.agent_id]"
          @view="(g) => viewing = { id: g.id, project_id: g.project_id ?? '', agent_id: g.agent_id, title: g.title, content: g.content, tags: g.tags, created_at: g.created_at }"
          @delete="(g) => api.deleteGlobalContext(g.id).then(() => refreshGlobal())"
        />
      </div>
    </div>

    <EmptyState
      v-if="status === 'success' && filtered.length === 0 && !globals?.length"
      icon="i-lucide-book-text"
      title="No documentation yet"
      description="Share context from a project's Documentation tab."
    />

    <div v-if="filtered.length" class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <ContextCard
        v-for="c in filtered" :key="c.id" :context="c" :agent-name="agentNames[c.agent_id]"
        @view="(c) => viewing = c"
        @delete="(c) => deleting = c"
      />
    </div>

    <ContextViewer v-model="viewingOpen" :context="viewing" :agent-name="viewing ? agentNames[viewing.agent_id] : undefined" />
    <ConfirmDialog v-model="deletingOpen" :title="`Delete context ${deleting?.title}?`" confirm-label="Delete" @confirm="confirmDelete" />
  </div>
</template>
