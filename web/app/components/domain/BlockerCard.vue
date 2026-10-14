<script setup lang="ts">
/**
 * One blocker: an agent that has stopped and is waiting on a human decision.
 *
 * Two decisions are baked into this card's layout:
 *
 *   - The left border carries the status colour. A blocker list is scanned at a
 *     glance, not read, and the colour is what does the scanning — it is the
 *     difference between "which of these are still mine" and scrolling.
 *   - A closed blocker shows its text, labelled by what it actually is. An
 *     answer told the agent what to do; a reason told it to stop asking. The
 *     labels are not cosmetic: the difference between them is the entire
 *     reason the backend has two terminal statuses instead of one.
 */
import type { Blocker, BlockerStatus, BlockerType } from '~/types/api'
import { BLOCKER_STATUS_LABELS, BLOCKER_TYPE_LABELS } from '~/types/api'

const props = defineProps<{
  blocker: Blocker
  /** Denormalised by the parent: the panel has no agents list to join against. */
  agentName?: string
  taskTitle?: string
}>()

defineEmits<{
  (e: 'resolve', blocker: Blocker): void
  (e: 'dismiss', blocker: Blocker): void
}>()

const { relativeTime, absoluteIso } = useFormatters()

/**
 * The palette is the app's semantic one rather than per-status hues: red is
 * "an agent is stuck", amber is "being looked at", green is "answered", and
 * neutral recedes so closed blockers stop competing for attention.
 */
const statusBorder: Record<BlockerStatus, string> = {
  open: 'border-l-error',
  in_review: 'border-l-warning',
  resolved: 'border-l-success',
  dismissed: 'border-l-default'
}

const statusColor: Record<BlockerStatus, 'neutral' | 'warning' | 'success' | 'error'> = {
  open: 'error',
  in_review: 'warning',
  resolved: 'success',
  dismissed: 'neutral'
}

const typeColor: Record<BlockerType, 'neutral' | 'info' | 'warning' | 'success'> = {
  question: 'info',
  dependency: 'warning',
  resource: 'neutral',
  decision: 'info',
  approval: 'success'
}

/** Only an open or in-review blocker is a decision still owed to an agent. */
const isOpen = computed(() => props.blocker.status === 'open' || props.blocker.status === 'in_review')

/**
 * How long the agent has been stopped, in the wording that makes the wait the
 * headline rather than a timestamp. Past a day the label goes red on purpose:
 * a blocker from this morning and one from last Thursday need different
 * urgency, and both otherwise render as a neutral grey age.
 */
const waiting = computed(() => {
  const raised = new Date(props.blocker.created_at).getTime()
  if (Number.isNaN(raised)) return { label: '—', overdue: false }
  const days = Math.floor((Date.now() - raised) / 86_400_000)
  const label = days >= 1 ? `Waiting ${days}d` : `Raised ${relativeTime(props.blocker.created_at)}`
  return { label, overdue: days >= 1 && isOpen.value }
})

const closedLabel = computed(() => (props.blocker.status === 'dismissed' ? 'Reason' : 'Answer'))
</script>

<template>
  <UCard class="border-l-4" :class="statusBorder[blocker.status]">
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <h3 class="font-semibold text-sm">{{ blocker.title }}</h3>
          <UBadge :color="typeColor[blocker.blocker_type]" variant="subtle" size="xs">
            {{ BLOCKER_TYPE_LABELS[blocker.blocker_type] }}
          </UBadge>
          <UBadge :color="statusColor[blocker.status]" variant="subtle" size="xs">
            {{ BLOCKER_STATUS_LABELS[blocker.status] }}
          </UBadge>
        </div>

        <p v-if="blocker.description" class="text-sm text-muted mt-2 whitespace-pre-wrap line-clamp-3">
          {{ blocker.description }}
        </p>
      </div>

      <!--
        Actions sit on the right and only while a decision is still owed. A
        closed blocker offers nothing: the controls that close it are exactly
        the ones that would be wrong to press again.
      -->
      <div v-if="isOpen" class="flex items-center gap-1 shrink-0">
        <UButton
          size="xs"
          color="success"
          icon="i-lucide-check"
          :aria-label="`Resolve blocker: ${blocker.title}`"
          @click="$emit('resolve', blocker)"
        >
          Resolve
        </UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-x"
          :aria-label="`Dismiss blocker: ${blocker.title}`"
          @click="$emit('dismiss', blocker)"
        >
          Dismiss
        </UButton>
      </div>
    </div>

    <!--
      The closing text is kept in a quiet block instead of being dropped:
      "who said what, and why" is the history an operator reads when the same
      agent blocks again on the same thing.
    -->
    <div
      v-if="!isOpen && blocker.resolution"
      class="mt-3 rounded-lg border border-default bg-elevated/50 px-3 py-2"
    >
      <span class="text-xs uppercase tracking-wide text-muted">{{ closedLabel }}</span>
      <p class="text-sm mt-1 whitespace-pre-wrap">{{ blocker.resolution }}</p>
    </div>

    <div class="mt-3 pt-3 border-t border-default flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted">
      <span v-if="agentName" class="flex items-center gap-1">
        <UIcon name="i-lucide-bot" class="size-3.5 shrink-0" aria-hidden="true" />
        {{ agentName }}
      </span>
      <span v-if="taskTitle" class="flex items-center gap-1 min-w-0">
        <UIcon name="i-lucide-list-checks" class="size-3.5 shrink-0" aria-hidden="true" />
        <span class="truncate">{{ taskTitle }}</span>
      </span>
      <span
        class="flex items-center gap-1"
        :class="waiting.overdue ? 'text-error font-medium' : ''"
        :title="absoluteIso(blocker.created_at)"
      >
        <UIcon name="i-lucide-clock" class="size-3.5 shrink-0" aria-hidden="true" />
        {{ waiting.label }}
      </span>
    </div>
  </UCard>
</template>