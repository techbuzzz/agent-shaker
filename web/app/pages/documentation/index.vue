<script setup lang="ts">
import type { Context, Agent } from '~/types/api'

useHead({ title: 'Documentation · Agent Shaker' })

const api = useApi()
const toast = useToastBus()
const search = ref('')

const { data: contexts, status, refresh } = await useFetch<Context[]>('/contexts', {
  baseURL: api.apiBase,
  key: 'contexts:all',
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
        <p class="text-muted mt-1">{{ contexts?.length ?? 0 }} documents</p>
      </div>
      <UInput v-model="search" icon="i-lucide-search" placeholder="Search title, content, tags…" size="sm" class="w-full sm:w-64" />
    </div>

    <EmptyState
      v-if="status === 'success' && filtered.length === 0"
      icon="i-lucide-book-text"
      title="No documentation yet"
      description="Share context from a project's Documentation tab."
    />

    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
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
