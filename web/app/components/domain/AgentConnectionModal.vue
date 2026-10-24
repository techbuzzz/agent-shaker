<script setup lang="ts">
import type { Agent, AgentConfigFile, ConfigFormat } from '~/types/api'
import { CONFIG_FORMATS, CONFIG_FORMAT_LABELS, RENDERABLE_TOOL_TYPES } from '~/types/api'
import {
  API_KEY_ENV_VAR,
  copyToClipboard,
  downloadFile,
  fileLocation,
  renderedFile
} from '~/composables/useAgentConfig'

const props = defineProps<{
  modelValue: boolean
  agent: Agent | null
  /** Format preselected from the agent's tool_type; the server decides otherwise. */
  suggestedFormat?: ConfigFormat
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const api = useApi()
const toast = useToastBus()

type Step = 'choose' | 'issued'
const step = ref<Step>('choose')

const format = ref<ConfigFormat>('generic')
const files = ref<AgentConfigFile[]>([])
/**
 * The one and only copy of the plaintext.
 *
 * Held here, in a single ref, and cleared when the modal closes — because the
 * server keeps only the hash and cannot return this again. Everything the
 * operator copies is rendered from it on demand.
 */
const secret = ref('')

const minting = ref(false)
const error = ref('')
const expanded = ref<string | null>(null)
const copied = ref<string | null>(null)

/** Whether this agent's tool has a dedicated layout, or only the generic one. */
const hasDedicatedLayout = computed(() =>
  props.agent ? RENDERABLE_TOOL_TYPES.includes(props.agent.tool_type) : false
)

const formatItems = computed(() =>
  CONFIG_FORMATS.map((f) => ({
    value: f,
    // Only offer what fits: a Claude Code agent handed a `.cursor/mcp.json`
    // produces a connection that silently does not work.
    label: CONFIG_FORMAT_LABELS[f]
  }))
)

/**
 * The layout this agent's tool type maps to.
 *
 * Mirrors `models.ToolType.DefaultFormat` on the Go side, and the server is the
 * authority: it re-derives the format from the agent row when no format is
 * named, so a mismatch here only changes which entry starts selected, never what
 * is rendered.
 */
function defaultFormatFor(agent: Agent): ConfigFormat {
  switch (agent.tool_type) {
    case 'claude-code':
      return 'claude'
    case 'codex':
      return 'codex'
    case 'opencode':
      return 'opencode'
    case 'cursor':
      return 'cursor'
    case 'copilot-vs':
      return 'vscode'
    case 'copilot-cli':
      return 'copilot'
    default:
      return 'generic'
  }
}

watch(
  () => props.modelValue,
  (isOpen) => {
    if (!isOpen) {
      // Drop the secret on close. Leaving it in a ref would keep a live
      // credential in a component that outlives the operator's visit, reachable
      // from anything else that holds a reference to it.
      secret.value = ''
      files.value = []
      step.value = 'choose'
      error.value = ''
      expanded.value = null
      return
    }
    format.value = props.agent
      ? (props.suggestedFormat ?? defaultFormatFor(props.agent))
      : 'generic'
  }
)

async function mint() {
  if (!props.agent) return
  minting.value = true
  error.value = ''
  try {
    const result = await api.mintAgentConnection(props.agent.id, { format: format.value })
    secret.value = result.key.secret
    files.value = result.files
    format.value = result.format
    step.value = 'issued'
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not create the connection'
  } finally {
    minting.value = false
  }
}

/** Render one file with the secret in place. */
function withSecret(file: AgentConfigFile): string {
  return renderedFile(file, secret.value)
}

async function copyFile(file: AgentConfigFile) {
  const ok = await copyToClipboard(withSecret(file))
  if (ok) {
    copied.value = fileLocation(file)
    setTimeout(() => (copied.value = null), 2000)
  } else {
    toast.error('Could not copy — your browser blocked the clipboard. Select the text instead.')
  }
}

async function copySecret() {
  const ok = await copyToClipboard(secret.value)
  if (ok) {
    copied.value = 'key'
    setTimeout(() => (copied.value = null), 2000)
  } else {
    toast.error('Could not copy the key. Select it and copy manually.')
  }
}

/**
 * Download keeps the placeholder, not the secret.
 *
 * A file that can be committed is worth more than one that works the instant it
 * is written: the developer sets AGENT_SHAKER_API_KEY in the agent's environment
 * anyway, and a file carrying a live key would put it into git history.
 */
function saveFile(file: AgentConfigFile) {
  downloadFile(fileLocation(file).split('/').pop()!, file.content)
}

function toggle(path: string) {
  expanded.value = expanded.value === path ? null : path
}
</script>

<template>
  <UModal v-model:open="open" title="Connect this agent" icon="i-lucide-plug">
    <template #body>
      <div v-if="!agent" class="text-sm text-muted">Select an agent first.</div>

      <div v-else-if="step === 'choose'" class="space-y-4">
        <p class="text-sm text-muted">
          Creating a connection issues a credential for
          <span class="font-medium text-default">{{ agent.name }}</span>. The agent
          will act as exactly this row in Agent Shaker — no other agent's tasks,
          no other agent's handoffs.
        </p>

        <UFormField
          label="Configuration format"
          help="Where the file goes. Pick the one matching the tool this agent runs in."
        >
          <USelect v-model="format" :items="formatItems" value-key="value" class="w-full" />
        </UFormField>

        <UAlert
          v-if="!hasDedicatedLayout"
          icon="i-lucide-info"
          color="neutral"
          variant="subtle"
          title="No dedicated layout for this tool yet"
          :description="`The generic .mcp.json is what most MCP hosts read, so this still connects. A dedicated Windsurf layout arrives in a later phase.`"
        />

        <UAlert
          icon="i-lucide-shield"
          color="warning"
          variant="subtle"
          title="This issues a new credential every time"
          :description="`The previous one stays valid until you revoke it. If you retry after a timeout, revoke the extra key in Access.`"
        />

        <UAlert v-if="error" icon="i-lucide-circle-alert" color="error" variant="subtle" :description="error" />

        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
          <UButton color="primary" :loading="minting" icon="i-lucide-key-round" @click="mint">
            Create connection
          </UButton>
        </div>
      </div>

      <div v-else class="space-y-4">
        <UAlert
          icon="i-lucide-shield-check"
          color="success"
          variant="subtle"
          title="Copy this key now — it is not shown again"
          :description="`The server keeps only a hash of it, so if you lose it you will have to issue another credential.`"
        />

        <div class="rounded-lg border border-default p-3 space-y-2">
          <div class="flex items-center justify-between gap-2">
            <span class="text-xs uppercase tracking-wide text-muted">Agent credential</span>
            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              :icon="copied === 'key' ? 'i-lucide-check' : 'i-lucide-copy'"
              @click="copySecret"
            >
              {{ copied === 'key' ? 'Copied' : 'Copy' }}
            </UButton>
          </div>
          <code class="block text-xs break-all select-all">{{ secret }}</code>
          <p class="text-xs text-muted">
            Put it in the agent's environment as
            <code>{{ API_KEY_ENV_VAR }}</code>. It never appears in the files below.
          </p>
        </div>

        <div class="border border-default rounded-lg divide-y divide-default overflow-hidden">
          <div v-for="file in files" :key="fileLocation(file)" class="p-3">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <code class="text-sm font-medium">{{ fileLocation(file) }}</code>
                <p class="text-xs text-muted mt-0.5">{{ file.description }}</p>
                <p v-if="file.install" class="text-xs text-muted mt-1">
                  <code>{{ file.install }}</code>
                </p>
              </div>
              <div class="flex gap-1 shrink-0">
                <UButton
                  size="xs"
                  variant="soft"
                  color="neutral"
                  :icon="copied === fileLocation(file) ? 'i-lucide-check' : 'i-lucide-copy'"
                  @click="copyFile(file)"
                >
                  {{ copied === fileLocation(file) ? 'Copied' : 'Copy' }}
                </UButton>
                <UButton size="xs" variant="soft" color="neutral" icon="i-lucide-download" @click="saveFile(file)" />
              </div>
            </div>

            <button
              type="button"
              class="text-xs text-primary hover:underline mt-2"
              @click="toggle(fileLocation(file))"
            >
              {{ expanded === fileLocation(file) ? 'Hide' : 'Preview' }}
            </button>

            <pre
              v-if="expanded === fileLocation(file)"
              class="mt-2 max-h-64 overflow-auto rounded bg-elevated p-2 text-xs whitespace-pre-wrap break-all"
            >{{ withSecret(file) }}</pre>
          </div>
        </div>

        <p class="text-xs text-muted">
          <span class="font-medium">Copy</span> puts the key into the file.
          <span class="font-medium">Download</span> keeps
          <code>{{ API_KEY_ENV_VAR }}</code> in it, so the file is safe to commit.
        </p>

        <div class="flex justify-end">
          <UButton color="primary" @click="open = false">Done</UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>