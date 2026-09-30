<script setup lang="ts">
/**
 * /global — server-wide playbook console.
 *
 * Lists every `scope='global'` global_context (one per title) and lets the
 * PM publish new ones. Non-PM callers can read but the publish CTA is
 * disabled for them.
 */
import type { GlobalContext, CreateGlobalContextInput } from '~/types/api'

useHead({ title: 'Global context · Agent Shaker' })

const api = useApi()
const { apiBase } = useServerUrl()
const toast = useToastBus()
const settings = useSettingsStore()

const { data: globals,  refresh: refreshGlobal }   = await useFetch<GlobalContext[]>('/global_contexts', { baseURL: apiBase, key: 'gctx:global',  params: { scope: 'global' } })
const { data: projects } = await useFetch<{ id: string; name: string }[]>('/projects', { baseURL: apiBase, key: 'gctx:projects' })

// Re-fetch on focus to pick up writes from other agents / clients.
onMounted(() => { refreshGlobal() })

const showPublish = ref(false)
const viewing = ref<GlobalContext | null>(null)
const viewingOpen = computed({
  get: () => viewing.value !== null,
  set: (v: boolean) => { if (!v) viewing.value = null }
})

const deleting = ref<GlobalContext | null>(null)
const deletingOpen = computed({
  get: () => deleting.value !== null,
  set: (v: boolean) => { if (!v) deleting.value = null }
})

async function handlePublish(input: CreateGlobalContextInput) {
  await api.createGlobalContext(input)
  await refreshGlobal()
  toast.success('Playbook published')
}

async function confirmDelete() {
  if (!deleting.value) return
  await api.deleteGlobalContext(deleting.value.id)
  await refreshGlobal()
  deleting.value = null
  toast.success('Playbook removed')
}
</script>

<template>
  <div>
    <div class="mb-6 flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold flex items-center gap-2">
          <UIcon name="i-lucide-globe" class="text-primary w-6 h-6" /> Global context
        </h1>
        <p class="text-muted mt-1">
          Server-wide playbooks and notes. Every agent can read these via
          <code class="font-mono">read_resource("global://&lt;title&gt;")</code>.
        </p>
      </div>
      <UButton icon="i-lucide-plus" color="primary" size="sm" @click="showPublish = true">
        Publish playbook
      </UButton>
    </div>

    <EmptyState
      v-if="!globals?.length"
      icon="i-lucide-globe"
      title="No global docs yet"
      description="A PM can publish server-wide playbooks so every agent reads the same source of truth."
    />
    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <GlobalContextCard
        v-for="g in globals" :key="g.id" :context="g"
        @view="(g) => viewing = g"
        @delete="(g) => deleting = g"
      />
    </div>

    <!-- View modal reuses the markdown viewer from the Contexts tab. -->
    <ContextViewer v-model="viewingOpen" :context="viewing" />

    <GlobalContextFormModal
      v-model="showPublish"
      :agent-id="settings.recentAgentId ?? ''"
      initial-scope="global"
      @submit="handlePublish"
    />

    <ConfirmDialog
      v-model="deletingOpen"
      :title="`Delete playbook ${deleting?.title}?`"
      confirm-label="Delete"
      @confirm="confirmDelete"
    />

    <div v-if="projects?.length" class="mt-10">
      <h2 class="text-sm font-medium text-muted mb-2">Projects on this server</h2>
      <div class="flex flex-wrap gap-2">
        <NuxtLink
          v-for="p in projects" :key="p.id"
          :to="`/projects/${p.id}`"
          class="text-xs text-primary hover:underline px-2 py-1 rounded bg-elevated"
        >
          {{ p.name }}
        </NuxtLink>
      </div>
    </div>
  </div>
</template>
