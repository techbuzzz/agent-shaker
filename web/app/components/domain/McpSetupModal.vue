<script setup lang="ts">
import type { AgentConfigFile } from '~/types/api'
import { useMcpSetup, type SetupFile } from '~/composables/useMcpSetup'
import { API_KEY_ENV_VAR, copyToClipboard } from '~/composables/useAgentConfig'
import { NO_SELECTION, optionalId } from '~/utils/select'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  projectName?: string
  agents?: Array<{ id: string; name: string }>
  /** Pre-select an agent, e.g. the one currently active in Settings. */
  defaultAgentId?: string
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const api = useApi()
const toast = useToastBus()
const { downloadFile, downloadAll } = useMcpSetup()

const selectedAgentId = ref<string>(NO_SELECTION)
watch(() => props.modelValue, (v) => {
  if (v) selectedAgentId.value = props.defaultAgentId ?? NO_SELECTION
})

const agentId = computed(() => optionalId(selectedAgentId.value))

/**
 * The files, fetched from the server.
 *
 * This used to be built in the browser from a local template. The generator now
 * lives in `internal/configgen` and is the only implementation — two copies of
 * "what should this config say" is how a config for the wrong client ships with
 * nothing pointing at the bug.
 *
 * Requiring an agent is a consequence, not a restriction: the endpoint renders
 * for one agent, because the endpoint renders for one agent's *credential*. A
 * connection that names no agent has no credential to render around.
 */
const files = ref<AgentConfigFile[]>([])
const endpoint = ref('')
const loading = ref(false)
const loadError = ref('')

async function load() {
  if (!agentId.value) {
    files.value = []
    endpoint.value = ''
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    const result = await api.getAgentConfig(agentId.value)
    files.value = result.files
    endpoint.value = result.endpoints.mcp
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : 'Could not load the configuration'
    files.value = []
  } finally {
    loading.value = false
  }
}

// Reload whenever the binding changes, including on open.
watch([() => props.modelValue, agentId], ([isOpen]) => { if (isOpen) load() }, { immediate: true })

/** Map the server's file shape onto the download helper's. */
function asSetupFiles(list: AgentConfigFile[]): SetupFile[] {
  return list.map((f) => ({ path: f.filename || f.target || 'agent-shaker', contents: f.content }))
}

const previewFor = ref<string | null>(null)
function location(f: AgentConfigFile) { return f.filename || f.target || '' }
function togglePreview(path: string) {
  previewFor.value = previewFor.value === path ? null : path
}

const downloading = ref(false)
function onDownloadAll() {
  if (!files.value.length) return
  downloading.value = true
  try {
    downloadAll(asSetupFiles(files.value))
  } finally {
    // The downloads are staggered internally; this only guards the button.
    setTimeout(() => (downloading.value = false), 800)
  }
}

async function copyEndpoint() {
  if (!endpoint.value) return
  const ok = await copyToClipboard(endpoint.value)
  if (ok) toast.success('Endpoint copied')
  else toast.error('Could not copy — your browser blocked the clipboard.')
}
</script>

<template>
  <UModal v-model:open="open" title="Connect an AI agent" icon="i-lucide-plug">
    <template #body>
      <div class="space-y-4">
        <p class="text-sm text-muted">
          Download a ready-made MCP configuration for
          <span class="font-medium text-default">{{ projectName || 'this project' }}</span>.
          Commit it to the agent's repository and the agent is connected.
        </p>

        <UFormField
          label="Bind an agent"
          help="The configuration is generated for one agent, so its connection can be bound to that agent rather than to the project."
        >
          <USelect
            v-model="selectedAgentId"
            :items="[{ label: '— choose an agent —', value: NO_SELECTION }, ...(agents ?? []).map((a) => ({ label: a.name, value: a.id }))]"
            value-key="value"
          />
        </UFormField>

        <UAlert
          icon="i-lucide-shield"
          color="warning"
          variant="subtle"
          title="No key in these files"
          :description="`They reference ${API_KEY_ENV_VAR} instead, so the files are safe to commit. Issue the key from the agent's Connect action, then put it in the agent's environment.`"
        />

        <UAlert v-if="loadError" icon="i-lucide-circle-alert" color="error" variant="subtle" :description="loadError" />

        <div v-if="loading" class="text-sm text-muted">Generating…</div>

        <div v-else-if="!agentId" class="text-sm text-muted">
          Choose an agent to generate its configuration.
        </div>

        <div v-else-if="files.length" class="border border-default rounded-lg divide-y divide-default overflow-hidden">
          <div v-for="f in files" :key="location(f)" class="p-3">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <p class="font-mono text-sm truncate">{{ location(f) }}</p>
                <p class="text-xs text-muted mt-0.5">{{ f.description }}</p>
              </div>
              <div class="flex gap-1 shrink-0">
                <UButton
                  size="xs" variant="ghost" color="neutral"
                  :icon="previewFor === location(f) ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                  :aria-label="`Preview ${location(f)}`"
                  @click="togglePreview(location(f))"
                />
                <UButton
                  size="xs" variant="outline" color="neutral" icon="i-lucide-download"
                  :aria-label="`Download ${location(f)}`"
                  @click="downloadFile({ path: location(f), contents: f.content })"
                />
              </div>
            </div>
            <pre
              v-if="previewFor === location(f)"
              class="mt-3 text-xs bg-default/50 rounded p-2 overflow-auto max-h-56 font-mono whitespace-pre"
            >{{ f.content }}</pre>
          </div>
        </div>

        <div v-if="endpoint" class="text-xs text-muted space-y-1">
          <p>
            Endpoint:
            <code class="font-mono">{{ endpoint }}</code>
            <UButton size="xs" variant="ghost" color="neutral" icon="i-lucide-copy" @click="copyEndpoint" />
          </p>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Close</UButton>
        <UButton
          color="primary" icon="i-lucide-download" :loading="downloading"
          :disabled="!files.length"
          @click="onDownloadAll"
        >
          Download all
        </UButton>
      </div>
    </template>
  </UModal>
</template>