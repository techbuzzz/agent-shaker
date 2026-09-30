<script setup lang="ts">
import type { Agent } from '~/types/api'

defineProps<{ agent: Agent }>()
defineEmits<{ (e: 'edit', a: Agent): void; (e: 'delete', a: Agent): void }>()

const { relativeTime } = useFormatters()

const statusColor = computed(() => {
  // Not used as computed inside template directly; expose via getter.
  return (s: Agent['status']) =>
    s === 'active' ? 'success' : s === 'idle' ? 'warning' : 'neutral'
})
</script>

<template>
  <UCard :ui="{ body: { padding: 'p-5' } }">
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <h3 class="font-semibold truncate">{{ agent.name }}</h3>
          <UBadge
            :color="agent.role === 'pm' ? 'primary' : 'neutral'"
            variant="subtle"
            size="xs"
          >
            {{ agent.role }}
          </UBadge>
        </div>
        <p class="text-xs text-muted mt-1">{{ agent.team || '—' }}</p>
      </div>
      <UBadge :color="statusColor(agent.status)" variant="subtle" size="xs">
        {{ agent.status }}
      </UBadge>
    </div>

    <div class="mt-3 pt-3 border-t border-default flex items-center justify-between text-xs text-muted">
      <span>Last seen {{ relativeTime(agent.last_seen) }}</span>
      <div class="flex gap-1">
        <UButton
          icon="i-lucide-pencil"
          variant="ghost"
          color="neutral"
          size="xs"
          aria-label="Edit agent"
          @click="$emit('edit', agent)"
        />
        <UButton
          icon="i-lucide-trash"
          variant="ghost"
          color="error"
          size="xs"
          aria-label="Delete agent"
          @click="$emit('delete', agent)"
        />
      </div>
    </div>
  </UCard>
</template>
