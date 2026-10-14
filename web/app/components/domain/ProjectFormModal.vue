<script setup lang="ts">
import type { Project, CreateProjectInput } from '~/types/api'
import { compose, maxLength, required } from '~/utils/formValidation'

const props = defineProps<{
  modelValue: boolean
  initial?: Project | null
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer â€” see the note in useFormModal.
   */
  onSubmit: (payload: CreateProjectInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const isEdit = computed(() => !!props.initial)

const form = reactive<CreateProjectInput>({
  name: '',
  description: '',
  status: 'active'
})

const validate = compose<CreateProjectInput>(
  // 255 matches projects.name VARCHAR(255) in migrations/001_init.up.sql; the
  // limit belongs next to the constraint, not only in the database.
  required('name', 'A project name'),
  maxLength('name', 255, 'The name')
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  onSubmit: props.onSubmit,
  close: () => { open.value = false },
  failureMessage: 'Could not save the project'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.name = props.initial?.name ?? ''
    form.description = props.initial?.description ?? ''
    form.status = props.initial?.status ?? 'active'
  }
})
</script>

<template>
  <FormModal
    v-model:open="open"
    :title="isEdit ? 'Edit project' : 'Create project'"
    :description="isEdit
      ? `Changes apply to ${initial?.name}.`
      : 'A project groups tasks, agents and standups around one piece of work.'"
    :icon="isEdit ? 'i-lucide-pencil' : 'i-lucide-folder-plus'"
    :submit-label="isEdit ? 'Save changes' : 'Create project'"
    :submitting="submitting"
    :error="submitError"
    size="sm"
    @submit="submit"
  >
    <FormField
      label="Name"
      name="name"
      required
      :hint="'How the team refers to this project.'"
      :aside="`${form.name.length}/255`"
      :error="errors.name"
    >
      <UInput
        v-model="form.name"
        name="name"
        autofocus
        placeholder="InvoiceAI"
        class="w-full"
        maxlength="255"
      />
    </FormField>

    <FormField
      label="Description"
      name="description"
      :hint="'Optional. What is being built, and for whom.'"
      :error="errors.description"
    >
      <UTextarea
        v-model="form.description"
        name="description"
        :rows="4"
        placeholder="What is this project about?"
        class="w-full resize-y"
      />
    </FormField>

    <FormField
      label="Status"
      name="status"
      :hint="'Archiving keeps history but drops it out of active views.'"
      :error="errors.status"
    >
      <USelect
        v-model="form.status"
        name="status"
        class="w-full"
        :items="[
          { label: 'Active', value: 'active' },
          { label: 'Archived', value: 'archived' }
        ]"
        value-key="value"
      />
    </FormField>
  </FormModal>
</template>
