<script setup lang="ts">
const colorMode = useColorMode()

const isDark = computed(() => colorMode.value === 'dark')

function toggle() {
  colorMode.preference = isDark.value ? 'light' : 'dark'
}
</script>

<template>
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
</template>
