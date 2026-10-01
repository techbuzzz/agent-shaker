<script setup lang="ts">
import type { Agent } from '~/types/api'

useHead({ title: 'Agents · Agent Shaker' })

const api = useApi()
const toast = useToastBus()
const search = ref('')
const statusFilter = ref<'all' | Agent['status']>('all')

const { data: agents, status, refresh } = await useFetch<Agent[]>('/agents', {
  baseURL: api.apiBase,
  key: 'agents:all',
  default: () => []
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return (agents.value ?? []).filter((a) => {
    if (statusFilter.value !== 'all' && a.status !== statusFilter.value) return false
    if (!q) return true
    return a.name.toLowerCase().includes(q) || a.role.toLowerCase().includes(q)
  })
})

async function updateStatus(a: Agent, next: Agent['status']) {
  await api.updateAgentStatus(a.id, next)
  await refresh()
  toast.success(`${a.name} marked ${next}`)
}

const deleting = ref<Agent | null>(null)
const deletingOpen = computed({
  get: () => deleting.value !== null,
  set: (v: boolean) => { if (!v) deleting.value = null }
})
async function confirmDelete() {
  if (!deleting.value) return
  await api.deleteAgent(deleting.value.id)
  await refresh()
  deleting.value = null
  toast.success('Agent removed')
}
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
      <div>
        <h1 class="text-2xl font-semibold">Agents</h1>
        <p class="text-muted mt-1">{{ agents?.length ?? 0 }} registered</p>
      </div>
      <div class="flex items-center gap-2">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Search…" size="sm" class="w-full sm:w-64" />
        <USelectMenu
          v-model="statusFilter"
          :items="[
            { label: 'All statuses', value: 'all' },
            { label: 'Active', value: 'active' },
            { label: 'Idle', value: 'idle' },
            { label: 'Offline', value: 'offline' }
          ]"
          value-key="value"
          size="sm"
        />
      </div>
    </div>

    <EmptyState
      v-if="status === 'success' && filtered.length === 0"
      icon="i-lucide-bot"
      title="No agents match"
      description="Adjust the filters or register an agent from a project."
    />

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
      <UCard v-for="a in filtered" :key="a.id" :ui="{ body: 'p-5' }">
        <div class="flex items-start justify-between gap-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2">
              <h3 class="font-semibold truncate">{{ a.name }}</h3>
              <UBadge variant="subtle" size="xs">{{ a.role }}</UBadge>
            </div>
            <p class="text-xs text-muted mt-1">{{ a.team || '—' }}</p>
            <NuxtLink :to="`/projects/${a.project_id}`" class="text-xs text-primary hover:underline">View project</NuxtLink>
          </div>
          <UBadge :color="a.status === 'active' ? 'success' : a.status === 'idle' ? 'warning' : 'neutral'" variant="subtle" size="xs">
            {{ a.status }}
          </UBadge>
        </div>

        <div class="mt-3 pt-3 border-t border-default flex items-center justify-between text-xs">
          <USelectMenu
            :model-value="a.status"
            :items="[
              { label: 'Active', value: 'active' },
              { label: 'Idle', value: 'idle' },
              { label: 'Offline', value: 'offline' }
            ]"
            value-key="value"
            size="xs"
            @update:model-value="(v) => updateStatus(a, v as Agent['status'])"
          />
          <UButton icon="i-lucide-trash" variant="ghost" color="error" size="xs" aria-label="Delete" @click="deleting = a" />
        </div>
      </UCard>
    </div>

    <ConfirmDialog v-model="deletingOpen" :title="`Delete agent ${deleting?.name}?`" confirm-label="Delete" @confirm="confirmDelete" />
  </div>
</template>
