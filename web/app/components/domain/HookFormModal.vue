<script setup lang="ts">
import type { CreateHookInput, Hook, HookEvent, HookPost, HookStdin, HookStdinMode } from '~/types/hooks'
import {
  HOOK_DEFAULT_TIMEOUT_SECONDS,
  HOOK_MAX_TIMEOUT_SECONDS,
  HOOK_PLACEHOLDERS,
  hookEventItems
} from '~/types/hooks'
import { compose, requiredChoice, type FormErrors } from '~/utils/formValidation'

/**
 * The hook form.
 *
 * Two things are load-bearing here and are why this is not a generic CRUD
 * form:
 *
 *   - The command is an argument list, never a shell command line. The server
 *     executes the vector it stored, with no shell in between, so there is
 *     nothing to quote and no way to smuggle a pipe. The form therefore edits
 *     one argument at a time and the helper text says so plainly.
 *   - A hook is arbitrary code execution. The dialog says what the server does
 *     to contain it, where the operator decides to create one — not only in a
 *     footnote elsewhere.
 */
const props = defineProps<{
  modelValue: boolean
  /** The hook being edited. Null for a new one. */
  initial?: Hook | null
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer — see the note in useFormModal.
   */
  onSubmit: (payload: CreateHookInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const isEdit = computed(() => !!props.initial)

type HookForm = {
  event: HookEvent
  matcherEnabled: boolean
  matcherField: string
  matcherPattern: string
  /** One entry per argument. The first one is the executable. */
  commandArgs: string[]
  /**
   * The hook's environment, carried through untouched.
   *
   * The form does not edit it, but the server's update path replaces the map
   * with whatever the body carries — so a hook with a `{secret:NAME}`
   * reference would lose it the first time somebody edited its command. It is
   * round-tripped verbatim instead.
   */
  env: Record<string, string>
  stdinMode: HookStdinMode
  stdinTemplate: string
  postEnabled: boolean
  postTitle: string
  /** A string, because the input is a number field the operator may empty. */
  timeoutSeconds: string
  enabled: boolean
}

const eventItems = hookEventItems()

const stdinItems: { label: string; value: HookStdinMode }[] = [
  { label: 'Event JSON — the whole event', value: 'event' },
  { label: 'Prompt — a rendered template', value: 'prompt' }
]

const form = reactive<HookForm>({
  event: 'task.completed',
  matcherEnabled: false,
  matcherField: '',
  matcherPattern: '',
  commandArgs: [],
  env: {},
  stdinMode: 'event',
  stdinTemplate: '',
  postEnabled: false,
  postTitle: '',
  timeoutSeconds: String(HOOK_DEFAULT_TIMEOUT_SECONDS),
  enabled: true
})

const validate = compose<HookForm>(
  requiredChoice('event', 'an event'),
  (values) => {
    const errors: FormErrors<HookForm> = {}
    if (!values.commandArgs.some((a) => a.trim() !== '')) {
      errors.commandArgs = 'At least one argument is required — the first one is the executable'
    }
    if (values.matcherEnabled && !values.matcherField.trim()) {
      errors.matcherField = 'Name the field the pattern is matched against'
    }
    if (values.stdinMode === 'prompt' && !values.stdinTemplate.trim()) {
      errors.stdinTemplate = 'A prompt template is required in prompt mode'
    }
    if (values.postEnabled && !values.postTitle.trim()) {
      errors.postTitle = 'A title is required for the context'
    }
    const timeout = Number(values.timeoutSeconds)
    if (!Number.isInteger(timeout) || timeout < 1 || timeout > HOOK_MAX_TIMEOUT_SECONDS) {
      errors.timeoutSeconds = `The timeout must be a whole number of seconds between 1 and ${HOOK_MAX_TIMEOUT_SECONDS}`
    }
    return errors
  }
)

/**
 * Builds the request body from the form.
 *
 * Empty arguments are dropped — an empty box is an unfilled box, not an
 * argument — but the arguments themselves are passed through verbatim. The
 * stored vector is the executed vector, so nothing here may rewrite it.
 */
function toHookInput(values: HookForm): CreateHookInput {
  const command = values.commandArgs.filter((a) => a.trim() !== '')
  const matcher =
    values.matcherEnabled && values.matcherField.trim()
      ? { field: values.matcherField.trim(), pattern: values.matcherPattern }
      : null
  const stdin: HookStdin =
    values.stdinMode === 'prompt'
      ? { mode: 'prompt', template: values.stdinTemplate }
      : { mode: 'event' }
  const post: HookPost | null =
    values.postEnabled && values.postTitle.trim()
      ? { kind: 'context', title: values.postTitle }
      : null
  const payload: CreateHookInput = {
    event: values.event,
    matcher,
    command,
    stdin,
    post,
    timeout_seconds: Number(values.timeoutSeconds),
    enabled: values.enabled
  }
  // Only sent when there is something to send. An empty map would be read as
  // "this hook has no environment" by the server's update path, which replaces
  // the stored map outright rather than merging into it.
  if (Object.keys(values.env).length > 0) payload.env = { ...values.env }
  return payload
}

/**
 * FormModal emits `submit` twice for one click: its footer button carries both
 * a click handler and `type="submit"`, so the click and the form's own submit
 * event both reach the handler. One in-flight request is kept and the second
 * call waits for it, so a hook cannot be created twice.
 */
let pendingSubmit: Promise<unknown> | null = null

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  onSubmit: (values) => {
    if (pendingSubmit) return pendingSubmit
    pendingSubmit = props.onSubmit(toHookInput(values)).finally(() => {
      pendingSubmit = null
    })
    return pendingSubmit
  },
  close: () => { open.value = false },
  failureMessage: 'Could not save the hook'
})

/** Loads the form with either a fresh hook or the one being edited. */
watch(open, (isOpen) => {
  if (!isOpen) return
  const hook = props.initial
  form.event = hook?.event ?? 'task.completed'
  form.matcherEnabled = !!hook?.matcher
  form.matcherField = hook?.matcher?.field ?? ''
  form.matcherPattern = hook?.matcher?.pattern ?? ''
  form.commandArgs = hook ? [...hook.command] : ['']
  form.env = { ...(hook?.env ?? {}) }
  form.stdinMode = hook?.stdin?.mode ?? 'event'
  form.stdinTemplate = hook?.stdin?.template ?? ''
  form.postEnabled = !!hook?.post
  form.postTitle = hook?.post?.title ?? ''
  form.timeoutSeconds = String(hook?.timeout_seconds ?? HOOK_DEFAULT_TIMEOUT_SECONDS)
  form.enabled = hook?.enabled ?? true
})
</script>

<template>
  <FormModal
    v-model:open="open"
    :title="isEdit ? 'Edit hook' : 'New hook'"
    description="A hook is a command the server runs when a task-lifecycle event happens."
    icon="i-lucide-webhook"
    :submit-label="isEdit ? 'Save changes' : 'Create hook'"
    :submitting="submitting"
    :error="submitError"
    size="lg"
    @submit="submit"
  >
    <UAlert
      icon="i-lucide-shield-alert"
      color="warning"
      variant="subtle"
      title="This hook runs a command on the server"
      description="Hooks stay disabled on the server until AGENT_SHAKER_HOOKS_ENABLED is set and the project has opted in, the executable must be on the project's allow-list, and the command is executed without a shell. Secrets are redacted from the captured output."
    />

    <FormField
      label="Event"
      name="event"
      required
      :hint="isEdit
        ? 'The event a hook reacts to cannot be changed after it is created. Delete the hook and create another one for a different event.'
        : 'The event that fires this hook. A hook bound to an event the server never publishes is rejected when it is created.'"
      :error="errors.event"
    >
      <USelect
        v-model="form.event"
        name="event"
        class="w-full"
        :items="eventItems"
        value-key="value"
        :disabled="isEdit"
      />
    </FormField>

    <!--
      The command editor. Each argument gets its own box, because the
      argument boundary is the whole point: the server passes the vector to
      the process unchanged, so a box is one argument and nothing in the UI
      pretends the text is a shell command line.
    -->
    <div class="w-full space-y-3 rounded-lg border border-default p-4">
      <div>
        <p class="text-sm font-medium text-default">Command arguments</p>
        <p class="text-xs text-muted mt-1">
          One argument per box. This is an argument list, not a shell command line: the server runs
          the executable directly, so there is no quoting, no pipes and no redirections. The first
          argument is the executable, and it has to be on the project's allow-list.
        </p>
      </div>

      <div class="space-y-2">
        <div
          v-for="(argument, index) in form.commandArgs"
          :key="index"
          class="flex items-center gap-2"
        >
          <span class="text-xs text-muted w-5 shrink-0 tabular-nums">{{ index + 1 }}</span>
          <UInput
            v-model="form.commandArgs[index]"
            :name="`command-${index}`"
            class="w-full font-mono"
            :placeholder="index === 0 ? 'mcode' : 'exec'"
            :aria-label="`Command argument ${index + 1}`"
          />
          <UButton
            icon="i-lucide-x"
            variant="ghost"
            color="neutral"
            size="sm"
            :aria-label="`Remove argument ${index + 1}`"
            @click="form.commandArgs.splice(index, 1)"
          />
        </div>
      </div>

      <p v-if="errors.commandArgs" class="text-sm text-error">
        {{ errors.commandArgs }}
      </p>

      <UButton
        icon="i-lucide-plus"
        variant="outline"
        color="neutral"
        size="xs"
        @click="form.commandArgs.push('')"
      >
        Add argument
      </UButton>
    </div>

    <FormField
      label="Matcher"
      name="matcherEnabled"
      hint="Optional. A regular expression over one field of the entity. Without it the hook fires on every event of this type."
    >
      <div class="w-full space-y-3">
        <USwitch v-model="form.matcherEnabled" label="Only fire on matching entities" size="sm" />
        <div v-if="form.matcherEnabled" class="grid gap-3 sm:grid-cols-2">
          <FormField
            label="Field"
            name="matcherField"
            required
            hint="The entity field the pattern is matched against."
            :error="errors.matcherField"
          >
            <UInput
              v-model="form.matcherField"
              name="matcherField"
              placeholder="title"
              class="w-full font-mono"
            />
          </FormField>
          <FormField
            label="Pattern"
            name="matcherPattern"
            hint="A regular expression. A malformed one is rejected now, not at fire time."
          >
            <UInput
              v-model="form.matcherPattern"
              name="matcherPattern"
              placeholder="^\[API\]"
              class="w-full font-mono"
            />
          </FormField>
        </div>
      </div>
    </FormField>

    <FormField
      label="stdin"
      name="stdinMode"
      hint="What the command reads on its standard input."
    >
      <USelect
        v-model="form.stdinMode"
        name="stdinMode"
        class="w-full"
        :items="stdinItems"
        value-key="value"
      />
    </FormField>

    <div v-if="form.stdinMode === 'prompt'" class="w-full space-y-3 rounded-lg border border-default p-4">
      <FormField
        label="Prompt template"
        name="stdinTemplate"
        required
        hint="Rendered before the command runs. The same placeholders are substituted in every command argument."
        :error="errors.stdinTemplate"
      >
        <UTextarea
          v-model="form.stdinTemplate"
          name="stdinTemplate"
          :rows="5"
          class="w-full resize-y font-mono"
          placeholder="Review the completed task {{entity.title}} in {{project.repo_path}} and report defects."
        />
      </FormField>

      <div>
        <p class="text-xs uppercase tracking-wide text-muted">Available placeholders</p>
        <ul class="mt-1 space-y-0.5">
          <li v-for="placeholder in HOOK_PLACEHOLDERS" :key="placeholder.token" class="text-xs flex gap-2">
            <code class="font-mono shrink-0">{{ placeholder.token }}</code>
            <span class="text-muted">{{ placeholder.description }}</span>
          </li>
        </ul>
      </div>
    </div>

    <div class="w-full space-y-3 rounded-lg border border-default p-4">
      <USwitch
        v-model="form.postEnabled"
        label="Write the command's output back as a context"
        size="sm"
      />
      <FormField
        v-if="form.postEnabled"
        label="Context title"
        name="postTitle"
        required
        hint="The title of the context the output is written to. Placeholders work here too."
        :error="errors.postTitle"
      >
        <UInput
          v-model="form.postTitle"
          name="postTitle"
          class="w-full"
          placeholder="Review of {{entity.title}}"
        />
      </FormField>
    </div>

    <div class="grid gap-5 sm:grid-cols-2">
      <FormField
        label="Timeout (seconds)"
        name="timeoutSeconds"
        required
        :hint="`Between 1 and ${HOOK_MAX_TIMEOUT_SECONDS}. The process group is killed when it passes.`"
        :error="errors.timeoutSeconds"
      >
        <UInput
          v-model="form.timeoutSeconds"
          name="timeoutSeconds"
          type="number"
          min="1"
          :max="HOOK_MAX_TIMEOUT_SECONDS"
          class="w-full"
        />
      </FormField>
      <div class="flex items-end pb-2">
        <USwitch v-model="form.enabled" label="Enabled" size="sm" />
      </div>
    </div>
  </FormModal>
</template>
