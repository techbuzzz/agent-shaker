<script setup lang="ts">
import type { CreateStandupInput } from '~/types/api'
import { compose, required, requiredChoice } from '~/utils/formValidation'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer — see the note in useFormModal.
   */
  onSubmit: (payload: CreateStandupInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

function todayIso() {
  return new Date().toISOString().slice(0, 10)
}

const form = reactive<CreateStandupInput>({
  project_id: props.projectId,
  agent_id: '',
  date: todayIso(),
  yesterday: '',
  today: '',
  blockers: ''
})

const validate = compose<CreateStandupInput>(
  // No maxLength here on purpose: daily_standups stores yesterday/today/
  // blockers as TEXT (migrations/003_daily_standups.up.sql), so a length cap
  // here would be a limit the database never had.
  required('date', 'A date'),
  // A standup belongs to one agent, and an unchosen picker reads as empty text
  // rather than as a choice — hence requiredChoice instead of required.
  requiredChoice('agent_id', 'An agent')
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  onSubmit: (payload) => props.onSubmit({ ...payload, project_id: props.projectId }),
  close: () => { open.value = false },
  failureMessage: 'Could not submit the standup'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.date = todayIso()
    form.agent_id = ''
    form.yesterday = ''
    form.today = ''
    form.blockers = ''
  }
})
</script>

<template>
  <FormModal
    v-model:open="open"
    title="Submit standup"
    description="One entry per agent per day. Yesterday's answer is what the room reads first."
    icon="i-lucide-calendar-plus"
    submit-label="Submit standup"
    :submitting="submitting"
    :error="submitError"
    @submit="submit"
  >
    <FormField
      label="Date"
      name="date"
      required
      hint="Defaults to today. Backdate only to fill in a day you missed."
      :error="errors.date"
    >
      <UInput
        v-model="form.date"
        name="date"
        type="date"
        autofocus
        class="w-full"
      />
    </FormField>

    <FormField
      label="Agent"
      name="agent_id"
      required
      hint="Who is reporting. Each agent gets one standup per date."
      :error="errors.agent_id"
    >
      <USelect
        v-model="form.agent_id"
        name="agent_id"
        class="w-full"
        :items="agents?.map(a => ({ label: a.name, value: a.id })) ?? []"
        value-key="value"
        placeholder="Select an agent"
      />
    </FormField>

    <FormField
      label="Yesterday"
      name="yesterday"
      hint="Optional. What actually landed, not what was planned."
      :error="errors.yesterday"
    >
      <UTextarea
        v-model="form.yesterday"
        name="yesterday"
        :rows="3"
        placeholder="What did you ship yesterday?"
        class="w-full resize-y"
      />
    </FormField>

    <FormField
      label="Today"
      name="today"
      hint="Optional. The one thing you are picking up next."
      :error="errors.today"
    >
      <UTextarea
        v-model="form.today"
        name="today"
        :rows="3"
        placeholder="What's the plan for today?"
        class="w-full resize-y"
      />
    </FormField>

    <FormField
      label="Blockers"
      name="blockers"
      hint="Optional. Say what you need and from whom — silence reads as 'nothing in the way'."
      :error="errors.blockers"
    >
      <UTextarea
        v-model="form.blockers"
        name="blockers"
        :rows="2"
        placeholder="Anything in the way?"
        class="w-full resize-y"
      />
    </FormField>
  </FormModal>
</template>