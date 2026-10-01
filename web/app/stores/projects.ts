/**
 * Recent-projects cache. Survives across navigations so the dashboard sidebar
 * can show "Continue working on..." without re-fetching on every mount.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Project } from '~/types/api'

export const useProjectsStore = defineStore('projects', () => {
  /**
   * A plain Vue `ref`, deliberately NOT Nuxt's `useState`.
   *
   * Pinia unwraps Vue refs into its own state, so `store.recent` is a plain
   * array. It does *not* unwrap a `useState` ref: the store's state would then
   * hold a Ref object, and during SSR payload serialisation pinia calls
   * `hasOwnProperty` on each state entry, which throws
   * "obj.hasOwnProperty is not a function" and turns every page into a 500.
   *
   * Pinia already handles SSR hydration of its own state, so `useState` is
   * unnecessary here anyway.
   */
  const recent = ref<Project[]>([])

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
