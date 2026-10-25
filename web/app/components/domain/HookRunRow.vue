<script setup lang="ts">
import type { HookRun } from '~/types/hooks'
import { HOOK_RUN_STATUS_LABELS } from '~/types/hooks'
import { HOOK_RUN_STATUS_COLORS, formatHookDuration, hookEventLabel } from '~/utils/hooks'

/**
 * One run of one hook.
 *
 * The captured output is collapsed by default and expands in place. A run log
 * that shows every command's output by default is unreadable after ten runs,
 * and the output is the part nobody reads until they need it.
 */
const props = defineProps<{
  run: HookRun
  /** The output is on screen. */
  expanded?: boolean
}>()

const emit = defineEmits<{ (e: 'toggle', id: string): void }>()

const { relativeTime } = useFormatters()
</script>

<template>
  <UCard :ui="{ body: 'p-4' }">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <UBadge :color="HOOK_RUN_STATUS_COLORS[run.status]" variant="subtle" size="xs">
            {{ HOOK_RUN_STATUS_LABELS[run.status] }}
          </UBadge>
          <span class="text-sm font-medium">{{ hookEventLabel(run.event) }}</span>
          <code class="font-mono text-xs text-muted">{{ run.event }}</code>
        </div>
        <p class="mt-1 text-xs text-muted">
          exit {{ run.exit_code ?? '—' }}
          · {{ formatHookDuration(run.duration_ms) }}
          · {{ relativeTime(run.started_at) }}
          <template v-if="run.entity_type && run.entity_id">
            · {{ run.entity_type }} {{ run.entity_id }}
          </template>
        </p>
      </div>
      <UButton
        size="xs"
        color="neutral"
        variant="ghost"
        :icon="expanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
        @click="emit('toggle', run.id)"
      >
        Output
      </UButton>
    </div>

    <div v-if="expanded" class="mt-3 space-y-3">
      <div v-if="run.stdout">
        <p class="text-xs uppercase tracking-wide text-muted">stdout</p>
        <pre class="mt-1 text-xs bg-default/50 rounded p-2 overflow-auto max-h-56 font-mono whitespace-pre-wrap">{{ run.stdout }}</pre>
      </div>
      <div v-if="run.stderr">
        <p class="text-xs uppercase tracking-wide text-muted">stderr</p>
        <pre class="mt-1 text-xs bg-default/50 rounded p-2 overflow-auto max-h-56 font-mono whitespace-pre-wrap text-error">{{ run.stderr }}</pre>
      </div>
      <p v-if="!run.stdout && !run.stderr" class="text-xs text-muted">
        The command produced no output.
      </p>
    </div>
  </UCard>
</template>
