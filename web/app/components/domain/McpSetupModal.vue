<script setup lang="ts">
import { useMcpSetup, buildSetupFiles, API_KEY_ENV_VAR, type SetupFile } from '~/composables/useMcpSetup'

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

const { downloadFile, downloadAll } = useMcpSetup()

const selectedAgentId = ref('')
watch(() => props.modelValue, (v) => {
  if (v) selectedAgentId.value = props.defaultAgentId ?? ''
})

/**
 * The origin is read from the running page, not from configuration.
 *
 * That is what makes a single generated config correct in every topology: with
 * the TLS edge the browser origin *is* the public origin, and without it the
 * browser origin is the Nuxt service. Either way the agent is pointed at
 * something reachable from outside the container.
 */
const origin = import.meta.client ? window.location.origin : ''

const input = computed(() => ({
  origin,
  projectId: props.projectId,
  projectName: props.projectName,
  agentId: selectedAgentId.value || undefined,
  agentName: props.agents?.find((a) => a.id === selectedAgentId.value)?.name
}))

/**
 * The actual file contents, rebuilt whenever the agent binding changes.
 *
 * Computed once and used for BOTH the preview and the per-file download. An
 * earlier draft derived them lazily per click, which meant a download could
 * fire with empty contents — the two paths have to read the same source.
 */
const files = computed<SetupFile[]>(() => buildSetupFiles(input.value))

const DESCRIPTIONS: Record<string, string> = {
  '.mcp.json': 'Visual Studio 2026 and MCP hosts using this convention',
  '.vscode/mcp.json': 'Visual Studio Code',
  '.github/copilot-instructions.md': 'Tells Copilot what this server is for and how to work with it',
  'AGENT_SHAKER_SETUP.md': 'Step-by-step setup, including how to verify the connection'
}

const listed = computed(() =>
  files.value.map((f) => ({ file: f, description: DESCRIPTIONS[f.path] ?? '' }))
)

/** Only one preview is expanded at a time; the markdown files are long. */
const previewFor = ref<string | null>(null)
function togglePreview(path: string) {
  previewFor.value = previewFor.value === path ? null : path
}

const downloading = ref(false)
function onDownloadAll() {
  downloading.value = true
  try {
    downloadAll(input.value)
  } finally {
    // The downloads are staggered internally; this only guards the button.
    setTimeout(() => (downloading.value = false), 800)
  }
}

function copyEndpoint() {
  navigator.clipboard?.writeText(`${origin}/mcp?project_id=${props.projectId}`)
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
          label="Bind an agent (optional)"
          help="Scopes the connection so project-scoped tools work without restating identity. Leave unset if the agent should call register_self."
        >
          <USelect
            v-model="selectedAgentId"
            :items="[{ label: '— not bound —', value: '' }, ...(agents ?? []).map((a) => ({ label: a.name, value: a.id }))]"
            value-key="value"
          />
        </UFormField>

        <UAlert
          icon="i-lucide-shield"
          color="warning"
          variant="subtle"
          title="No key in these files"
          :description="`They reference ${API_KEY_ENV_VAR} instead, so the files are safe to commit. The agent picks the key up from its own environment.`"
        />

        <div class="border border-default rounded-lg divide-y divide-default overflow-hidden">
          <div v-for="entry in listed" :key="entry.file.path" class="p-3">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <p class="font-mono text-sm truncate">{{ entry.file.path }}</p>
                <p class="text-xs text-muted mt-0.5">{{ entry.description }}</p>
              </div>
              <div class="flex gap-1 shrink-0">
                <UButton
                  size="xs" variant="ghost" color="neutral"
                  :icon="previewFor === entry.file.path ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                  :aria-label="`Preview ${entry.file.path}`"
                  @click="togglePreview(entry.file.path)"
                />
                <UButton
                  size="xs" variant="outline" color="neutral" icon="i-lucide-download"
                  :aria-label="`Download ${entry.file.path}`"
                  @click="downloadFile(entry.file)"
                />
              </div>
            </div>
            <pre
              v-if="previewFor === entry.file.path"
              class="mt-3 text-xs bg-default/50 rounded p-2 overflow-auto max-h-56 font-mono whitespace-pre"
            >{{ entry.file.contents }}</pre>
          </div>
        </div>

        <div class="text-xs text-muted space-y-1">
          <p>
            Endpoint:
            <code class="font-mono">{{ origin }}/mcp?project_id={{ projectId }}</code>
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
          @click="onDownloadAll"
        >
          Download all
        </UButton>
      </div>
    </template>
  </UModal>
</template>
