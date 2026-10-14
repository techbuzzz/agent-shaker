<script setup lang="ts">
/**
 * One-time display of a freshly minted API key.
 *
 * The plaintext exists in exactly one place — the response to the mint call —
 * because the server keeps only a digest. This modal therefore:
 *
 *   - takes the secret as a prop and never copies it anywhere else (no store,
 *     no localStorage, no log, no query string);
 *   - does not reveal it until the operator asks for it;
 *   - copies to the clipboard on an explicit click only;
 *   - clears it on close and on unmount, so navigating away cannot leave a live
 *     credential sitting in memory.
 */
import type { APIKey, IssuedAPIKey } from '~/types/admin'

const props = defineProps<{
  modelValue: boolean
  issued: IssuedAPIKey | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'dismissed'): void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => {
    emit('update:modelValue', v)
    // Any dismissal (button, escape, overlay) drops the secret. The parent
    // nulls `issued`, which is what makes it unrecoverable.
    if (!v) emit('dismissed')
  }
})

const revealed = ref(false)
const copied = ref(false)
const secret = ref<string | null>(null)

watch(() => props.modelValue, (v) => {
  if (v) {
    // Copied into a local ref so the rendered node never reads straight off a
    // prop that the parent may null while the modal is still animating out.
    secret.value = props.issued?.secret ?? null
    revealed.value = false
    copied.value = false
  }
})

onBeforeUnmount(() => {
  secret.value = null
})

// The parent nulls `issued` on dismissal; mirror that here so the local copy
// dies with it rather than lingering until the next mint.
watch(() => props.issued, (v) => {
  if (!v) secret.value = null
})

const key = computed<APIKey | null>(() => props.issued)

async function copy() {
  if (!secret.value) return
  try {
    await navigator.clipboard.writeText(secret.value)
    copied.value = true
  } catch {
    // Clipboard access can be denied by permissions policy or an insecure
    // context; revealing the secret is still a working path.
    revealed.value = true
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Your new API key"
    icon="i-lucide-key-round"
    :close-on-click-outside="false"
  >
    <template #body>
      <div v-if="issued" class="space-y-4">
        <div class="flex items-start gap-2 p-3 rounded-lg bg-warning/10 border border-warning/40 text-sm">
          <UIcon name="i-lucide-triangle-alert" class="w-4 h-4 mt-0.5 text-warning flex-shrink-0" />
          <p>
            This secret is shown <span class="font-semibold">once</span>. The server stores only a
            digest and cannot show it again — if you lose it, revoke this key and issue a new one.
          </p>
        </div>

        <div class="space-y-1">
          <div class="text-xs uppercase tracking-wide text-muted">Secret</div>
          <div class="flex items-start gap-2">
            <code
              class="flex-1 min-w-0 font-mono text-xs break-all p-3 rounded-lg bg-elevated border border-default select-all"
              :class="revealed ? '' : 'blur-sm select-none'"
              data-testid="issued-secret"
            >{{ secret ?? '' }}</code>
            <UButton
              :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
              variant="soft"
              color="neutral"
              size="sm"
              :aria-label="copied ? 'Copied' : 'Copy secret'"
              @click="copy"
            />
          </div>
          <p v-if="copied" class="text-xs text-success">Copied to clipboard.</p>
        </div>

        <dl v-if="key" class="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
          <dt class="text-muted">Key id</dt>
          <dd class="font-mono break-all">{{ key.id }}</dd>
          <dt class="text-muted">Prefix</dt>
          <dd class="font-mono">{{ key.key_prefix }}…</dd>
          <dt class="text-muted">Principal</dt>
          <dd>{{ key.principal_id }}</dd>
          <dt class="text-muted">Scopes</dt>
          <dd>{{ (key.scopes ?? []).join(', ') || 'unrestricted' }}</dd>
        </dl>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton
          variant="ghost"
          color="neutral"
          icon="i-lucide-eye"
          :disabled="revealed"
          @click="revealed = true"
        >
          Reveal
        </UButton>
        <UButton color="error" icon="i-lucide-x" @click="open = false">
          I have saved it — close
        </UButton>
      </div>
    </template>
  </UModal>
</template>