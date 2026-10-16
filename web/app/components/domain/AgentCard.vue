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

/**
 * How much of the agent's lease is left, as a percentage.
 *
 * The reaper expires the lease at 0, so the bar is the one thing on this card
 * that answers "is this agent about to have its work taken back". Computed from
 * the deadline the SERVER returned, never from the agent's own clock, so an
 * agent whose clock is wrong does not render as a fleet-wide incident.
 */
const leaseRemaining = (agent: Agent): number | null => {
  if (!agent.lease_expires_at) return null
  const remaining = new Date(agent.lease_expires_at).getTime() - Date.now()
  if (Number.isNaN(remaining)) return null
  // The span is unknown here (it varies with TTL config), so this is a
  // three-state signal rather than a true fraction: plenty, thin, gone.
  if (remaining <= 0) return 0
  if (remaining < 20_000) return 1
  return 2
}

const leaseBarColor = (agent: Agent) => {
  const state = leaseRemaining(agent)
  if (state === 0) return 'bg-error'
  if (state === 1) return 'bg-warning'
  return 'bg-success'
}

/** The reported activity, which is not the same field as the stored status. */
const reportedProgress = (agent: Agent): number | null =>
  typeof agent.last_progress_pct === 'number' ? agent.last_progress_pct : null

/**
 * The ETA the agent last reported, rendered as a short phrase.
 *
 * It is only shown while the agent still holds a lease: an ETA from an agent
 * the reaper has already taken offline is a stale promise, and a card that
 * keeps displaying "done by 20 Oct" three hours after the agent died is worse
 * than no card at all.
 */
const reportedEta = (agent: Agent): string | null => {
  if (!agent.lease_expires_at || !agent.last_heartbeat_eta) return null
  // A lease that has already run out means the reaper may have taken this agent's
  // work back; the ETA it promised belongs to a claim nobody holds any more.
  if (new Date(agent.lease_expires_at).getTime() <= Date.now()) return null
  const eta = new Date(agent.last_heartbeat_eta)
  if (Number.isNaN(eta.getTime())) return null
  const days = Math.ceil((eta.getTime() - Date.now()) / 86_400_000)
  if (days < 0) return 'ETA passed'
  if (days === 0) return 'ETA today'
  if (days === 1) return 'ETA tomorrow'
  return `ETA in ${days}d`
}
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

        <!--
          The lease strip only renders for agents that actually hold one.
          An agent created through the UI has never heartbeat, so it has no
          deadline and no bar — showing a depleted bar next to a healthy agent
          would read as an outage that is not happening.
        -->
        <div v-if="agent.lease_expires_at" class="mt-3">
          <div class="flex items-center justify-between text-xs text-muted">
            <span>{{ reportedProgress(agent) !== null ? `${reportedProgress(agent)}% reported` : 'Lease' }}</span>
            <span v-if="reportedEta(agent)" class="text-warning">{{ reportedEta(agent) }}</span>
            <span v-else>{{ leaseRemaining(agent) === 0 ? 'expired' : `renews ${relativeTime(agent.lease_expires_at)}` }}</span>
          </div>
          <div
            class="mt-1 h-1 w-full rounded-full bg-elevated"
            role="img"
            :aria-label="leaseRemaining(agent) === 0 ? 'Work lease expired' : 'Work lease'"
          >
            <div
              class="h-1 rounded-full transition-[width]"
              :class="leaseBarColor(agent)"
              :style="{ width: leaseRemaining(agent) === 0 ? '100%' : leaseRemaining(agent) === 1 ? '25%' : '100%' }"
            />
          </div>
        </div>
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
