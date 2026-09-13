<script setup lang="ts">
import type { Project, CreateProjectInput } from '~/types/api'

useHead({ title: 'Projects · Agent Shaker' })

const api = useApi()
const projectsStore = useProjectsStore()
const toast = useToastBus()
const search = ref('')

const { data: projects, status, error, refresh } = await useFetch<Project[]>('/projects', {
  baseURL: api.apiBase,
  key: 'projects:list',
  default: () => [],
  transform: (list) => { projectsStore.setRecent(list); return list }
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return projects.value ?? []
  return (projects.value ?? []).filter((p) =>
    p.name.toLowerCase().includes(q) ||
    (p.description ?? '').toLowerCase().includes(q)
  )
})

const showCreate = ref(false)
const editing = ref<Project | null>(null)
const deleting = ref<Project | null>(null)
const deletingOpen = computed({
  get: () => deleting.value !== null,
  set: (v: boolean) => { if (!v) deleting.value = null }
})

async function handleCreate(input: CreateProjectInput) {
  const created = await api.createProject(input)
  projectsStore.upsert(created)
  await refresh()
  toast.success(`Project "${created.name}" created`)
}

async function handleEdit(input: CreateProjectInput) {
  if (!editing.value) return
  // Backend doesn't expose PUT /projects/{id}; only status update. Preserve original status.
  await api.updateProjectStatus(editing.value.id, input.status ?? editing.value.status)
  const updated = await api.getProject(editing.value.id)
  projectsStore.upsert(updated)
  await refresh()
  toast.success(`Project "${updated.name}" updated`)
}

async function handleDelete() {
  if (!deleting.value) return
  await api.deleteProject(deleting.value.id)
  projectsStore.remove(deleting.value.id)
  await refresh()
  toast.success(`Project deleted`)
  deleting.value = null
}
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-6">
      <div>
        <h1 class="text-2xl font-semibold">Projects</h1>
        <p class="text-muted mt-1">{{ projects?.length ?? 0 }} project{{ (projects?.length ?? 0) === 1 ? '' : 's' }}</p>
      </div>
      <div class="flex items-center gap-2">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Search projects…" size="sm" class="w-full sm:w-64" />
        <UButton icon="i-lucide-plus" color="primary" @click="showCreate = true">New project</UButton>
      </div>
    </div>

    <UAlert
      v-if="status === 'error'"
      color="error"
      variant="subtle"
      icon="i-lucide-octagon-x"
      :title="(error as Error)?.message || 'Failed to load projects'"
    />

    <div v-else-if="status === 'pending'" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <USkeleton v-for="i in 6" :key="i" class="h-32" />
    </div>

    <EmptyState
      v-else-if="filtered.length === 0"
      icon="i-lucide-folder-open"
      :title="search ? 'No projects match your search' : 'No projects yet'"
      :description="search ? 'Try a different search term.' : 'Get started by creating your first project.'"
    >
      <UButton v-if="!search" color="primary" @click="showCreate = true">Create a project</UButton>
    </EmptyState>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <ProjectCard
        v-for="p in filtered"
        :key="p.id"
        :project="p"
        @edit="(proj) => { editing = proj; showCreate = true }"
        @delete="(proj) => deleting = proj"
      />
    </div>

    <ProjectFormModal
      v-model="showCreate"
      :initial="editing"
      @submit="editing ? handleEdit($event) : handleCreate($event)"
    />
    <ConfirmDialog
      v-model="deletingOpen"
      :title="`Delete ${deleting?.name}?`"
      description="This permanently removes the project and all its agents, tasks, contexts, and standups."
      confirm-label="Delete project"
      icon="i-lucide-trash"
      @confirm="handleDelete"
    />
  </div>
</template>
