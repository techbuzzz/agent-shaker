<script setup lang="ts">
/**
 * /access — the principal directory and the key inventory.
 *
 * Both surfaces are gated by the `admin` scope server-side (see
 * RequireScopeOnPaths in cmd/server/routes.go), so a read-only or write-only
 * caller sees a 403 here rather than a filtered list; this page renders that
 * as an explicit error instead of an empty directory, because "no principals"
 * and "you may not look" are different facts.
 */
import type { APIKey, CreateKeyInput, CreatePersonInput, IssuedAPIKey, Person } from '~/types/admin'
import type { Project } from '~/types/api'

useHead({ title: 'Access · Agent Shaker' })

const api = useApi()
import { NO_SELECTION, isNone } from '~/utils/select'
const toast = useToastBus()
const projectsStore = useProjectsStore()

const peopleError = ref<string | null>(null)
const keysError = ref<string | null>(null)
const includeRevoked = ref(false)
const principalFilter = ref<string>(NO_SELECTION)

const { data: people, status: peopleStatus, refresh: refreshPeople } = await useAsyncData(
  'access:people',
  async () => {
    try {
      return await api.listPeople()
    } catch (err: unknown) {
      peopleError.value = errMessage(err, 'Could not load principals')
      return [] as Person[]
    }
  },
  { default: () => [] as Person[] }
)

const { data: keys, status: keysStatus, refresh: refreshKeys } = await useAsyncData(
  'access:keys',
  async () => {
    try {
      keysError.value = null
      return await api.listKeys({ include_revoked: includeRevoked.value })
    } catch (err: unknown) {
      keysError.value = errMessage(err, 'Could not load API keys')
      return [] as APIKey[]
    }
  },
  { default: () => [] as APIKey[], watch: [includeRevoked] }
)

const projects = computed<Project[]>(() => projectsStore.recent)
const projectNames = computed(() => Object.fromEntries(projects.value.map(p => [p.id, p.name])))

const filteredKeys = computed(() => {
  if (isNone(principalFilter.value)) return keys.value ?? []
  return (keys.value ?? []).filter(k => k.principal_id === principalFilter.value)
})

const principalOptions = computed(() => [
  { label: 'All principals', value: NO_SELECTION },
  ...(people.value ?? []).map(p => ({ label: p.display_name, value: p.id }))
])

// Re-fetch on focus so a key issued from another tab shows up here.
onMounted(() => { refreshKeys() })

// --- Modals ---
const showPersonForm = ref(false)
const showKeyForm = ref(false)
const mintFor = ref<string>('')

const revoking = ref<APIKey | null>(null)
const revokingOpen = computed({
  get: () => revoking.value !== null,
  set: (v: boolean) => { if (!v) revoking.value = null }
})

/**
 * The plaintext secret, held here and nowhere else. It is cleared the moment
 * the operator dismisses the dialog and on unmount, so it cannot survive a
 * navigation or be read back from any store.
 */
const issued = ref<IssuedAPIKey | null>(null)
const issuedOpen = ref(false)

function onSecretDismissed() {
  issued.value = null
}

/**
 * Unwraps the backend error envelope `{"error":{"code","message","request_id"}}`
 * (internal/httpx/errors.go), falling back to the transport message.
 *
 * Only used for the write paths that have no dialog of their own — revoke and
 * the other inline actions. A failed create/edit is reported by the form modal
 * itself, which keeps the message on screen while the input that caused it is
 * still there.
 */
function errMessage(err: unknown, fallback: string): string {
  const fe = err as { data?: { error?: { message?: string } }; message?: string }
  return fe?.data?.error?.message ?? fe?.message ?? fallback
}

async function handleCreatePerson(input: CreatePersonInput) {
  await api.createPerson(input)
  await refreshPeople()
  toast.success(`${input.display_name} added to the directory`)
}

function openMint(principalId = '') {
  mintFor.value = principalId
  showKeyForm.value = true
}

async function handleCreateKey(input: CreateKeyInput) {
  const result = await api.createKey(input)
  await refreshKeys()
  issued.value = result
  issuedOpen.value = true
}

async function confirmRevoke() {
  if (!revoking.value) return
  const target = revoking.value
  try {
    // Idempotent server-side: revoking an already-revoked key is a 200, not a 409.
    await api.revokeKey(target.id)
    await refreshKeys()
    revoking.value = null
    toast.success(`${target.key_prefix}… revoked`)
  } catch (err: unknown) {
    toast.error(errMessage(err, 'Could not revoke key'))
  }
}
</script>

<template>
  <div>
    <div class="mb-6 flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold flex items-center gap-2">
          <UIcon name="i-lucide-shield-check" class="text-primary w-6 h-6" /> People &amp; keys
        </h1>
        <p class="text-muted mt-1">
          Principals and the credentials minted for them. Requires the
          <code class="font-mono">admin</code> scope.
        </p>
      </div>
    </div>

    <!-- Principals -->
    <section class="mb-10">
      <div class="flex flex-wrap items-center justify-between gap-3 mb-3">
        <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">
          Principals
          <span class="font-normal normal-case">({{ people?.length ?? 0 }})</span>
        </h2>
        <UButton icon="i-lucide-plus" color="primary" size="sm" @click="showPersonForm = true">
          Add principal
        </UButton>
      </div>

      <div v-if="peopleError" class="flex items-start gap-2 p-3 rounded-lg bg-error/10 border border-error/40 text-sm mb-3">
        <UIcon name="i-lucide-triangle-alert" class="w-4 h-4 mt-0.5 text-error flex-shrink-0" />
        <div>
          <p>{{ peopleError }}</p>
          <UButton variant="link" color="error" size="xs" @click="refreshPeople()">Retry</UButton>
        </div>
      </div>

      <EmptyState
        v-else-if="peopleStatus === 'success' && !people?.length"
        icon="i-lucide-users"
        title="No principals yet"
        description="Add a human approver or a service principal — every API key belongs to one of them."
      >
        <UButton icon="i-lucide-plus" color="primary" size="sm" @click="showPersonForm = true">
          Add principal
        </UButton>
      </EmptyState>

      <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <PersonCard
          v-for="p in people" :key="p.id"
          :person="p"
          @show-keys="(person) => openMint(person.id)"
        />
      </div>
    </section>

    <USeparator class="mb-10" />

    <!-- Keys -->
    <section>
      <div class="flex flex-wrap items-center justify-between gap-3 mb-3">
        <h2 class="text-sm font-semibold uppercase tracking-wide text-muted">
          API keys
          <span class="font-normal normal-case">({{ filteredKeys.length }})</span>
        </h2>
        <div class="flex flex-wrap items-center gap-2">
          <USelectMenu
            v-model="principalFilter"
            :items="principalOptions"
            value-key="value"
            size="sm"
            placeholder="All principals"
          />
          <USwitch
            v-model="includeRevoked"
            label="Include revoked"
            size="sm"
          />
          <UButton icon="i-lucide-plus" color="primary" size="sm" @click="openMint()">
            Issue key
          </UButton>
        </div>
      </div>

      <div v-if="keysError" class="flex items-start gap-2 p-3 rounded-lg bg-error/10 border border-error/40 text-sm mb-3">
        <UIcon name="i-lucide-triangle-alert" class="w-4 h-4 mt-0.5 text-error flex-shrink-0" />
        <div>
          <p>{{ keysError }}</p>
          <UButton variant="link" color="error" size="xs" @click="refreshKeys()">Retry</UButton>
        </div>
      </div>

      <EmptyState
        v-else-if="keysStatus === 'success' && !filteredKeys.length"
        icon="i-lucide-key-round"
        title="No keys to show"
        :description="includeRevoked
          ? 'Nothing has been issued for this principal yet.'
          : 'No active keys. Revoked keys are hidden — use the switch above to see them.'"
      >
        <UButton icon="i-lucide-plus" color="primary" size="sm" :disabled="!people?.length" @click="openMint()">
          Issue key
        </UButton>
      </EmptyState>

      <div v-else class="space-y-3">
        <KeyCard
          v-for="k in filteredKeys" :key="k.id"
          :api-key="k"
          :project-name="k.project_id ? projectNames[k.project_id] : undefined"
          @revoke="(key) => revoking = key"
        />
      </div>
    </section>

    <PersonFormModal v-model="showPersonForm" :on-submit="handleCreatePerson" />

    <KeyFormModal
      v-model="showKeyForm"
      :people="people ?? []"
      :projects="projects"
      :principal-id="mintFor"
      :on-submit="handleCreateKey"
    />

    <IssuedKeyModal v-model="issuedOpen" :issued="issued" @dismissed="onSecretDismissed" />

    <!--
      Inline confirmation rather than the shared ConfirmDialog: that component
      binds `modelValue` on UModal, while Nuxt UI v4 drives the modal through
      `open` (every other modal here uses v-model:open).
    -->
    <UModal v-model:open="revokingOpen" title="Revoke this key?" icon="i-lucide-triangle-alert">
      <template #body>
        <p class="text-sm text-muted">
          <code class="font-mono text-highlighted">{{ revoking?.key_prefix }}…</code> stops working
          immediately. Any client still holding it will get a 401 on its next call, and the key cannot
          be restored — issue a new one instead.
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" color="neutral" @click="revokingOpen = false">Cancel</UButton>
          <UButton color="error" @click="confirmRevoke">Revoke key</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>