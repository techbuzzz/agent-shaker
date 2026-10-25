<script setup lang="ts">
import type { Project } from '~/types/api'
import type { CreateHookInput, Hook, HookRun } from '~/types/hooks'
import { HOOK_EVENT_LABELS, HOOK_RUN_STATUS_LABELS } from '~/types/hooks'
import { formatHookDuration, hookEventLabel, HOOK_RUN_STATUS_COLORS } from '~/utils/hooks'

/**
 * The hook layer's operator surface: configure the commands the server runs
 * when something happens in a project, and read what those commands did.
 *
 * The page is a composition surface. The card, the run row and the form live
 * in `components/domain`, because the list and the log are separate concerns
 * with separate data, and a page that owns both ends up owning the markup for
 * both as well.
 */
useHead({ title: 'Hooks · Agent Shaker' })

const api = useApi()
const toast = useToastBus()
const settings = useSettingsStore()

const hooks = useHooks()
const {
  hooks: hookList,
  runs: runList,
  projectEnabled,
  loading,
  saving,
  loadError,
  runError,
  testingId,
  setProjectEnabled
} = hooks

// --- Project selection ----------------------------------------------------
//
// The project is not in the route: `/hooks` is a top-level page, and the app
// already keeps the project the operator was last working in
// (`settings.lastProjectId`, the `ui:last-project` cookie, written by the
// project page). That is the starting selection here, with a picker on the
// page to change it — the same store, the same cookie, no second notion of
// "the current project".
const { data: projects } = await useFetch<Project[]>('/projects', {
  baseURL: api.apiBase,
  key: 'hooks:projects',
  default: () => []
})

const selectedProjectId = ref<string | null>(settings.lastProjectId)

/**
 * The project the page is about.
 *
 * Falls back to the first project when the remembered one is gone, so the page
 * is never blank for a reason the operator cannot see.
 */
const projectId = computed(() => {
  const list = projects.value ?? []
  if (selectedProjectId.value && list.some((p) => p.id === selectedProjectId.value)) {
    return selectedProjectId.value
  }
  return list[0]?.id ?? null
})

const projectItems = computed(() => (projects.value ?? []).map((p) => ({ label: p.name, value: p.id })))
const currentProjectName = computed(
  () => (projects.value ?? []).find((p) => p.id === projectId.value)?.name ?? 'this project'
)

/** A deliberate choice is remembered; a fallback is not. */
function selectProject(value: unknown): void {
  const next = typeof value === 'string' ? value : null
  selectedProjectId.value = next
  settings.rememberProject(next)
}

watch(
  projectId,
  (id) => {
    // Client-only: the endpoints are relative to the app's own origin, and a
    // server-side render cannot resolve them.
    if (!import.meta.client || !id) return
    void hooks.load(id)
  },
  { immediate: true }
)

// --- The per-project switch ------------------------------------------------
//
// It is separate from each hook's own enabled flag on purpose: the project
// decides whether the server may run commands for it at all, and each hook
// decides whether it runs when it may.
async function toggleProjectEnabled(enabled: boolean): Promise<void> {
  if (!projectId.value) return
  try {
    await setProjectEnabled(projectId.value, enabled)
  } catch {
    // The store records the message; the switch keeps the server's value, so
    // a refused change does not leave the interface lying about the state.
  }
}

// --- Dialogs --------------------------------------------------------------

const formOpen = ref(false)
const editingHook = ref<Hook | null>(null)

function openCreate(): void {
  editingHook.value = null
  formOpen.value = true
}

function openEdit(hook: Hook): void {
  editingHook.value = hook
  formOpen.value = true
}

const deletingHook = ref<Hook | null>(null)
const deletingOpen = computed({
  get: () => deletingHook.value !== null,
  set: (v: boolean) => { if (!v) deletingHook.value = null }
})

/** The run a test produced, and whether its result is on screen. */
const testRun = ref<HookRun | null>(null)
const testRunOpen = ref(false)

/** The run whose captured output is expanded in the log. */
const expandedRunId = ref<string | null>(null)

function toggleRunOutput(id: string): void {
  expandedRunId.value = expandedRunId.value === id ? null : id
}

// --- Actions --------------------------------------------------------------

/**
 * Creates or updates the hook the form is holding.
 *
 * A prop rather than an event, because the dialog has to wait for the answer
 * and report a failure instead of closing over it — see the note in
 * useFormModal.
 */
async function saveHook(input: CreateHookInput): Promise<unknown> {
  const project = projectId.value
  if (!project) throw new Error('Choose a project before saving a hook')
  return editingHook.value
    ? hooks.update(editingHook.value.id, input)
    : hooks.create(project, input)
}

async function handleToggle(hook: Hook): Promise<void> {
  try {
    const updated = await hooks.toggle(hook)
    toast.success(updated.enabled ? 'Hook enabled' : 'Hook disabled')
  } catch (err: unknown) {
    toast.error(describeHookError(err, 'Could not change the hook'))
  }
}

async function handleDelete(): Promise<void> {
  if (!deletingHook.value) return
  try {
    await hooks.remove(deletingHook.value.id)
    toast.success('Hook deleted')
  } catch (err: unknown) {
    toast.error(describeHookError(err, 'Could not delete the hook'))
  } finally {
    deletingHook.value = null
  }
}

/**
 * A dry run against a synthetic payload: it proves the command works without
 * touching a real task, agent or blocker. A hook that can only be validated
 * by firing it at real work is a hook nobody will ever enable.
 */
async function handleTest(hook: Hook): Promise<void> {
  try {
    testRun.value = await hooks.test(hook.id)
    testRunOpen.value = true
  } catch (err: unknown) {
    toast.error(describeHookError(err, 'Could not run the test'))
  }
}
</script>

<template>
  <div>
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
      <div>
        <h1 class="text-2xl font-semibold">Hooks</h1>
        <p class="text-muted mt-1">
          Commands the server runs when something happens in
          <span class="font-medium text-default">{{ currentProjectName }}</span>.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <USelectMenu
          :model-value="projectId ?? undefined"
          :items="projectItems"
          value-key="value"
          size="sm"
          placeholder="Choose a project"
          class="w-56"
          aria-label="Project"
          @update:model-value="selectProject"
        />
        <UButton
          icon="i-lucide-plus"
          size="sm"
          color="primary"
          :disabled="!projectId"
          @click="openCreate"
        >
          New hook
        </UButton>
      </div>
    </div>

    <!--
      The security posture, stated where the operator reads it. Everything in
      this list is a control the server enforces — the global switch, the
      per-project opt-in, the executable allow-list, the absence of a shell,
      and the redaction of secrets from captured output. The list is accurate
      and it is not a promise that the command is harmless.
    -->
    <UAlert
      icon="i-lucide-shield-alert"
      color="warning"
      variant="subtle"
      title="A hook runs a command on the server"
      description="This is arbitrary code execution by design. The server keeps the layer contained: hooks stay off until AGENT_SHAKER_HOOKS_ENABLED is set, each project has to opt in, the executable has to be on the project's allow-list, and the command runs as an argument list without a shell. Secrets are redacted from the captured output — the rest of it is stored and shown on this page."
      class="mb-6"
    />

    <!--
      The per-project switch. It sits above the list because it decides
      whether anything in the list runs at all: a project with hooks and the
      switch off runs none of them.
    -->
    <UCard v-if="projectId" class="mb-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="font-medium">Run hooks for {{ currentProjectName }}</p>
          <p class="text-muted text-sm mt-1">
            The server runs this project's hooks only while this is on. Creating a hook
            turns it on; turning it off leaves the hooks in place, disabled.
          </p>
        </div>
        <USwitch
          :model-value="projectEnabled"
          :disabled="saving"
          aria-label="Run hooks for this project"
          @update:model-value="toggleProjectEnabled"
        />
      </div>
    </UCard>

    <!-- HOOKS -->
    <section class="mb-10">
      <h2 class="text-sm font-semibold uppercase tracking-wide text-muted mb-3">
        Hooks
        <span class="font-normal normal-case">({{ hookList.length }})</span>
      </h2>

      <div v-if="loading" class="space-y-3" aria-busy="true">
        <USkeleton v-for="n in 3" :key="n" class="h-32 w-full rounded-xl" />
      </div>

      <div
        v-else-if="loadError"
        role="alert"
        class="flex items-start gap-3 rounded-lg border border-error/40 bg-error/10 px-4 py-3 text-sm text-error"
      >
        <UIcon name="i-lucide-circle-alert" class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
        <div class="leading-snug">
          <p>{{ loadError }}</p>
          <UButton size="xs" variant="ghost" color="error" class="mt-2" @click="hooks.load(projectId ?? '')">
            Try again
          </UButton>
        </div>
      </div>

      <EmptyState
        v-else-if="!hookList.length"
        icon="i-lucide-webhook"
        title="No hooks in this project"
        description="A hook runs a command when a task-lifecycle event happens. Nothing is configured yet."
      >
        <UButton icon="i-lucide-plus" size="sm" color="primary" @click="openCreate">Create the first hook</UButton>
      </EmptyState>

      <div v-else class="space-y-3">
        <HookCard
          v-for="hook in hookList"
          :key="hook.id"
          :hook="hook"
          :testing="testingId === hook.id"
          @edit="openEdit"
          @remove="deletingHook = $event"
          @toggle="handleToggle"
          @test="handleTest"
        />
      </div>
    </section>

    <!-- RUN LOG -->
    <section>
      <h2 class="text-sm font-semibold uppercase tracking-wide text-muted mb-3">
        Recent runs
        <span class="font-normal normal-case">({{ runList.length }})</span>
      </h2>

      <div
        v-if="runError"
        role="alert"
        class="flex items-start gap-3 rounded-lg border border-error/40 bg-error/10 px-4 py-3 text-sm text-error"
      >
        <UIcon name="i-lucide-circle-alert" class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
        <p class="leading-snug">{{ runError }}</p>
      </div>

      <EmptyState
        v-else-if="!runList.length"
        icon="i-lucide-scroll-text"
        title="No runs yet"
        description="A run is recorded every time a hook fires. Use Test on a hook to try it against a synthetic payload, without touching real work."
      />

      <div v-else class="space-y-2">
        <HookRunRow
          v-for="run in runList"
          :key="run.id"
          :run="run"
          :expanded="expandedRunId === run.id"
          @toggle="toggleRunOutput"
        />
      </div>
    </section>

    <!-- CREATE / EDIT -->
    <HookFormModal
      v-model="formOpen"
      :initial="editingHook"
      :on-submit="saveHook"
    />

    <!-- TEST RESULT -->
    <UModal
      v-model:open="testRunOpen"
      title="Test run"
      icon="i-lucide-flask-conical"
      :ui="{ content: 'sm:max-w-3xl' }"
    >
      <template #body>
        <div v-if="testRun" class="space-y-4">
          <div class="flex flex-wrap items-center gap-2">
            <UBadge :color="HOOK_RUN_STATUS_COLORS[testRun.status]" variant="subtle" size="xs">
              {{ HOOK_RUN_STATUS_LABELS[testRun.status] }}
            </UBadge>
            <span class="text-sm font-medium">{{ hookEventLabel(testRun.event) }}</span>
            <span class="text-xs text-muted">
              exit {{ testRun.exit_code ?? '—' }} · {{ formatHookDuration(testRun.duration_ms) }}
            </span>
          </div>

          <p class="text-xs text-muted">
            A dry run against a synthetic payload. No real task, agent or blocker was touched.
          </p>

          <div v-if="testRun.stdout">
            <p class="text-xs uppercase tracking-wide text-muted">stdout</p>
            <pre class="mt-1 text-xs bg-default/50 rounded p-2 overflow-auto max-h-64 font-mono whitespace-pre-wrap">{{ testRun.stdout }}</pre>
          </div>
          <div v-if="testRun.stderr">
            <p class="text-xs uppercase tracking-wide text-muted">stderr</p>
            <pre class="mt-1 text-xs bg-default/50 rounded p-2 overflow-auto max-h-64 font-mono whitespace-pre-wrap text-error">{{ testRun.stderr }}</pre>
          </div>
          <p v-if="!testRun.stdout && !testRun.stderr" class="text-sm text-muted">
            The command produced no output.
          </p>
        </div>
      </template>
    </UModal>

    <ConfirmDialog
      v-model="deletingOpen"
      :title="`Delete the ${deletingHook ? HOOK_EVENT_LABELS[deletingHook.event] : ''} hook?`"
      description="The hook is removed. Its past runs stay in the log — the log outlives the hook."
      confirm-label="Delete hook"
      @confirm="handleDelete"
    />
  </div>
</template>
