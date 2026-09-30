<script setup lang="ts">
import type { NuxtError } from '#app'

const props = defineProps<{ error: NuxtError }>()

const isNotFound = computed(() => props.error?.statusCode === 404)

function handleHome() {
  return clearError({ redirect: '/' })
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-default text-default p-6">
    <UCard class="max-w-md w-full" :ui="{ body: { padding: 'p-8' } }">
      <div class="text-center">
        <div class="text-6xl mb-2 font-bold bg-gradient-to-br from-indigo-500 to-violet-600 bg-clip-text text-transparent">
          {{ error?.statusCode || 500 }}
        </div>
        <h1 class="text-xl font-semibold mb-2">
          {{ isNotFound ? 'Page not found' : 'Something went wrong' }}
        </h1>
        <p class="text-muted mb-6">
          {{ error?.message || 'An unexpected error occurred.' }}
        </p>
        <UButton color="primary" @click="handleHome">Back to Dashboard</UButton>
      </div>
    </UCard>
  </div>
</template>
