<script setup lang="ts">
/**
 * The project's blocker list, and the only place a human answers a stuck agent.
 *
 * It owns its own data rather than taking blockers from the page: the panel is
 * mounted on a project route where the blockers are only needed by this card,
 * and lifting them would mean every parent had to know about live blocker
 * events. Live updates arrive over the shared socket, so the list stays current
 * without the page re-rendering around it.
 *
 * The ordering is the whole point of this panel. It answers "what needs me
 * now", so open blockers come first, oldest first — an agent blocked since
 * Monday is the one that has been losing the most time — and closed ones sink
 * to the bottom where they document the past rather than compete for attention.
 */
import type { Blocker, BlockerStatus, DismissBlockerInput, ResolveBlockerInput } from '~/types/api'
import type { WsEventFor } from '~/types/ws'
import { compose, required } from '~/utils/formValidation'

const props = defineProps<{ projectId: string }>()

const api = useApi()
const toast = useToastBus()
const realtime = useRealtime()

const blockers = ref<Blocker[]>([])
const loading = ref(true)
const loadError = ref<string | null>(null)

/**
 * Group rank first, age second. `created_at` is the moment the agent stopped,
 * not the moment the row was written, so it is the correct clock for "how long
 * has this been costing us".
 */
const OPEN_RANK: Record<BlockerStatus, number> = {
  open: 0,
  in_review: 1,
  resolved: 2,
  dismissed: 2
}

function sortBlockers(rows: Blocker[]): Blocker[] {
  return [...rows].sort((a, b) => {
    const rank = OPEN_RANK[a.status] - OPEN_RANK[b.status]
    if (rank !== 0) return rank
    const left = new Date(a.created_at).getTime()
    const right = new Date(b.created_at).getTime()
    if (Number.isNaN(left) || Number.isNaN(right)) return 0
    // Closed blockers read best newest-first: the most recent decision is the
    // one an operator is most likely to be checking against.
    return a.status === 'open' || a.status === 'in_review' ? left - right : right - left
  })
}

async function load(): Promise<void> {
  loading.value = true
  loadError.value = null
  try {
    blockers.value = sortBlockers(await api.listBlockers(props.projectId))
  } catch (err: unknown) {
    const candidate = err as { data?: { error?: { message?: string } }; message?: string }
    loadError.value = candidate?.data?.error?.message ?? candidate?.message ?? 'Could not load blockers'
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(() => props.projectId, load)

// --- Dialogs -------------------------------------------------------------
//
// Resolve and dismiss are separate instances rather than one dialog with a
// mutable mode: each owns its own text and its own failure banner, so a server
// error from the last resolve cannot be shown above a fresh dismiss form.

type DialogMode = 'resolve' | 'dismiss'

const dialogMode = ref<DialogMode | null>(null)
const target = ref<Blocker | null>(null)

function openDialog(mode: DialogMode, blocker: Blocker): void {
  target.value = blocker
  dialogMode.value = mode
}

const dialogOpen = computed({
  get: () => dialogMode.value !== null,
  set: (v: boolean) => { if (!v) dialogMode.value = null }
})

const resolveForm = reactive<ResolveBlockerInput>({ resolution: '' })
const dismissForm = reactive<DismissBlockerInput>({ reason: '' })

/**
 * The server trims and rejects an empty answer (internal/handlers/blockers.go
 * closeBlocker → 400). Validating with the same rule here means the field
 * complains before a round trip, using the same words.
 *
 * No length cap: `blockers.resolution` is TEXT (migrations/015), so a limit
 * invented in the browser would be one the database never had.
 */
const validateResolve = compose<ResolveBlockerInput>(required('resolution', 'An answer to the agent'))
const validateDismiss = compose<DismissBlockerInput>(required('reason', 'A reason'))

/** Replaces the row in place from the server's own copy, then re-sorts. */
function applyClosed(updated: Blocker): void {
  const index = blockers.value.findIndex((b) => b.id === updated.id)
  if (index === -1) {
    blockers.value = sortBlockers([...blockers.value, updated])
    return
  }
  blockers.value.splice(index, 1, updated)
  blockers.value = sortBlockers(blockers.value)
}

watch(dialogOpen, (v) => {
  if (v) {
    resolveForm.resolution = ''
    dismissForm.reason = ''
  }
})

const resolveModal = useFormModal({
  form: resolveForm,
  validate: validateResolve,
  onSubmit: async (values) => {
    if (!target.value) throw new Error('No blocker selected')
    const updated = await api.resolveBlocker(props.projectId, target.value.id, { resolution: values.resolution })
    applyClosed(updated)
    toast.success('Blocker resolved')
  },
  close: () => { dialogMode.value = null },
  failureMessage: 'Could not resolve the blocker'
})

const dismissModal = useFormModal({
  form: dismissForm,
  validate: validateDismiss,
  onSubmit: async (values) => {
    if (!target.value) throw new Error('No blocker selected')
    const updated = await api.dismissBlocker(props.projectId, target.value.id, { reason: values.reason })
    applyClosed(updated)
    toast.success('Blocker dismissed')
  },
  close: () => { dialogMode.value = null },
  failureMessage: 'Could not dismiss the blocker'
})

// --- Live updates --------------------------------------------------------

/**
 * `blocker_raised` carries the title and the type but no `created_at` and no
 * status, and its ids are not rows this panel has seen. Building a card from
 * it would produce a blocker with an invented age; a refetch is the honest way
 * to put it on screen.
 */
const offRaised = realtime.on('blocker_raised', () => {
  void load()
})

/**
 * `blocker_resolved` is broadcast by both the resolve and the dismiss path, and
 * it does not carry the terminal status. Guessing "resolved" would file a
 * dismissal under the card's "Answer" heading, so the authoritative row is read
 * back by id instead of being reconstructed here.
 */
const offResolved = realtime.on('blocker_resolved', async (event: WsEventFor<'blocker_resolved'>) => {
  const id = event.payload.blocker_id
  if (!blockers.value.some((b) => b.id === id)) return
  try {
    const closed = await api.getBlocker(props.projectId, id)
    applyClosed(closed)
  } catch {
    // The socket said something the API cannot confirm (the panel may have been
    // pointed at another project in between). A refetch still gets the truth.
    void load()
  }
})

onUnmounted(() => {
  offRaised()
  offResolved()
})
</script>

<template>
  <section class="space-y-3">
    <!--
      The panel is a section of the page, so it carries an h2. Without it every
      blocker title jumps from the page's h1 straight to h3, which is both a
      Lighthouse failure and a real navigation problem for anyone moving by
      heading level.
    -->
    <h2 class="sr-only">Blockers</h2>
    <!--
      Skeletons rather than a spinner: the list has a known shape, so showing
      three empty cards keeps the page from jumping when the real rows land.
    -->
    <div v-if="loading" class="space-y-3" aria-busy="true">
      <div v-for="n in 3" :key="n" class="rounded-xl border border-default p-5 space-y-3">
        <USkeleton class="h-4 w-2/5" />
        <USkeleton class="h-3 w-4/5" />
        <USkeleton class="h-3 w-1/4" />
      </div>
    </div>

    <!--
      Same role="alert" treatment as FormModal's request-error banner: the list
      failed to load, which is a whole-panel failure, not a field's problem, and
      it has to be announced rather than only drawn.
    -->
    <div
      v-else-if="loadError"
      role="alert"
      class="flex items-start gap-3 rounded-lg border border-error/40 bg-error/10 px-4 py-3 text-sm text-error"
    >
      <UIcon name="i-lucide-circle-alert" class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <div class="leading-snug">
        <p>{{ loadError }}</p>
        <UButton size="xs" variant="ghost" color="error" class="mt-2" @click="load">
          Try again
        </UButton>
      </div>
    </div>

    <EmptyState
      v-else-if="blockers.length === 0"
      icon="i-lucide-circle-check-big"
      title="No blockers"
      description="A blocker is an agent that has stopped and is waiting on a decision from you. Nothing is waiting."
    />

    <div v-else class="space-y-3">
      <BlockerCard
        v-for="blocker in blockers"
        :key="blocker.id"
        :blocker="blocker"
        @resolve="openDialog('resolve', $event)"
        @dismiss="openDialog('dismiss', $event)"
      />
    </div>

    <FormModal
      v-if="dialogMode === 'resolve'"
      v-model:open="dialogOpen"
      title="Answer the agent"
      description="The agent picks this up on its next heartbeat and acts on it."
      icon="i-lucide-message-square-reply"
      submit-label="Resolve blocker"
      :submitting="resolveModal.submitting.value"
      :error="resolveModal.submitError.value"
      @submit="resolveModal.submit"
    >
      <FormField
        label="Answer to the agent"
        name="resolution"
        required
        hint="The agent acts on this text verbatim — it is the whole payload it gets back. Say what to do, not what you decided."
        :error="resolveModal.errors.value.resolution"
      >
        <UTextarea
          v-model="resolveForm.resolution"
          name="resolution"
          :rows="5"
          autofocus
          placeholder="Use the existing schema; the field is non-null, so an empty string is a server error, not a skip."
          class="w-full resize-y"
        />
      </FormField>
    </FormModal>

    <FormModal
      v-if="dialogMode === 'dismiss'"
      v-model:open="dialogOpen"
      title="Dismiss the blocker"
      description="Use this when the blocker is not worth answering — not to close one you have not looked at."
      icon="i-lucide-circle-slash"
      submit-label="Dismiss blocker"
      :submitting="dismissModal.submitting.value"
      :error="dismissModal.submitError.value"
      @submit="dismissModal.submit"
    >
      <FormField
        label="Reason"
        name="reason"
        required
        hint="This is what tells the agent to stop asking. An agent that is dismissed without a reason raises the same blocker again on the next run."
        :error="dismissModal.errors.value.reason"
      >
        <UTextarea
          v-model="dismissForm.reason"
          name="reason"
          :rows="4"
          autofocus
          placeholder="Not needed — the deploy pipeline handles this case."
          class="w-full resize-y"
        />
      </FormField>
    </FormModal>
  </section>
</template>