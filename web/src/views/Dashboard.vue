<template>
  <div class="animate-fade-in">
    <div class="mb-8">
      <div class="flex items-center gap-3 mb-2">
        <div class="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center">
          <span class="text-white text-lg">📊</span>
        </div>
        <h2 class="text-3xl font-bold bg-gradient-to-r from-slate-900 to-slate-600 bg-clip-text text-transparent">
          Dashboard
        </h2>
      </div>
      <p class="text-slate-600">Overview of your MCP Task Tracker</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <div class="card group cursor-pointer transform hover:scale-105 transition-all duration-200">
        <div class="flex items-center gap-4">
          <div class="w-12 h-12 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform duration-200">
            <span class="text-2xl">📁</span>
          </div>
          <div>
            <h3 class="text-2xl font-bold text-blue-600">{{ projects.length }}</h3>
            <p class="text-slate-600 text-sm font-medium">Projects</p>
          </div>
        </div>
      </div>

      <div class="card group cursor-pointer transform hover:scale-105 transition-all duration-200">
        <div class="flex items-center gap-4">
          <div class="w-12 h-12 bg-gradient-to-br from-green-500 to-green-600 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform duration-200">
            <span class="text-2xl">🤖</span>
          </div>
          <div>
            <h3 class="text-2xl font-bold text-green-600">{{ agents.length }}</h3>
            <p class="text-slate-600 text-sm font-medium">Agents</p>
          </div>
        </div>
      </div>

      <div class="card group cursor-pointer transform hover:scale-105 transition-all duration-200">
        <div class="flex items-center gap-4">
          <div class="w-12 h-12 bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform duration-200">
            <span class="text-2xl">📋</span>
          </div>
          <div>
            <h3 class="text-2xl font-bold text-purple-600">{{ tasks.length }}</h3>
            <p class="text-slate-600 text-sm font-medium">Total Tasks</p>
          </div>
        </div>
      </div>

      <div class="card group cursor-pointer transform hover:scale-105 transition-all duration-200">
        <div class="flex items-center gap-4">
          <div class="w-12 h-12 bg-gradient-to-br from-emerald-500 to-emerald-600 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform duration-200">
            <span class="text-2xl">✅</span>
          </div>
          <div>
            <h3 class="text-2xl font-bold text-emerald-600">{{ completedTasks }}</h3>
            <p class="text-slate-600 text-sm font-medium">Completed</p>
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div class="card">
        <div class="flex items-center gap-3 mb-6">
          <div class="w-8 h-8 bg-gradient-to-br from-blue-500 to-blue-600 rounded-lg flex items-center justify-center">
            <span class="text-white text-sm">📁</span>
          </div>
          <h3 class="text-xl font-semibold text-slate-900">Recent Projects</h3>
        </div>
        <div class="space-y-3">
          <div v-for="project in recentProjects" :key="project.id" class="group rounded-xl overflow-hidden border border-slate-200 hover:border-slate-300 transition-all duration-200">
            <router-link :to="`/projects/${project.id}`" class="block p-4 hover:bg-slate-50 transition-colors">
              <div class="flex items-start justify-between">
                <div class="flex-1">
                  <h4 class="font-semibold text-slate-900 group-hover:text-blue-600 transition-colors mb-1">{{ project.name }}</h4>
                  <p class="text-slate-600 text-sm mb-2 line-clamp-2">{{ project.description }}</p>
                  <div class="flex items-center gap-2">
                    <span :class=" [
                      'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
                      project.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-800'
                    ]">
                      {{ project.status }}
                    </span>
                    <span class="text-xs text-slate-500">{{ formatDate(project.created_at) }}</span>
                  </div>
                </div>
                <div class="ml-3 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                  <svg class="w-5 h-5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
                  </svg>
                </div>
              </div>
            </router-link>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="flex items-center gap-3 mb-6">
          <div class="w-8 h-8 bg-gradient-to-br from-green-500 to-green-600 rounded-lg flex items-center justify-center">
            <span class="text-white text-sm">🤖</span>
          </div>
          <h3 class="text-xl font-semibold text-slate-900">Active Agents</h3>
        </div>
        <div class="space-y-3">
          <div v-for="agent in activeAgents" :key="agent.id" class="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
            <div class="flex items-center gap-3">
              <span class="font-medium">{{ agent.name }}</span>
              <span :class=" [
                'px-2 py-1 rounded text-xs font-semibold',
                agent.role === 'frontend' ? 'bg-blue-100 text-blue-800' : 'bg-pink-100 text-pink-800'
              ]">{{ agent.role }}</span>
            </div>
            <span :class=" [
              'px-3 py-1 rounded-full text-sm font-semibold',
              agent.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
            ]">{{ agent.status }}</span>
          </div>
          <p v-if="activeAgents.length === 0" class="text-center py-12 text-gray-500">No active agents</p>
        </div>
      </div>

      <div class="col-span-full bg-white p-6 rounded-lg shadow-sm">
        <h3 class="text-xl font-semibold text-gray-900 mb-4">Recent Tasks</h3>
        <div class="space-y-3">
          <div v-for="task in recentTasks" :key="task.id" class="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
            <div class="flex-1">
              <h4 class="font-medium">{{ task.title }}</h4>
              <p class="text-gray-600 text-sm">{{ task.description }}</p>
            </div>
            <div class="flex gap-2">
              <span :class=" [
                'px-2 py-1 rounded text-xs font-semibold',
                task.priority === 'high' ? 'bg-red-100 text-red-800' : 
                task.priority === 'medium' ? 'bg-yellow-100 text-yellow-800' : 'bg-blue-100 text-blue-800'
              ]">{{ task.priority }}</span>
              <span :class=" [
                'px-2 py-1 rounded text-xs font-semibold',
                task.status === 'done' ? 'bg-green-100 text-green-800' : 
                task.status === 'in_progress' ? 'bg-blue-100 text-blue-800' : 
                task.status === 'pending' ? 'bg-gray-100 text-gray-800' : 'bg-red-100 text-red-800'
              ]">{{ task.status }}</span>
            </div>
          </div>
          <p v-if="tasks.length === 0" class="text-center py-12 text-gray-500">No tasks yet</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { computed, onMounted } from 'vue'
import { useProjectStore } from '../stores/projectStore'
import { useAgentStore } from '../stores/agentStore'
import { useTaskStore } from '../stores/taskStore'

export default {
  name: 'Dashboard',
  setup() {
    const projectStore = useProjectStore()
    const agentStore = useAgentStore()
    const taskStore = useTaskStore()

    onMounted(() => {
      projectStore.fetchProjects()
      agentStore.fetchAgents()
      taskStore.fetchTasks()
    })

    const recentProjects = computed(() => projectStore.projects.slice(0, 5))
    const activeAgents = computed(() => 
      agentStore.agents.filter(a => a.status === 'active').slice(0, 5)
    )
    const recentTasks = computed(() => taskStore.tasks.slice(0, 10))
    const completedTasks = computed(() => 
      taskStore.tasks.filter(t => t.status === 'done').length
    )

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleDateString()
    }

    return {
      projects: projectStore.projects,
      agents: agentStore.agents,
      tasks: taskStore.tasks,
      recentProjects,
      activeAgents,
      recentTasks,
      completedTasks,
      formatDate
    }
  }
}
</script>

<style>
@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.animate-fade-in {
  animation: fadeIn 0.5s ease-out;
}
</style>
