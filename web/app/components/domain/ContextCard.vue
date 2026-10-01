<script setup lang="ts">
import type { Context } from '~/types/api'

const props = defineProps<{
  context: Context
  agentName?: string
}>()

defineEmits<{ (e: 'view', c: Context): void; (e: 'delete', c: Context): void }>()

const { truncate, relativeTime } = useFormatters()
</script>

<template>
  <UCard :ui="{ body: 'p-5' }" class="hover:ring-1 hover:ring-primary/30 transition-shadow cursor-pointer" @click="$emit('view', context)">
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <h3 class="font-semibold truncate">{{ context.title }}</h3>
        <p class="text-sm text-muted mt-1 line-clamp-2 font-mono">{{ truncate(context.content.replace(/[#*`]/g, ''), 180) }}</p>
      </div>
      <UButton
        icon="i-lucide-trash"
        variant="ghost"
        color="error"
        size="xs"
        aria-label="Delete"
        @click.stop="$emit('delete', context)"
      />
    </div>
    <div class="mt-3 pt-3 border-t border-default flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
      <div class="flex flex-wrap gap-x-3 gap-y-1">
        <span v-if="agentName"><span class="font-medium">By:</span> {{ agentName }}</span>
        <span>{{ relativeTime(context.created_at) }}</span>
      </div>
      <div v-if="context.tags?.length" class="flex flex-wrap gap-1">
        <UBadge v-for="t in context.tags" :key="t" variant="subtle" size="xs" color="neutral">{{ t }}</UBadge>
      </div>
    </div>
  </UCard>
</template>
