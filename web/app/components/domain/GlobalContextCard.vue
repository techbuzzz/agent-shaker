<script setup lang="ts">
import type { GlobalContext } from '~/types/api'

defineProps<{
  context: GlobalContext
  agentName?: string
}>()

defineEmits<{
  (e: 'view', c: GlobalContext): void
  (e: 'delete', c: GlobalContext): void
}>()

const { relativeTime, truncate } = useFormatters()
</script>

<template>
  <UCard>
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <UIcon
            :name="context.scope === 'global' ? 'i-lucide-globe' : 'i-lucide-book-text'"
            :class="context.scope === 'global' ? 'text-primary' : 'text-muted'"
            class="w-4 h-4"
          />
          <h3 class="font-semibold">{{ context.title }}</h3>
          <UBadge
            :color="context.scope === 'global' ? 'primary' : 'neutral'"
            variant="subtle"
            size="xs"
          >
            {{ context.scope }}
          </UBadge>
          <UBadge
            v-for="tag in context.tags"
            :key="tag"
            variant="subtle"
            size="xs"
            color="neutral"
          >
            {{ tag }}
          </UBadge>
        </div>
        <p class="text-sm text-muted mt-2 line-clamp-3">{{ truncate(context.content, 280) }}</p>
      </div>
    </div>

    <div class="mt-4 pt-3 border-t border-default flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
      <div class="flex flex-wrap gap-x-4 gap-y-1">
        <span v-if="agentName"><span class="font-medium">By:</span> {{ agentName }}</span>
        <span><span class="font-medium">Updated:</span> {{ relativeTime(context.updated_at) }}</span>
        <code v-if="context.scope === 'global'" class="font-mono text-[10px] opacity-70">global://{{ context.title }}</code>
      </div>
      <div class="flex gap-1 items-center">
        <UButton icon="i-lucide-eye" variant="ghost" color="neutral" size="xs" aria-label="View" @click="$emit('view', context)" />
        <UButton icon="i-lucide-trash" variant="ghost" color="error" size="xs" aria-label="Delete" @click="$emit('delete', context)" />
      </div>
    </div>
  </UCard>
</template>
