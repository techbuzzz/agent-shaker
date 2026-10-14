<script setup lang="ts">
import type { CreateMilestoneInput, MilestoneStatus } from '~/types/api'
import { compose, maxLength, required } from '~/utils/formValidation'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  createdBy: string
  initial?: { title: string; description?: string; status?: MilestoneStatus; target_date?: string } | null
  onSubmit: (payload: CreateMilestoneInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const form = reactive<CreateMilestoneInput>({
  project_id: props.projectId,
  title: '',
  description: '',
  status: 'planned',
  target_date: '',
  created_by: props.createdBy
})

const validate = compose<CreateMilestoneInput>(
  required('title', 'A milestone title'),
  // milestones.title VARCHAR(255) — migrations/005_milestones.up.sql.
  maxLength('title', 255, 'The title')
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  onSubmit: (values) => {
    // Optional fields are omitted rather than sent empty: the create/update
    // handlers treat "" as "set this to empty" on some paths.
    const payload: CreateMilestoneInput = {
      project_id: values.project_id,
      title: values.title,
      created_by: values.created_by
    }
    if (values.description) payload.description = values.description
    if (values.status) payload.status = values.status
    if (values.target_date) payload.target_date = values.target_date
    return props.onSubmit(payload)
  },
  close: () => { open.value = false },
  failureMessage: 'Could not save the milestone'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.title = props.initial?.title ?? ''
    form.description = props.initial?.description ?? ''
    form.status = props.initial?.status ?? 'planned'
    form.target_date = props.initial?.target_date ?? ''
    form.created_by = props.createdBy
  }
})
</script>

<template>
  <FormModal
    v-model:open="open"
    :title="initial ? 'Edit milestone' : 'Create milestone'"
    description="A milestone groups tasks under a delivery the project can be judged against."
    icon="i-lucide-flag"
    :submit-label="initial ? 'Save changes' : 'Create milestone'"
    :submitting="submitting"
    :error="submitError"
    @submit="submit"
  >
    <FormField
      label="Title"
      name="title"
      required
      :hint="'Name the delivery, not the work: “Auth MVP”, not “Fix login bug”.'"
      :aside="`${form.title.length}/255`"
      :error="errors.title"
    >
      <UInput
        v-model="form.title"
        name="title"
        autofocus
        maxlength="255"
        placeholder="M1 — Auth MVP"
        class="w-full"
      />
    </FormField>

    <FormField
      label="Description"
      name="description"
      :hint="'What has to be true for this milestone to be done.'"
      :error="errors.description"
    >
      <UTextarea
        v-model="form.description"
        name="description"
        :rows="3"
        placeholder="What does this milestone deliver?"
        class="w-full resize-y"
      />
    </FormField>

    <div class="grid gap-5 sm:grid-cols-2">
      <FormField
        label="Status"
        name="status"
        :hint="'Dropped keeps the history but stops counting towards progress.'"
        :error="errors.status"
      >
        <USelect
          v-model="form.status"
          name="status"
          class="w-full"
          :items="[
            { label: 'Planned', value: 'planned' },
            { label: 'Active',  value: 'active' },
            { label: 'Done',    value: 'done' },
            { label: 'Dropped', value: 'dropped' }
          ]"
          value-key="value"
        />
      </FormField>

      <FormField
        label="Target date"
        name="target_date"
        :hint="'Optional. Used for the overdue marker on the milestone card.'"
        :error="errors.target_date"
      >
        <UInput v-model="form.target_date" name="target_date" type="date" class="w-full" />
      </FormField>
    </div>
  </FormModal>
</template>