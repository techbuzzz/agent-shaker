<script setup lang="ts">
import type { Standup, Agent, Project } from '~/types/api'

useHead({ title: 'Standups · Agent Shaker' })

const api = useApi()
const toast = useToastBus()
const projectFilter = ref<string>('')
const agentFilter = ref<string>('')

const { data: standups, status, refresh } = await useFetch<Standup[]>('/standups', {
  baseURL: api.apiBase,
  key: 'standups:all',
  default: () => []
})

const { data: agents } = await useFetch<Agent[]>('/agents', {
  baseURL: api.apiBase,
  key: 'standups:agents',
  default: () => []
})
const agentNames = computed(() => Object.fromEntries((agents.value ?? []).map((a) => [a.id, a.name])))

const { data: projects } = await useFetch<Project[]>('/projects', {
  baseURL: api.apiBase,
  key: 'standups:projects',
  default: () => []
})
const projectOptions = computed(() => [
  { label: 'All projects', value: '' },
  ...(projects.value ?? []).map((p) => ({ label: p.name, value: p.id }))
])

const filtered = computed(() => (standups.value ?? []).filter((s) => {
  if (projectFilter.value && s.project_id !== projectFilter.value) return false
  if (agentFilter.value && s.agent_id !== agentFilter.value) return false
  return true
}))

async function confirmDelete(s: Standup) {
  await api.deleteStandup(s.id)
  await refresh()
  toast.success('Standup removed')
}
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
      <div>
        <h1 class="text-2xl font-semibold">Daily Standups</h1>
        <p class="text-muted mt-1">{{ standups?.length ?? 0 }} submissions</p>
      </div>
    </div>

    <div class="bg-elevated border border-default rounded-lg p-4 mb-6">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <UFormField label="Project">
          <USelect v-model="projectFilter" :items="projectOptions" value-key="value" />
        </UFormField>
        <UFormField label="Agent">
          <USelect
            v-model="agentFilter"
            :items="[{ label: 'All agents', value: '' }, ...(agents ?? []).map(a => ({ label: a.name, value: a.id }))]"
            value-key="value"
          />
        </UFormField>
      </div>
    </div>

    <EmptyState
      v-if="status === 'success' && filtered.length === 0"
      icon="i-lucide-calendar-days"
      title="No standups match"
      description="Submit a standup from a project's Standups tab."
    />

    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <StandupCard
        v-for="s in filtered" :key="s.id" :standup="s" :agent-name="agentNames[s.agent_id]"
        @delete="confirmDelete"
      />
    </div>
  </div>
</template>
