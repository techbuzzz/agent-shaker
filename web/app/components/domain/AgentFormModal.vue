<script setup lang="ts">
import type { Agent, CreateAgentInput, AgentStatus, AgentToolType } from '~/types/api'
import { AGENT_STATUSES, AGENT_STATUS_LABELS, AGENT_TOOL_TYPES, AGENT_TOOL_TYPE_LABELS, statusItems } from '~/types/api'
import { compose, maxLength, required } from '~/utils/formValidation'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  initial?: Agent | null
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer — see the note in useFormModal.
   */
  onSubmit: (payload: CreateAgentInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const isEdit = computed(() => !!props.initial)

const form = reactive<CreateAgentInput>({
  project_id: props.projectId,
  name: '',
  role: 'backend',
  team: '',
  status: 'active' as AgentStatus,
  tool_type: 'unknown' as AgentToolType
})

const validate = compose<CreateAgentInput>(
  // 255 matches agents.name VARCHAR(255) and agents.team VARCHAR(255) in
  // migrations/001_init.up.sql; the limits belong next to the constraints,
  // not only in the database.
  required('name', 'An agent name'),
  maxLength('name', 255, 'The name'),
  maxLength('team', 255, 'The team')
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  onSubmit: props.onSubmit,
  close: () => { open.value = false },
  failureMessage: 'Could not register the agent'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.name = props.initial?.name ?? ''
    form.role = props.initial?.role ?? 'backend'
    form.team = props.initial?.team ?? ''
    form.status = props.initial?.status ?? 'active'
    // An agent registered before this field existed reads back as 'unknown',
    // which is the honest value rather than a default that would claim a tool
    // the agent may not be running.
    form.tool_type = props.initial?.tool_type ?? 'unknown'
  }
})
</script>

<template>
  <FormModal
    v-model:open="open"
    :title="isEdit ? 'Edit agent' : 'Register agent'"
    :description="isEdit
      ? `Changes apply to ${initial?.name}.`
      : 'An agent is whoever does the work here: a service, a person, or a bot.'"
    icon="i-lucide-bot"
    :submit-label="isEdit ? 'Save changes' : 'Register agent'"
    :submitting="submitting"
    :error="submitError"
    size="sm"
    @submit="submit"
  >
    <FormField
      label="Name"
      name="name"
      required
      hint="How the agent is listed in pickers and standups."
      :aside="`${form.name.length}/255`"
      :error="errors.name"
    >
      <UInput
        v-model="form.name"
        name="name"
        autofocus
        placeholder="Backend-Copilot"
        class="w-full"
        maxlength="255"
      />
    </FormField>

    <FormField
      label="Role"
      name="role"
      required
      hint="Decides what this agent is asked for; defaults to backend."
      :error="errors.role"
    >
      <USelect
        v-model="form.role"
        name="role"
        class="w-full"
        :items="[
          { label: 'Project Manager', value: 'pm' },
          { label: 'Backend',         value: 'backend' },
          { label: 'Frontend',        value: 'frontend' },
          { label: 'DevOps',          value: 'devops' },
          { label: 'Design',          value: 'design' },
          { label: 'QA',              value: 'qa' }
        ]"
        value-key="value"
      />
    </FormField>

    <FormField
      label="Tool"
      name="tool_type"
      hint="Which client this agent runs in. Decides the configuration file we generate for it."
      :error="errors.tool_type"
    >
      <USelect
        v-model="form.tool_type"
        name="tool_type"
        class="w-full"
        :items="statusItems(AGENT_TOOL_TYPES, AGENT_TOOL_TYPE_LABELS)"
        value-key="value"
      />
    </FormField>

    <FormField
      label="Team"
      name="team"
      hint="Optional. Groups agents in the roster."
      :aside="`${form.team.length}/255`"
      :error="errors.team"
    >
      <UInput
        v-model="form.team"
        name="team"
        placeholder="Backend Team"
        class="w-full"
        maxlength="255"
      />
    </FormField>

    <FormField
      label="Initial status"
      name="status"
      hint="Paused agents keep their history but stop being assigned work."
      :error="errors.status"
    >
      <USelect
        v-model="form.status"
        name="status"
        class="w-full"
        :items="statusItems(AGENT_STATUSES, AGENT_STATUS_LABELS)"
        value-key="value"
      />
    </FormField>
  </FormModal>
</template>