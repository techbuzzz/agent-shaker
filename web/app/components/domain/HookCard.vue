<script setup lang="ts">
import type { Hook } from '~/types/hooks'
import { HOOK_EVENT_LABELS } from '~/types/hooks'

/**
 * One configured hook.
 *
 * Presentational on purpose: it takes a hook and reports what the operator
 * did with it, and the page decides what that means. The alternative — a card
 * that calls the API itself — is how a list ends up with four different
 * error-handling stories.
 */
const props = defineProps<{
  hook: Hook
  /** The hook's test run is in flight, so only its button waits. */
  testing?: boolean
}>()

const emit = defineEmits<{
  (e: 'edit', hook: Hook): void
  (e: 'remove', hook: Hook): void
  (e: 'toggle', hook: Hook): void
  (e: 'test', hook: Hook): void
}>()

/**
 * The event's label.
 *
 * The vocabulary is the server's, so a hook carrying an event this build does
 * not know about renders with its raw name rather than as a blank badge.
 */
const eventLabel = computed(
  () => (HOOK_EVENT_LABELS as Partial<Record<string, string>>)[props.hook.event] ?? props.hook.event
)
</script>

<template>
  <UCard :ui="{ body: 'p-5' }">
    <div class="flex flex-col lg:flex-row lg:items-start lg:justify-between gap-4">
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <UBadge variant="subtle" size="xs" color="info">
            {{ eventLabel }}
          </UBadge>
          <code class="font-mono text-xs text-muted">{{ hook.event }}</code>
          <UBadge :color="hook.enabled ? 'success' : 'neutral'" variant="subtle" size="xs">
            {{ hook.enabled ? 'Enabled' : 'Disabled' }}
          </UBadge>
        </div>

        <div class="mt-3">
          <p class="text-xs uppercase tracking-wide text-muted">Command</p>
          <p class="mt-1 font-mono text-sm break-all">{{ hook.command.join(' ') }}</p>
          <p class="text-xs text-muted mt-1">
            {{ hook.command.length }} argument{{ hook.command.length === 1 ? '' : 's' }}, executed as an
            argument list — no shell, so there is nothing to quote.
          </p>
        </div>

        <dl class="mt-3 grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-1 text-xs">
          <div class="flex gap-2">
            <dt class="font-medium text-default shrink-0">Matcher</dt>
            <dd class="text-muted min-w-0">
              <template v-if="hook.matcher">
                <span class="font-mono">{{ hook.matcher.field }}</span> matches
                <span class="font-mono">{{ hook.matcher.pattern }}</span>
              </template>
              <template v-else>Every event of this type</template>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="font-medium text-default shrink-0">stdin</dt>
            <dd class="text-muted">
              {{ hook.stdin.mode === 'prompt' ? 'Rendered prompt template' : 'Event JSON' }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="font-medium text-default shrink-0">post</dt>
            <dd class="text-muted min-w-0">
              <template v-if="hook.post">Context: {{ hook.post.title }}</template>
              <template v-else>Nothing</template>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="font-medium text-default shrink-0">Timeout</dt>
            <dd class="text-muted">{{ hook.timeout_seconds }}s</dd>
          </div>
          <!--
            The environment is shown, not edited. What is stored here is
            a reference such as {secret:NAME}; the server resolves it at
            run time and redacts the resolved value from the captured
            output, so this line can be shown without leaking anything.
          -->
          <div v-if="hook.env && Object.keys(hook.env).length" class="flex gap-2">
            <dt class="font-medium text-default shrink-0">env</dt>
            <dd class="text-muted min-w-0 font-mono">
              <span v-for="(value, name) in hook.env" :key="name" class="mr-3">
                {{ name }}={{ value }}
              </span>
            </dd>
          </div>
        </dl>
      </div>

      <div class="flex shrink-0 flex-wrap items-center gap-1 lg:justify-end">
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          icon="i-lucide-flask-conical"
          :loading="testing"
          @click="emit('test', hook)"
        >
          Test
        </UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-pencil"
          @click="emit('edit', hook)"
        >
          Edit
        </UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          :icon="hook.enabled ? 'i-lucide-pause' : 'i-lucide-play'"
          @click="emit('toggle', hook)"
        >
          {{ hook.enabled ? 'Disable' : 'Enable' }}
        </UButton>
        <UButton
          size="xs"
          color="error"
          variant="ghost"
          icon="i-lucide-trash"
          :aria-label="`Delete the ${eventLabel} hook`"
          @click="emit('remove', hook)"
        />
      </div>
    </div>
  </UCard>
</template>
