<script setup lang="ts">
import type { Agent, AgentStatus } from '~/types/api'

defineProps<{ agent: Agent }>()
defineEmits<{ (e: 'edit', a: Agent): void; (e: 'delete', a: Agent): void }>()

const { relativeTime } = useFormatters()

// Explicit map rather than a ternary chain: adding a status to AGENT_STATUSES
// must produce a deliberate colour, not a silent fall-through to 'neutral'.
const STATUS_COLOR: Record<AgentStatus, 'success' | 'warning' | 'error' | 'neutral'> = {
  active: 'success',
  idle: 'warning',
  blocked: 'error',
  offline: 'neutral'
}

const statusColor = (s: AgentStatus) => STATUS_COLOR[s] ?? 'neutral'
</script>

<template>
  <UCard :ui="{ body: 'p-5' }">
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
