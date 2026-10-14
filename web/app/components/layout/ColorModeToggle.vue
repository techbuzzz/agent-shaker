<script setup lang="ts">
const colorMode = useColorMode()

/**
 * The active colour mode is browser-only knowledge (cookie + localStorage), so
 * the server cannot know which icon to render. A `mounted` ref was not enough:
 * AppHeader's onMounted runs before the async page subtree has hydrated, and the
 * client then compared "sun" against a server that had rendered "moon".
 *
 * <ClientOnly> removes the race by construction. The fallback keeps the same
 * icon name and button size, so there is no visible shift when it swaps.
 */
const isDark = computed(() => colorMode.value === 'dark')

function toggle() {
  colorMode.preference = isDark.value ? 'light' : 'dark'
}
</script>

<template>
  <ClientOnly>
    <UTooltip :text="isDark ? 'Switch to light mode' : 'Switch to dark mode'">
      <UButton
        :icon="isDark ? 'i-lucide-sun' : 'i-lucide-moon'"
        variant="ghost"
        color="neutral"
        size="sm"
        :aria-label="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
        @click="toggle"
      />
    </UTooltip>

    <template #fallback>
      <UTooltip text="Switch to dark mode">
        <UButton
          icon="i-lucide-moon"
          variant="ghost"
          color="neutral"
          size="sm"
          aria-label="Switch to dark mode"
        />
      </UTooltip>
    </template>
  </ClientOnly>
</template>