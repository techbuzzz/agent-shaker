<script setup lang="ts">
import type { APIKey } from '~/types/admin'
import { KEY_SCOPE_LABELS } from '~/types/admin'
import type { KeyScope } from '~/types/admin'

const props = defineProps<{
  apiKey: APIKey
  projectName?: string
}>()

defineEmits<{
  (e: 'revoke', k: APIKey): void
}>()

const { relativeTime } = useFormatters()

/**
 * The scope array is nullable: models.NormalizeScopes maps an empty request to
 * nil, and models.Allows reads nil as "unrestricted". Rendering that as an
 * empty row of badges would look like a key with no permissions at all.
 */
const scopes = computed<KeyScope[]>(() => props.apiKey.scopes ?? [])
const revoked = computed(() => Boolean(props.apiKey.revoked_at))
</script>

<template>
  <UCard :class="revoked ? 'opacity-70' : ''">
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <UIcon
            :name="revoked ? 'i-lucide-key-round' : 'i-lucide-shield-check'"
            class="w-4 h-4"
            :class="revoked ? 'text-muted' : 'text-primary'"
          />
          <code class="font-mono text-sm font-semibold">{{ apiKey.key_prefix }}…</code>
          <UBadge
            :color="revoked ? 'error' : 'success'"
            variant="subtle"
            size="xs"
          >
            {{ revoked ? 'revoked' : 'active' }}
          </UBadge>
          <UBadge
            v-for="scope in scopes"
            :key="scope"
            variant="outline"
            size="xs"
            color="neutral"
          >
            {{ KEY_SCOPE_LABELS[scope] ?? scope }}
          </UBadge>
          <UBadge v-if="scopes.length === 0" variant="outline" size="xs" color="warning">
            unrestricted
          </UBadge>
        </div>

        <p class="text-sm text-muted mt-2">
          <UIcon
            :name="apiKey.principal_type === 'human' ? 'i-lucide-user' : 'i-lucide-bot'"
            class="w-3.5 h-3.5 inline"
          />
          {{ apiKey.principal_name || apiKey.principal_id }}
          <span class="opacity-70">({{ apiKey.principal_type }})</span>
          <template v-if="projectName">
            · scoped to <span class="font-medium">{{ projectName }}</span>
          </template>
        </p>
      </div>
    </div>

    <div class="mt-4 pt-3 border-t border-default flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
      <div class="flex flex-wrap gap-x-4 gap-y-1">
        <span><span class="font-medium">Created:</span> {{ relativeTime(apiKey.created_at) }}</span>
        <span><span class="font-medium">Last used:</span> {{ relativeTime(apiKey.last_used_at) }}</span>
        <span v-if="revoked"><span class="font-medium">Revoked:</span> {{ relativeTime(apiKey.revoked_at) }}</span>
      </div>
      <UButton
        v-if="!revoked"
        icon="i-lucide-ban"
        variant="ghost"
        color="error"
        size="xs"
        label="Revoke"
        @click="$emit('revoke', apiKey)"
      />
    </div>
  </UCard>
</template>