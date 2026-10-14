<script setup lang="ts">
const { isConnected, checkHealth } = useApi()
const { url } = useServerUrl()
const realtime = useRealtime()
const realtimeConnected = realtime.isConnected
const toast = useToastBus()

/**
 * Reachability and colour-mode are browser-only facts, and both resolved before
 * Vue finished hydrating the surrounding tree. A plain `mounted` ref was not
 * enough: the layout's onMounted fires ahead of its async page subtree, so the
 * client compared "busy"/"sun" against a server that had rendered
 * "offline"/"moon".
 *
 * <ClientOnly> removes the race by construction — the server renders the
 * placeholder below and the browser swaps in the real control once the app has
 * hydrated. The placeholder keeps the button's dimensions so nothing shifts.
 */
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

const color = computed<'success' | 'warning' | 'error'>(() =>
  status.value === 'online' ? 'success' : status.value === 'busy' ? 'warning' : 'error'
)
// The base 'status-dot' class lives on the element; only the state modifier
// is computed, so the two cannot disagree and the class list stays clean.
const dotModifier = computed(() => `status-dot--${status.value}`)
</script>

<template>
  <ClientOnly>
    <UTooltip :text="tooltip">
      <UButton
        :color="color"
        variant="subtle"
        size="sm"
        :aria-label="label"
        @click="retry"
      >
        <span class="status-dot" :class="dotModifier" />
        <span class="hidden sm:inline">{{ label }}</span>
      </UButton>
    </UTooltip>

    <template #fallback>
      <UButton color="neutral" variant="subtle" size="sm" aria-label="Checking connection" disabled>
        <span class="status-dot status-dot--offline" />
        <span class="hidden sm:inline">Checking</span>
      </UButton>
    </template>
  </ClientOnly>
</template>