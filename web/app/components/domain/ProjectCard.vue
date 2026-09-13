<script setup lang="ts">
import type { Project } from '~/types/api'

defineProps<{ project: Project }>()
defineEmits<{ (e: 'edit', project: Project): void; (e: 'delete', project: Project): void }>()

const { formatDate, truncate } = useFormatters()
</script>

<template>
  <UCard class="hover:ring-1 hover:ring-primary/30 transition-shadow group" :ui="{ body: { padding: 'p-5' } }">
    <div class="flex items-start justify-between gap-3 mb-2">
      <div class="flex-1 min-w-0">
        <NuxtLink :to="`/projects/${project.id}`" class="block">
          <h3 class="text-base font-semibold truncate group-hover:text-primary transition-colors">
            {{ project.name }}
          </h3>
        </NuxtLink>
        <p class="text-sm text-muted mt-1 line-clamp-2">{{ truncate(project.description, 120) }}</p>
      </div>
      <UBadge :color="project.status === 'active' ? 'success' : 'neutral'" variant="subtle" size="xs">
        {{ project.status }}
      </UBadge>
    </div>

    <div class="mt-4 pt-3 border-t border-default flex items-center justify-between text-xs text-muted">
      <span>Created {{ formatDate(project.created_at) }}</span>
      <div class="opacity-0 group-hover:opacity-100 transition-opacity flex gap-1">
        <UButton
          icon="i-lucide-pencil"
          variant="ghost"
          color="neutral"
          size="xs"
          aria-label="Edit project"
          @click="$emit('edit', project)"
        />
        <UButton
          icon="i-lucide-trash"
          variant="ghost"
          color="error"
          size="xs"
          aria-label="Delete project"
          @click="$emit('delete', project)"
        />
      </div>
    </div>
  </UCard>
</template>
