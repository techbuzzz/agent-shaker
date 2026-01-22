<template>
  <div class="animate-fade-in">
    <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center">
          <span class="text-white text-lg">📁</span>
        </div>
        <div>
          <h2 class="text-3xl font-bold bg-gradient-to-r from-slate-900 to-slate-600 bg-clip-text text-transparent">
            Projects
          </h2>
          <p class="text-slate-600 text-sm">Manage your AI agent projects</p>
        </div>
      </div>
      <div class="flex gap-3">
        <button @click="showCreateModal = true" class="btn btn-primary group">
          <span class="mr-2">+</span>
          <span class="hidden sm:inline">Create Project</span>
          <span class="sm:hidden">New</span>
        </button>
        <button @click="handleRefresh" :disabled="isRefreshing" class="btn btn-secondary group ml-3">
          <svg v-if="isRefreshing" class="animate-spin w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span v-else class="mr-2">🔄</span>
          <span class="hidden sm:inline">{{ isRefreshing ? 'Refreshing...' : 'Refresh' }}</span>
        </button>
      </div>
    </div>

    <div v-if="loading && !isRefreshing" class="flex items-center justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      <span class="ml-3 text-slate-600">Loading projects...</span>
    </div>

    <div v-else-if="error" class="p-4 bg-red-50 border border-red-200 text-red-700 rounded-xl mb-6">
      <div class="flex items-center justify-between">
        <div class="flex items-center">
          <div class="w-5 h-5 bg-red-500 rounded-full flex items-center justify-center mr-3">
            <span class="text-white text-xs">⚠️</span>
          </div>
          <div>
            <p class="font-medium">Failed to load projects</p>
            <p class="text-sm text-red-600">{{ error }}</p>
          </div>
        </div>
        <button @click="handleRefresh" class="btn btn-secondary text-sm">
          Try Again
        </button>
      </div>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div v-for="project in projects" :key="project.id" class="card group cursor-pointer transform hover:scale-[1.02] transition-all duration-200">
        <router-link :to="`/projects/${project.id}`" class="block text-inherit no-underline">
          <div class="flex justify-between items-start mb-4">
            <h3 class="text-xl font-semibold text-slate-900 group-hover:text-blue-600 transition-colors line-clamp-1">
              {{ project.name }}
            </h3>
            <span :class="['inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium', project.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-800']">
              <div :class="['w-1.5 h-1.5 rounded-full mr-1.5', project.status === 'active' ? 'bg-green-500' : 'bg-slate-400']"></div>
              {{ project.status }}
            </span>
          </div>
          <p class="text-slate-600 mb-4 leading-relaxed line-clamp-3">{{ project.description }}</p>
          <div class="flex items-center justify-between text-sm text-slate-500">
            <span class="flex items-center gap-1">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
              {{ formatDate(project.created_at) }}
            </span>
            <svg class="w-5 h-5 text-slate-400 group-hover:text-blue-500 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
            </svg>
          </div>
        </router-link>
      </div>
    </div>

    <div v-if="!loading && projects.length === 0" class="text-center py-16">
      <div class="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mx-auto mb-4">
        <span class="text-2xl">📁</span>
      </div>
      <h3 class="text-xl font-semibold text-slate-900 mb-2">No projects yet</h3>
      <p class="text-slate-600 mb-6 max-w-md mx-auto">Create your first project to start coordinating AI agents and managing tasks.</p>
      <button @click="showCreateModal = true" class="btn btn-primary">
        <span class="mr-2">+</span>
        Create Your First Project
      </button>
    </div>

    <!-- Create Project Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 animate-fade-in" @click.self="showCreateModal = false">
      <div class="bg-white p-8 rounded-2xl max-w-md w-full mx-4 shadow-2xl transform animate-scale-in">
        <div class="flex items-center gap-3 mb-6">
          <div class="w-10 h-10 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center">
            <span class="text-white text-lg">+</span>
          </div>
          <h3 class="text-xl font-semibold text-slate-900">Create New Project</h3>
        </div>
        <form @submit.prevent="handleCreateProject" class="space-y-6">
          <div>
            <label class="label">Project Name</label>
            <input
              v-model="newProject.name"
              type="text"
              placeholder="Enter project name"
              class="input"
              required
            />
          </div>
          <div>
            <label class="label">Description</label>
            <textarea
              v-model="newProject.description"
              placeholder="Enter project description"
              rows="4"
              class="input resize-none"
            ></textarea>
          </div>
          <div class="flex justify-end gap-3 pt-4">
            <button type="button" @click="showCreateModal = false" class="btn btn-secondary">
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">
              Create Project
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { useProjectStore } from '../stores/projectStore'

export default {
  name: 'Projects',
  setup() {
    const projectStore = useProjectStore()
    const showCreateModal = ref(false)
    const newProject = ref({ name: '', description: '' })
    const isRefreshing = ref(false)

    onMounted(() => {
      fetchProjectsData()
    })

    const fetchProjectsData = async () => {
      try {
        await projectStore.fetchProjects()
      } catch (error) {
        console.error('Failed to load projects:', error)
        // Error is already handled in the store
        // Auto-retry once after 2 seconds if it's the first load and no data exists
        if (projectStore.projects.length === 0) {
          setTimeout(() => {
            if (projectStore.error && projectStore.projects.length === 0) {
              console.log('Auto-retrying to load projects...')
              fetchProjectsData()
            }
          }, 2000)
        }
      }
    }

    const handleRefresh = async () => {
      isRefreshing.value = true
      try {
        await projectStore.fetchProjects()
      } catch (error) {
        console.error('Failed to refresh projects:', error)
      } finally {
        isRefreshing.value = false
      }
    }

    const handleCreateProject = async () => {
      try {
        await projectStore.createProject(newProject.value)
        showCreateModal.value = false
        newProject.value = { name: '', description: '' }
      } catch (error) {
        console.error('Failed to create project:', error)
      }
    }

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleDateString()
    }

    return {
      projects: computed(() => projectStore.projects),
      loading: computed(() => projectStore.loading),
      error: computed(() => projectStore.error),
      showCreateModal,
      newProject,
      handleCreateProject,
      handleRefresh,
      isRefreshing,
      formatDate
    }
  }
}
</script>

<style>
.label {
  display: block;
  font-size: 0.875rem;
  font-weight: 500;
  color: rgb(55 65 81);
  margin-bottom: 0.5rem;
}

.input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid rgb(209 213 219);
  border-radius: 0.375rem;
  outline: none;
}

.input:focus {
  border-color: rgb(37 99 235);
  box-shadow: 0 0 0 2px rgb(37 99 235 / 0.2);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-weight: 500;
  transition: all 0.2s ease;
  border: none;
  cursor: pointer;
}

.btn-primary {
  background-color: rgb(37 99 235);
  color: white;
}

.btn-primary:hover {
  background-color: rgb(29 78 216);
}

.btn-secondary {
  background-color: rgb(229 231 235);
  color: rgb(31 41 55);
}

.btn-secondary:hover {
  background-color: rgb(209 213 219);
}

.animate-fade-in {
  opacity: 0;
  animation: fadeIn 0.4s forwards;
}

@keyframes fadeIn {
  to {
    opacity: 1;
  }
}

.animate-scale-in {
  transform: scale(0.95);
  animation: scaleIn 0.4s forwards;
}

@keyframes scaleIn {
  to {
    transform: scale(1);
  }
}
</style>
