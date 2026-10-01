<script setup lang="ts">
import type { Standup } from '~/types/api'

defineProps<{ standup: Standup; agentName?: string }>()
defineEmits<{ (e: 'delete', s: Standup): void }>()

const { formatDate } = useFormatters()
</script>

<template>
  <UCard :ui="{ body: 'p-5' }">
    <div class="flex items-start justify-between gap-3 mb-3">
      <div>
        <h3 class="font-semibold flex items-center gap-2">
          {{ agentName || standup.agent_id }}
          <UBadge variant="subtle" size="xs">{{ formatDate(standup.date) }}</UBadge>
        </h3>
      </div>
      <UButton icon="i-lucide-trash" variant="ghost" color="error" size="xs" aria-label="Delete" @click="$emit('delete', standup)" />
    </div>
    <dl class="space-y-3 text-sm">
      <div>
        <dt class="text-xs uppercase tracking-wide text-muted mb-1">Yesterday</dt>
        <dd class="whitespace-pre-wrap">{{ standup.yesterday || '—' }}</dd>
      </div>
      <div>
        <dt class="text-xs uppercase tracking-wide text-muted mb-1">Today</dt>
        <dd class="whitespace-pre-wrap">{{ standup.today || '—' }}</dd>
      </div>
      <div v-if="standup.blockers">
        <dt class="text-xs uppercase tracking-wide text-muted mb-1">Blockers</dt>
        <dd class="whitespace-pre-wrap text-error-600 dark:text-error-400">{{ standup.blockers }}</dd>
      </div>
    </dl>
  </UCard>
</template>
