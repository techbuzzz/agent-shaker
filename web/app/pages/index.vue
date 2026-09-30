<script setup lang="ts">
import type { DashboardStats, Project, Agent, Task } from '~/types/api'

useHead({ title: 'Dashboard · Agent Shaker' })

const api = useApi()
const projectsStore = useProjectsStore()
const toast = useToastBus()

// Dashboard stats
const { data: stats, status: statsStatus, error: statsError, refresh: refreshStats } =
  await useFetch<DashboardStats>('/dashboard', {
    baseURL: api.apiBase,
    key: 'dashboard:stats',
    default: () => ({
      projects: { total: 0, active: 0, archived: 0 },
      agents:   { total: 0, active: 0, idle: 0, offline: 0 },
      tasks:    { total: 0, pending: 0, in_progress: 0, done: 0, blocked: 0 },
      contexts: { total: 0 }
    })
  })

// Recent projects (top 5)
const { data: projects, status: projectsStatus } = await useFetch<Project[]>('/projects', {
  baseURL: api.apiBase,
  key: 'dashboard:projects',
  transform: (list) => {
    projectsStore.setRecent(list)
    return list
  }
})
const recentProjects = computed(() => (projects.value ?? []).slice(0, 5))

// Active agents (top 5)
const { data: agents, status: agentsStatus } = await useFetch<Agent[]>('/agents', {
  baseURL: api.apiBase,
  key: 'dashboard:agents',
  default: () => []
})
const activeAgents = computed(() =>
  (agents.value ?? []).filter((a) => a.status === 'active').slice(0, 5)
)

// Recent tasks (top 5 by updated_at)
const { data: tasks, status: tasksStatus } = await useFetch<Task[]>('/tasks', {
  baseURL: api.apiBase,
  key: 'dashboard:tasks',
  default: () => []
})
const recentTasks = computed(() =>
  [...(tasks.value ?? [])]
    .sort((a, b) => +new Date(b.updated_at) - +new Date(a.updated_at))
    .slice(0, 5)
)

const errorBanner = computed(() => {
  if (statsStatus.value === 'error' && statsError.value) {
    return (statsError.value as Error).message || 'Failed to load dashboard'
  }
  return null
})

async function refresh() {
  await Promise.all([refreshStats(), refreshNuxtData(['dashboard:projects', 'dashboard:agents', 'dashboard:tasks'])])
  toast.success('Dashboard refreshed')
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-semibold flex items-center gap-3">
          <span class="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center text-white">
            <UIcon name="i-lucide-layout-dashboard" class="w-5 h-5" />
          </span>
          Dashboard
        </h1>
        <p class="text-muted mt-1 ml-13">Overview of your MCP Task Tracker.</p>
      </div>
      <UButton icon="i-lucide-refresh-cw" variant="ghost" color="neutral" @click="refresh">Refresh</UButton>
    </div>

    <UAlert
      v-if="errorBanner"
      color="error"
      variant="subtle"
      icon="i-lucide-octagon-x"
      :title="errorBanner"
      class="mb-6"
    >
      <template #actions>
        <UButton to="/settings" variant="outline" color="error" size="sm">Open Settings</UButton>
      </template>
    </UAlert>

    <!-- Stat cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <StatCard
        title="Projects"
        :value="stats!.projects.total"
        icon="i-lucide-folder-kanban"
        accent="primary"
        :breakdown="[
          { label: 'Active', value: stats!.projects.active, color: '#10b981' },
          { label: 'Archived', value: stats!.projects.archived, color: '#94a3b8' }
        ]"
        href="/projects"
      />
      <StatCard
        title="Agents"
        :value="stats!.agents.total"
        icon="i-lucide-bot"
        accent="success"
        :breakdown="[
          { label: 'Active', value: stats!.agents.active, color: '#10b981' },
          { label: 'Idle', value: stats!.agents.idle, color: '#f59e0b' },
          { label: 'Offline', value: stats!.agents.offline, color: '#94a3b8' }
        ]"
        href="/agents"
      />
      <StatCard
        title="Tasks"
        :value="stats!.tasks.total"
        icon="i-lucide-list-checks"
        accent="warning"
        :breakdown="[
          { label: 'Pending', value: stats!.tasks.pending, color: '#94a3b8' },
          { label: 'In progress', value: stats!.tasks.in_progress, color: '#3b82f6' },
          { label: 'Blocked', value: stats!.tasks.blocked, color: '#ef4444' }
        ]"
        href="/tasks"
      />
      <StatCard
        title="Documentation"
        :value="stats!.contexts.total"
        icon="i-lucide-book-text"
        accent="info"
        :breakdown="[
          { label: 'Done tasks', value: stats!.tasks.done, color: '#10b981' }
        ]"
        href="/documentation"
      />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Recent projects -->
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h2 class="text-base font-semibold flex items-center gap-2">
              <UIcon name="i-lucide-folder-kanban" class="w-4 h-4" /> Recent Projects
            </h2>
            <UButton to="/projects" variant="ghost" size="xs" trailing-icon="i-lucide-arrow-right">All</UButton>
          </div>
        </template>
        <div v-if="projectsStatus === 'pending'" class="py-8 text-center text-sm text-muted">
          <UIcon name="i-lucide-loader" class="w-5 h-5 animate-spin mx-auto" />
        </div>
        <EmptyState v-else-if="recentProjects.length === 0" icon="i-lucide-folder-open" title="No projects yet" description="Get started by creating your first project.">
          <UButton to="/projects" color="primary" size="sm">Create a project</UButton>
        </EmptyState>
        <ul v-else class="divide-y divide-default">
          <li v-for="p in recentProjects" :key="p.id" class="py-3 first:pt-0 last:pb-0">
            <NuxtLink :to="`/projects/${p.id}`" class="flex items-center justify-between gap-2 group">
              <div class="min-w-0">
                <div class="font-medium truncate group-hover:text-primary transition-colors">{{ p.name }}</div>
                <div class="text-xs text-muted truncate">{{ p.description || '—' }}</div>
              </div>
              <UBadge :color="p.status === 'active' ? 'success' : 'neutral'" variant="subtle" size="xs">{{ p.status }}</UBadge>
            </NuxtLink>
          </li>
        </ul>
      </UCard>

      <!-- Active agents -->
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h2 class="text-base font-semibold flex items-center gap-2">
              <UIcon name="i-lucide-bot" class="w-4 h-4" /> Active Agents
            </h2>
            <UButton to="/agents" variant="ghost" size="xs" trailing-icon="i-lucide-arrow-right">All</UButton>
          </div>
        </template>
        <div v-if="agentsStatus === 'pending'" class="py-8 text-center text-sm text-muted">
          <UIcon name="i-lucide-loader" class="w-5 h-5 animate-spin mx-auto" />
        </div>
        <EmptyState v-else-if="activeAgents.length === 0" icon="i-lucide-bot" title="No active agents" description="Agents will appear here once they're online." />
        <ul v-else class="space-y-2">
          <li v-for="a in activeAgents" :key="a.id" class="flex items-center justify-between gap-2">
            <div class="min-w-0">
              <div class="font-medium truncate">{{ a.name }}</div>
              <div class="text-xs text-muted truncate">{{ a.role }} · {{ a.team || '—' }}</div>
            </div>
            <span class="status-dot status-dot--online" />
          </li>
        </ul>
      </UCard>

      <!-- Recent tasks -->
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h2 class="text-base font-semibold flex items-center gap-2">
              <UIcon name="i-lucide-list-checks" class="w-4 h-4" /> Recent Tasks
            </h2>
            <UButton to="/tasks" variant="ghost" size="xs" trailing-icon="i-lucide-arrow-right">All</UButton>
          </div>
        </template>
        <div v-if="tasksStatus === 'pending'" class="py-8 text-center text-sm text-muted">
          <UIcon name="i-lucide-loader" class="w-5 h-5 animate-spin mx-auto" />
        </div>
        <EmptyState v-else-if="recentTasks.length === 0" icon="i-lucide-list-checks" title="No tasks yet" description="Tasks will appear here as soon as they're created." />
        <ul v-else class="space-y-2">
          <li v-for="t in recentTasks" :key="t.id" class="flex items-start gap-2">
            <div class="flex-1 min-w-0">
              <div class="font-medium truncate">{{ t.title }}</div>
              <div class="text-xs text-muted">{{ t.priority }} · {{ t.status.replace('_', ' ') }}</div>
            </div>
          </li>
        </ul>
      </UCard>
    </div>
  </div>
</template>
