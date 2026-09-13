<script setup lang="ts">
// Global app shell. Nuxt UI's <UApp> provides overlay/toast/tooltip context.
// NuxtLayout picks layouts/default.vue (or layouts/blank.vue per page).
</script>

<template>
  <UApp>
    <NuxtLayout>
      <NuxtPage :transition="{ name: 'page', mode: 'out-in' }" />
    </NuxtLayout>
  </UApp>
</template>
