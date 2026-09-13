<script setup lang="ts">
import type { Task, TaskStatus, TaskPriority } from '~/types/api'

const props = defineProps<{
  task: Task
  assigneeName?: string
  projectName?: string
}>()

defineEmits<{
  (e: 'edit', t: Task): void
  (e: 'delete', t: Task): void
  (e: 'change-status', payload: { task: Task; status: TaskStatus }): void
}>()

const { relativeTime, truncate } = useFormatters()

const statusColor: Record<TaskStatus, 'neutral' | 'info' | 'warning' | 'success' | 'error'> = {
  pending: 'neutral',
  in_progress: 'info',
  blocked: 'error',
  done: 'success',
  cancelled: 'neutral'
}

const priorityColor: Record<TaskPriority, 'info' | 'warning' | 'error'> = {
  low: 'info',
  medium: 'warning',
  high: 'error'
}
</script>

<template>
  <UCard :ui="{ body: { padding: 'p-5' } }">
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <h3 class="font-semibold">{{ task.title }}</h3>
          <UBadge :color="priorityColor[task.priority]" variant="subtle" size="xs">
            {{ task.priority }}
          </UBadge>
          <UBadge :color="statusColor[task.status]" variant="subtle" size="xs">
            {{ task.status.replace('_', ' ') }}
          </UBadge>
        </div>
        <p class="text-sm text-muted mt-2 line-clamp-2">{{ truncate(task.description, 200) }}</p>
      </div>
    </div>

    <div class="mt-4 pt-3 border-t border-default flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
      <div class="flex flex-wrap gap-x-4 gap-y-1">
        <span v-if="projectName"><span class="font-medium">Project:</span> {{ projectName }}</span>
        <span v-if="assigneeName"><span class="font-medium">Assignee:</span> {{ assigneeName }}</span>
        <span><span class="font-medium">Updated:</span> {{ relativeTime(task.updated_at) }}</span>
      </div>
      <div class="flex gap-1 items-center">
        <USelectMenu
          :model-value="task.status"
          :items="[
            { label: 'Pending',     value: 'pending' },
            { label: 'In progress', value: 'in_progress' },
            { label: 'Blocked',     value: 'blocked' },
            { label: 'Done',        value: 'done' },
            { label: 'Cancelled',   value: 'cancelled' }
          ]"
          value-key="value"
          size="xs"
          @update:model-value="(v) => $emit('change-status', { task, status: v as TaskStatus })"
        />
        <UButton icon="i-lucide-pencil" variant="ghost" color="neutral" size="xs" aria-label="Edit task" @click="$emit('edit', task)" />
        <UButton icon="i-lucide-trash" variant="ghost" color="error" size="xs" aria-label="Delete task" @click="$emit('delete', task)" />
      </div>
    </div>
  </UCard>
</template>
