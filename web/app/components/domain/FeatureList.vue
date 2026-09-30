<script setup lang="ts">
/**
 * FeatureList — derived view over the project tasks list.
 *
 * Features are not a first-class concept; they are encoded as `feature:*`
 * tags on tasks (e.g. a task tagged `feature:auth` belongs to the
 * "auth" feature). This component groups tasks by that prefix and shows
 * a compact per-feature rollup. Tasks without a `feature:` tag appear
 * under the "Untagged" group at the bottom.
 *
 * This is intentionally a pure read view — there is no create-feature
 * form. Operators add a feature by tagging a task with `feature:<name>`,
 * either in TaskFormModal or via the MCP `create_task` tool's `tags`
 * argument.
 */
import type { Task } from '~/types/api'

const props = defineProps<{
  tasks: Task[]
}>()

const FEATURE_PREFIX = 'feature:'

interface FeatureGroup {
  name: string
  total: number
  done: number
  tasks: Task[]
}

const groups = computed<FeatureGroup[]>(() => {
  const buckets = new Map<string, Task[]>()
  const untagged: Task[] = []

  for (const t of props.tasks ?? []) {
    const featTag = (t.tags ?? []).find((tag) => tag.startsWith(FEATURE_PREFIX))
    if (!featTag) {
      untagged.push(t)
      continue
    }
    const name = featTag.slice(FEATURE_PREFIX.length) || '(unnamed)'
    if (!buckets.has(name)) buckets.set(name, [])
    buckets.get(name)!.push(t)
  }

  const result: FeatureGroup[] = []
  for (const [name, tasks] of buckets.entries()) {
    const done = tasks.filter((t) => t.status === 'done').length
    result.push({ name, total: tasks.length, done, tasks })
  }
  result.sort((a, b) => b.total - a.total)

  if (untagged.length) {
    result.push({
      name: 'Untagged',
      total: untagged.length,
      done: untagged.filter((t) => t.status === 'done').length,
      tasks: untagged
    })
  }
  return result
})
</script>

<template>
  <div class="space-y-3">
    <EmptyState
      v-if="!groups.length"
      icon="i-lucide-layout-grid"
      title="No features yet"
      description="Tag a task with feature:&lt;name&gt; in the task editor to roll it up here."
    />
    <UCard
      v-for="g in groups"
      :key="g.name"
    >
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-layout-grid" class="text-primary w-4 h-4" />
          <h4 class="font-semibold">{{ g.name }}</h4>
        </div>
        <div class="text-xs text-muted">
          {{ g.done }} / {{ g.total }} done
        </div>
      </div>
      <UProgress :value="g.total ? Math.round((g.done / g.total) * 100) : 0" :max="100" class="mt-2" />
      <div class="mt-3 space-y-1">
        <div
          v-for="t in g.tasks"
          :key="t.id"
          class="flex items-center justify-between text-xs"
        >
          <span class="truncate flex-1">{{ t.title }}</span>
          <UBadge
            :color="t.status === 'done' ? 'success' : t.status === 'in_progress' ? 'info' : 'neutral'"
            variant="subtle"
            size="xs"
          >
            {{ t.status.replace('_', ' ') }}
          </UBadge>
        </div>
      </div>
    </UCard>
  </div>
</template>
