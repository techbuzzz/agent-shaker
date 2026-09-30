<script setup lang="ts">
import type { ProjectRepo } from '~/types/api'

defineProps<{
  repo: ProjectRepo
  agentName?: string
}>()

defineEmits<{
  (e: 'delete', r: ProjectRepo): void
}>()
</script>

<template>
  <UCard>
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-git-branch" class="text-primary w-4 h-4 shrink-0" />
          <a
            :href="repo.url"
            target="_blank"
            rel="noopener"
            class="font-mono text-sm font-medium hover:text-primary truncate"
          >
            {{ repo.url }}
          </a>
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted">
          <UBadge variant="subtle" size="xs" color="neutral">{{ repo.branch }}</UBadge>
          <UBadge variant="subtle" size="xs" color="info">{{ repo.role }}</UBadge>
          <span v-if="agentName"><span class="font-medium">Owner:</span> {{ agentName }}</span>
        </div>
      </div>
      <UButton
        icon="i-lucide-trash"
        variant="ghost"
        color="error"
        size="xs"
        aria-label="Remove repo"
        @click="$emit('delete', repo)"
      />
    </div>
  </UCard>
</template>
