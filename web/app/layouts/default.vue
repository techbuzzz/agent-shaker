<script setup lang="ts">
// Default layout: vertical sidebar + main column with optional sticky header.
const route = useRoute()
// `url` lives on useServerUrl (it is the server-URL cookie), not on useApi.
// Destructuring it from useApi used to yield undefined, which made the watcher
// below a no-op and made `url.value` throw whenever the health check failed.
const { url } = useServerUrl()
const { isConnected, checkHealth } = useApi()
const toast = useToastBus()

// Run a health check on first mount and whenever the URL changes.
watch(url, async () => {
  const ok = await checkHealth()
  if (!ok) toast.warn(`Cannot reach MCP server at ${url.value}. Open Settings to fix.`)
}, { immediate: true })

// Detect mobile to collapse the sidebar.
const sidebarOpen = ref(true)
const isSmall = useMediaQuery('(max-width: 1024px)')
watch(isSmall, (v) => { sidebarOpen.value = !v }, { immediate: true })
</script>

<template>
  <div class="min-h-screen bg-default text-default">
    <div class="flex min-h-screen">
      <AppSidebar
        :collapsed="!sidebarOpen"
        @toggle="sidebarOpen = !sidebarOpen"
      />
      <div class="flex-1 flex flex-col min-w-0">
        <AppHeader @toggle-sidebar="sidebarOpen = !sidebarOpen" />
        <main class="flex-1 px-4 sm:px-6 lg:px-8 py-6 lg:py-8 max-w-screen-2xl w-full mx-auto">
          <slot />
        </main>
        <footer class="border-t border-default py-4 px-6 text-center text-xs text-muted">
          <span>© 2026 Agent Shaker — MCP Task Tracker · {{ route.path }}</span>
        </footer>
      </div>
    </div>
  </div>
</template>
