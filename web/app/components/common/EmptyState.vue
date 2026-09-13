<script setup lang="ts">
withDefaults(defineProps<{
  icon?: string
  title: string
  description?: string
}>(), {
  icon: 'i-lucide-inbox',
  description: ''
})
</script>

<template>
  <div class="text-center py-12 px-6 border border-dashed border-default rounded-xl bg-elevated/40">
    <div class="mx-auto w-12 h-12 rounded-full bg-default flex items-center justify-center mb-4">
      <UIcon :name="icon" class="w-6 h-6 text-muted" />
    </div>
    <h3 class="text-base font-semibold mb-1">{{ title }}</h3>
    <p v-if="description" class="text-sm text-muted max-w-md mx-auto">{{ description }}</p>
    <div class="mt-4">
      <slot />
    </div>
  </div>
</template>
