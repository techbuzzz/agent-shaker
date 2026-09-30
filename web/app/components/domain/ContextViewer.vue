<script setup lang="ts">
import type { Context, GlobalContext } from '~/types/api'

// The viewer is structural — both `Context` (project-scoped, agent_id)
// and `GlobalContext` (server-wide, optional project_id) share the same
// shape for the fields rendered below. We build a structural alias so
// callers don't have to cast.
type ViewableContext = {
  id: string
  title: string
  content: string
  tags: string[]
  agent_id: string
  created_at: string
  updated_at?: string
  project_id?: string
}

const _ctx: ViewableContext = {} as Context
const _gctx: ViewableContext = {} as GlobalContext
void _ctx
void _gctx

const props = defineProps<{
  modelValue: boolean
  context: ViewableContext | null
  agentName?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const { formatDateTime } = useFormatters()
</script>

<template>
  <UModal v-model:open="open" :title="context?.title || 'Documentation'" icon="i-lucide-book-text" :ui="{ content: 'sm:max-w-3xl' }">
    <template #body>
      <div v-if="context" class="space-y-4">
        <div class="flex flex-wrap items-center gap-3 text-xs text-muted">
          <span v-if="agentName"><UIcon name="i-lucide-user" class="w-3.5 h-3.5 inline -mt-0.5" /> {{ agentName }}</span>
          <span><UIcon name="i-lucide-clock" class="w-3.5 h-3.5 inline -mt-0.5" /> {{ formatDateTime(context.created_at) }}</span>
          <div v-if="context.tags?.length" class="flex flex-wrap gap-1">
            <UBadge v-for="t in context.tags" :key="t" variant="subtle" size="xs" color="neutral">{{ t }}</UBadge>
          </div>
        </div>
        <USeparator />
        <MarkdownView :source="context.content" />
      </div>
    </template>
  </UModal>
</template>
