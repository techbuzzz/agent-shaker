<script setup lang="ts">
const { isConnected, checkHealth } = useApi()
const { url } = useServerUrl()
const realtime = useRealtime()
const realtimeConnected = realtime.isConnected
const toast = useToastBus()

const status = computed<'online' | 'offline' | 'busy'>(() => {
  if (isConnected.value && realtimeConnected.value) return 'online'
  if (isConnected.value) return 'busy' // API reachable, WS not subscribed
  return 'offline'
})

const label = computed(() => {
  switch (status.value) {
    case 'online': return 'Live'
    case 'busy':   return 'Connected'
    default:       return 'Disconnected'
  }
})

const tooltip = computed(() => {
  const base = url.value
  if (status.value === 'online') return `${base} — live updates active`
  if (status.value === 'busy')   return `${base} — open a project for live updates`
  return `${base} — click to retry`
})

async function retry() {
  if (status.value === 'online') return
  const ok = await checkHealth()
  if (!ok) toast.error('Still cannot reach the MCP server.')
}
</script>

<template>
  <UTooltip :text="tooltip">
    <UButton
      :color="status === 'online' ? 'success' : status === 'busy' ? 'warning' : 'error'"
      variant="subtle"
      size="sm"
      :aria-label="label"
      @click="retry"
    >
      <span class="status-dot" :class="{
        'status-dot--online': status === 'online',
        'status-dot--busy':   status === 'busy',
        'status-dot--offline': status === 'offline'
      }" />
      <span class="hidden sm:inline">{{ label }}</span>
    </UButton>
  </UTooltip>
</template>
