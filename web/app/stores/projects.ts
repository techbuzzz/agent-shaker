/**
 * Recent-projects cache. Survives across navigations so the dashboard sidebar
 * can show "Continue working on..." without re-fetching on every mount.
 * Backed by Nuxt's `useState` (SSR-safe) and refreshed by useAsyncData on the
 * projects list page.
 */
import { defineStore } from 'pinia'
import type { Project } from '~/types/api'

export const useProjectsStore = defineStore('projects', () => {
  const recent = useState<Project[]>('projects:recent', () => [])

  function setRecent(list: Project[]) {
    // Keep most recently updated first
    recent.value = [...list].sort((a, b) => +new Date(b.updated_at) - +new Date(a.updated_at))
  }

  function upsert(p: Project) {
    const idx = recent.value.findIndex((x) => x.id === p.id)
    if (idx >= 0) recent.value.splice(idx, 1, p)
    else recent.value.unshift(p)
  }

  function remove(id: string) {
    recent.value = recent.value.filter((p) => p.id !== id)
  }

  return { recent, setRecent, upsert, remove }
})
