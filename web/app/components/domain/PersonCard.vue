<script setup lang="ts">
import type { Person } from '~/types/admin'

defineProps<{ person: Person }>()

defineEmits<{
  (e: 'show-keys', p: Person): void
}>()

const { relativeTime } = useFormatters()
</script>

<template>
  <UCard>
    <div class="flex items-start justify-between gap-3">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 flex-wrap">
          <UIcon
            :name="person.kind === 'human' ? 'i-lucide-user' : 'i-lucide-bot'"
            class="w-4 h-4"
            :class="person.kind === 'human' ? 'text-primary' : 'text-muted'"
          />
          <h3 class="font-semibold">{{ person.display_name }}</h3>
          <!--
            human and service are visually distinct on purpose: only a human may
            hold approval rights (models.PersonKind.CanApprove), and an operator
            scanning this list needs to see that without opening a record.
          -->
          <UBadge
            :color="person.kind === 'human' ? 'primary' : 'neutral'"
            variant="subtle"
            size="xs"
          >
            {{ person.kind }}
          </UBadge>
        </div>
        <p v-if="person.email" class="text-sm text-muted mt-2">{{ person.email }}</p>
        <p v-else class="text-sm text-muted mt-2 italic">No email on file</p>
      </div>
    </div>

    <div class="mt-4 pt-3 border-t border-default flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
      <div class="flex flex-wrap gap-x-4 gap-y-1">
        <span><span class="font-medium">Added:</span> {{ relativeTime(person.created_at) }}</span>
        <code class="font-mono text-[10px] opacity-70">{{ person.id }}</code>
      </div>
      <UButton
        icon="i-lucide-key-round"
        variant="ghost"
        color="neutral"
        size="xs"
        label="Keys"
        @click="$emit('show-keys', person)"
      />
    </div>
  </UCard>
</template>