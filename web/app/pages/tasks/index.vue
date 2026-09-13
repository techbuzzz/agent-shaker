<script setup lang="ts">
import type { Task, TaskStatus, TaskPriority } from '~/types/api'

useHead({ title: 'Tasks · Agent Shaker' })

const api = useApi()
const toast = useToastBus()
const search = ref('')
const statusFilter = ref<'all' | TaskStatus>('all')
const priorityFilter = ref<'all' | TaskPriority>('all')

const { data: tasks, status, refresh } = await useFetch<Task[]>('/tasks', {
  baseURL: api.apiBase,
  key: 'tasks:all',
  default: () => []
})

const projectsStore = useProjectsStore()
const projectNames = computed(() => Object.fromEntries(projectsStore.recent.map((p) => [p.id, p.name])))

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return (tasks.value ?? []).filter((t) => {
    if (statusFilter.value !== 'all' && t.status !== statusFilter.value) return false
    if (priorityFilter.value !== 'all' && t.priority !== priorityFilter.value) return false
    if (!q) return true
    return t.title.toLowerCase().includes(q) || (t.description ?? '').toLowerCase().includes(q)
  })
})

async function updateStatus({ task, status }: { task: Task; status: TaskStatus }) {
  await api.updateTaskStatus(task.id, status)
  await refresh()
  toast.success(`Task moved to ${status.replace('_', ' ')}`)
}

const deleting = ref<Task | null>(null)
const deletingOpen = computed({
  get: () => deleting.value !== null,
  set: (v: boolean) => { if (!v) deleting.value = null }
})
async function confirmDelete() {
  if (!deleting.value) return
  await api.deleteTask(deleting.value.id)
  await refresh()
  deleting.value = null
  toast.success('Task removed')
}
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
      <div>
        <h1 class="text-2xl font-semibold">Tasks</h1>
        <p class="text-muted mt-1">{{ tasks?.length ?? 0 }} total</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Search…" size="sm" class="w-full sm:w-64" />
        <USelectMenu
          v-model="statusFilter"
          :items="[
            { label: 'All statuses', value: 'all' },
            { label: 'Pending', value: 'pending' },
            { label: 'In progress', value: 'in_progress' },
            { label: 'Blocked', value: 'blocked' },
            { label: 'Done', value: 'done' },
            { label: 'Cancelled', value: 'cancelled' }
          ]"
          value-key="value"
          size="sm"
        />
        <USelectMenu
          v-model="priorityFilter"
          :items="[
            { label: 'All priorities', value: 'all' },
            { label: 'Low', value: 'low' },
            { label: 'Medium', value: 'medium' },
            { label: 'High', value: 'high' }
          ]"
          value-key="value"
          size="sm"
        />
      </div>
    </div>

    <EmptyState
      v-if="status === 'success' && filtered.length === 0"
      icon="i-lucide-list-checks"
      title="No tasks match"
      description="Adjust filters or create a new task from a project."
    />

    <div v-else class="space-y-3">
      <TaskCard
        v-for="t in filtered" :key="t.id" :task="t" :project-name="projectNames[t.project_id]"
        @change-status="updateStatus"
        @delete="(tk) => deleting = tk"
      />
    </div>

    <ConfirmDialog v-model="deletingOpen" :title="`Delete task ${deleting?.title}?`" confirm-label="Delete" @confirm="confirmDelete" />
  </div>
</template>
