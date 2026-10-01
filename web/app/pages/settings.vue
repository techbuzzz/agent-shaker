<script setup lang="ts">
definePageMeta({ layout: 'blank' })
useHead({ title: 'Settings · Agent Shaker' })

const { url, set, reset, isDefault } = useServerUrl()
const { checkHealth, isConnected } = useApi()
const toast = useToastBus()
const settings = useSettingsStore()

const input = ref(url.value)
const testing = ref(false)
const testResult = ref<'idle' | 'ok' | 'fail'>('idle')

watch(url, (v) => { input.value = v; testResult.value = 'idle' })

async function test() {
  testing.value = true
  await set(input.value)
  const ok = await checkHealth()
  testResult.value = ok ? 'ok' : 'fail'
  testing.value = false
}

async function save() {
  await set(input.value)
  toast.success('Server URL saved')
  await checkHealth()
}

async function doReset() {
  await reset()
  input.value = url.value
  testResult.value = 'idle'
  toast.info('Reset to default URL')
}

const colorMode = useColorMode()
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-6">
    <UCard class="w-full max-w-2xl" :ui="{ body: 'p-6 sm:p-8' }">
      <template #header>
        <div class="flex items-center gap-3">
          <NuxtLink to="/" class="flex items-center gap-2 text-sm text-muted hover:text-primary">
            <UIcon name="i-lucide-arrow-left" class="w-4 h-4" /> Back to app
          </NuxtLink>
        </div>
      </template>

      <h1 class="text-xl font-semibold mb-1">Settings</h1>
      <p class="text-sm text-muted mb-6">Configure how this UI connects to your MCP backend.</p>

      <!-- Connection -->
      <section class="space-y-4">
        <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">Connection</h2>
        <UFormField label="MCP Server URL" required>
          <UInput
            v-model="input"
            type="url"
            placeholder="http://localhost:8080"
            :trailing-icon="isConnected ? 'i-lucide-check-circle' : 'i-lucide-x-circle'"
          />
        </UFormField>
        <p class="text-xs text-muted">
          Includes protocol and host (e.g. <code>http://10.0.0.42:8080</code>). The path <code>/api</code> is appended automatically.
        </p>

        <div class="flex flex-wrap gap-2 items-center">
          <UButton color="primary" :loading="testing" @click="test">Test connection</UButton>
          <UButton variant="ghost" color="neutral" @click="save">Save</UButton>
          <UButton variant="link" color="neutral" :disabled="isDefault" @click="doReset">Reset to default</UButton>
          <span v-if="testResult === 'ok'" class="text-sm text-success inline-flex items-center gap-1">
            <UIcon name="i-lucide-check-circle" class="w-4 h-4" /> Reachable
          </span>
          <span v-else-if="testResult === 'fail'" class="text-sm text-error inline-flex items-center gap-1">
            <UIcon name="i-lucide-x-circle" class="w-4 h-4" /> Not reachable
          </span>
        </div>
      </section>

      <USeparator class="my-6" />

      <!-- Appearance -->
      <section class="space-y-4">
        <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">Appearance</h2>
        <div class="flex items-center justify-between">
          <div>
            <div class="font-medium">Theme</div>
            <p class="text-xs text-muted">Switches between light and dark. Persisted across sessions.</p>
          </div>
          <USelectMenu
            :model-value="colorMode.preference"
            :items="[
              { label: 'System', value: 'system' },
              { label: 'Light', value: 'light' },
              { label: 'Dark', value: 'dark' }
            ]"
            value-key="value"
            @update:model-value="(v) => (colorMode.preference = v as 'system' | 'light' | 'dark')"
          />
        </div>

        <div class="flex items-center justify-between">
          <div>
            <div class="font-medium">Table density</div>
            <p class="text-xs text-muted">Affects list spacing.</p>
          </div>
          <USelectMenu
            :model-value="settings.tableDensity"
            :items="[
              { label: 'Comfortable', value: 'comfortable' },
              { label: 'Compact', value: 'compact' }
            ]"
            value-key="value"
            @update:model-value="(v) => settings.setDensity(v as 'comfortable' | 'compact')"
          />
        </div>
      </section>
    </UCard>
  </div>
</template>
