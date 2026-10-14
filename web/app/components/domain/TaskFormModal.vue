<script setup lang="ts">
import type { Task, CreateTaskInput, TaskPriority, Milestone } from '~/types/api'
import { compose, maxLength, required } from '~/utils/formValidation'
import { NO_SELECTION, optionalId } from '~/utils/select'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
  milestones?: Milestone[]
  initial?: Task | null
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer — see the note in useFormModal.
   */
  onSubmit: (payload: CreateTaskInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const isEdit = computed(() => !!props.initial)

/**
 * Not `CreateTaskInput`: the form holds `tagsRaw` (what was typed) while the
 * API takes `tags` (a list), and the API treats an empty string as a value it
 * has to store rather than as "not set". The conversion happens in
 * `toCreateTaskInput`, so the fields below stay exactly what the inputs bind to.
 */
type TaskForm = {
  title: string
  description: string
  priority: TaskPriority
  assigned_to: string
  created_by: string
  milestone_id: string
  tagsRaw: string
}

const form = reactive<TaskForm>({
  title: '',
  description: '',
  priority: 'medium',
  assigned_to: NO_SELECTION,
  created_by: '',
  milestone_id: NO_SELECTION,
  tagsRaw: ''
})

const validate = compose<TaskForm>(
  // 255 matches tasks.title VARCHAR(255) in migrations/001_init.up.sql; the
  // limit belongs next to the constraint, not only in the database.
  required('title', 'A task title'),
  maxLength('title', 255, 'The title')
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  onSubmit: (payload) => props.onSubmit(toCreateTaskInput(payload)),
  close: () => { open.value = false },
  failureMessage: 'Could not save the task'
})

function toCreateTaskInput(values: TaskForm): CreateTaskInput {
  const tags = values.tagsRaw.split(',').map(s => s.trim()).filter(Boolean)
  return {
    project_id: props.projectId,
    title: values.title,
    description: values.description,
    priority: values.priority,
    // An unset picker is the sentinel rather than null, and the API wants the field gone.
    assigned_to: optionalId(values.assigned_to),
    created_by: values.created_by || undefined,
    milestone_id: optionalId(values.milestone_id),
    tags: tags.length ? tags : undefined
  }
}

watch(() => props.modelValue, (v) => {
  if (v) {
    form.title = props.initial?.title ?? ''
    form.description = props.initial?.description ?? ''
    form.priority = props.initial?.priority ?? 'medium'
    form.assigned_to = props.initial?.assigned_to ?? NO_SELECTION
    form.created_by = props.initial?.created_by ?? ''
    form.milestone_id = props.initial?.milestone_id ?? NO_SELECTION
    form.tagsRaw = (props.initial?.tags ?? []).join(', ')
  }
})
</script>

<template>
  <FormModal
    v-model:open="open"
    :title="isEdit ? 'Edit task' : 'Create task'"
    :description="isEdit
      ? `Changes apply to ${initial?.title}.`
      : 'One piece of work someone can pick up. Keep it small enough to finish.'"
    :icon="isEdit ? 'i-lucide-pencil' : 'i-lucide-plus'"
    :submit-label="isEdit ? 'Save changes' : 'Create task'"
    :submitting="submitting"
    :error="submitError"
    size="lg"
    @submit="submit"
  >
    <FormField
      label="Title"
      name="title"
      required
      hint="What is being done, as an outcome. Not a category, not a ticket number."
      :aside="`${form.title.length}/255`"
      :error="errors.title"
    >
      <UInput
        v-model="form.title"
        name="title"
        autofocus
        placeholder="Implement user authentication"
        class="w-full"
        maxlength="255"
      />
    </FormField>

    <FormField
      label="Description"
      name="description"
      hint="Optional. Context, acceptance criteria, links. Anything the assignee would otherwise have to ask for."
      :error="errors.description"
    >
      <UTextarea
        v-model="form.description"
        name="description"
        :rows="6"
        placeholder="What needs to be done?"
        class="w-full resize-y"
      />
    </FormField>

    <FormField
      label="Priority"
      name="priority"
      hint="Drives ordering in the backlog. Medium unless something is blocked on it."
      :error="errors.priority"
    >
      <USelect
        v-model="form.priority"
        name="priority"
        class="w-full"
        :items="[
          { label: 'Low', value: 'low' },
          { label: 'Medium', value: 'medium' },
          { label: 'High', value: 'high' }
        ]"
        value-key="value"
      />
    </FormField>

    <FormField
      label="Assigned to"
      name="assigned_to"
      hint="Leave unassigned to keep it in the backlog."
      :error="errors.assigned_to"
    >
      <USelect
        v-model="form.assigned_to"
        name="assigned_to"
        class="w-full"
        :items="[{ label: 'Unassigned', value: NO_SELECTION }, ...(agents?.map(a => ({ label: a.name, value: a.id })) ?? [])]"
        value-key="value"
      />
    </FormField>

    <FormField
      label="Milestone"
      name="milestone_id"
      hint="Optional. Groups this task into a delivery slice."
      :error="errors.milestone_id"
    >
      <USelect
        v-model="form.milestone_id"
        name="milestone_id"
        class="w-full"
        :items="[{ label: '— none —', value: NO_SELECTION }, ...(milestones?.map(m => ({ label: m.title, value: m.id })) ?? [])]"
        value-key="value"
      />
    </FormField>

    <FormField
      label="Tags"
      name="tagsRaw"
      hint="Comma-separated. Use feature:<name> to roll this task up on the Features view."
      :error="errors.tagsRaw"
    >
      <UInput
        v-model="form.tagsRaw"
        name="tagsRaw"
        placeholder="feature:auth, backend"
        class="w-full"
      />
    </FormField>
  </FormModal>
</template>