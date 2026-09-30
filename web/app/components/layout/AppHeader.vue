<script setup lang="ts">
defineEmits<{ (e: 'toggle-sidebar'): void }>()
</script>

<template>
  <header class="sticky top-0 z-30 h-16 bg-default/80 backdrop-blur-md border-b border-default">
    <div class="h-full px-4 sm:px-6 flex items-center justify-between gap-3">
      <!-- Mobile sidebar trigger -->
      <UButton
        icon="i-lucide-menu"
        variant="ghost"
        color="neutral"
        size="sm"
        class="lg:hidden"
        aria-label="Toggle navigation"
        @click="$emit('toggle-sidebar')"
      />

      <!-- Center: connection status (desktop & mobile) -->
      <div class="flex-1 flex items-center justify-end gap-2 sm:gap-3">
        <ConnectionStatus />
        <ColorModeToggle />
        <UButton
          icon="i-lucide-settings"
          variant="ghost"
          color="neutral"
          size="sm"
          aria-label="Settings"
          to="/settings"
        />
      </div>
    </div>
  </header>
</template>
