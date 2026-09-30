<script setup lang="ts">
import type { Milestone, MilestoneStatus } from '~/types/api'

const props = defineProps<{
  milestone: Milestone
  taskCount?: number
  openTaskCount?: number
}>()

defineEmits<{
  (e: 'change-status', payload: { milestone: Milestone; status: MilestoneStatus }): void
  (e: 'delete', m: Milestone): void
}>()

const { relativeTime } = useFormatters()

const statusColor: Record<MilestoneStatus, 'neutral' | 'info' | 'warning' | 'success' | 'error'> = {
  planned: 'neutral',
  active: 'info',
  done: 'success',
  dropped: 'error'
}

const progressPct = computed(() => {
  const total = props.taskCount ?? 0
  if (!total) return 0
  const done = total - (props.openTaskCount ?? 0)
  return Math.round((done / total) * 100)
})
</script>

<template>
  <UCard>
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <UIcon name="i-lucide-flag" class="text-primary w-4 h-4" />
          <h3 class="font-semibold">{{ milestone.title }}</h3>
          <UBadge :color="statusColor[milestone.status]" variant="subtle" size="xs">
            {{ milestone.status }}
          </UBadge>
        </div>
        <p v-if="milestone.description" class="text-sm text-muted mt-2 line-clamp-2">{{ milestone.description }}</p>
      </div>
    </div>

    <div class="mt-4">
      <div class="flex items-center justify-between text-xs text-muted mb-1">
        <span>Progress</span>
        <span>{{ openTaskCount ?? 0 }} open / {{ taskCount ?? 0 }} total</span>
      </div>
      <UProgress :value="progressPct" :max="100" />
    </div>

    <div class="mt-4 pt-3 border-t border-default flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
      <div class="flex flex-wrap gap-x-4 gap-y-1">
        <span v-if="milestone.target_date"><span class="font-medium">Target:</span> {{ milestone.target_date }}</span>
        <span><span class="font-medium">Updated:</span> {{ relativeTime(milestone.updated_at) }}</span>
      </div>
      <div class="flex gap-1 items-center">
        <USelectMenu
          :model-value="milestone.status"
          :items="[
            { label: 'Planned', value: 'planned' },
            { label: 'Active',  value: 'active' },
            { label: 'Done',    value: 'done' },
            { label: 'Dropped', value: 'dropped' }
          ]"
          value-key="value"
          size="xs"
          @update:model-value="(v) => $emit('change-status', { milestone, status: v as MilestoneStatus })"
        />
        <UButton icon="i-lucide-trash" variant="ghost" color="error" size="xs" aria-label="Delete milestone" @click="$emit('delete', milestone)" />
      </div>
    </div>
  </UCard>
</template>
