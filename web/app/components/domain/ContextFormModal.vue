<script setup lang="ts">
import type { Context, CreateContextInput } from '~/types/api'
import { compose, maxLength, required, requiredChoice } from '~/utils/formValidation'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
  initial?: Context | null
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer — see the note in useFormModal.
   */
  onSubmit: (payload: CreateContextInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

// A local alias rather than CreateContextInput directly: the composable needs a
// concrete object type to validate, and tags are edited as a comma-separated
// string until they are split on the way out.
type ContextForm = {
  project_id: string
  agent_id: string
  task_id: string
  title: string
  content: string
  tags: string[]
}

const form = reactive<ContextForm>({
  project_id: props.projectId,
  agent_id: '',
  task_id: '',
  title: '',
  content: '',
  tags: []
})

const tagsInput = ref('')

const validate = compose<ContextForm>(
  // 255 is contexts.title VARCHAR(255) in migrations/001_init.up.sql. The
  // content column is TEXT with no limit, so none is invented for the body.
  required('title', 'A title'),
  maxLength('title', 255, 'The title'),
  requiredChoice('agent_id', 'the authoring agent'),
  required('content', 'Content')
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  // Tags are split here rather than on every keystroke, so a half-typed comma
  // never becomes part of a saved tag.
  onSubmit: (payload) => {
    commitTags()
    const input: CreateContextInput = {
      project_id: payload.project_id,
      agent_id: payload.agent_id,
      title: payload.title,
      content: payload.content,
      tags: [...form.tags]
    }
    if (payload.task_id) input.task_id = payload.task_id
    return props.onSubmit(input)
  },
  close: () => { open.value = false },
  failureMessage: 'Could not save the context'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.agent_id = props.initial?.agent_id ?? ''
    form.task_id = props.initial?.task_id ?? ''
    form.title = props.initial?.title ?? ''
    form.content = props.initial?.content ?? ''
    form.tags = props.initial?.tags ?? []
    tagsInput.value = (form.tags ?? []).join(', ')
  }
})

function commitTags() {
  form.tags = tagsInput.value
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}
</script>

<template>
  <FormModal
    v-model:open="open"
    :title="initial ? 'Edit context' : 'Share context'"
    :description="initial
      ? `Changes apply to ${initial?.title}.`
      : 'A note the agents in this project can read.'"
    :icon="initial ? 'i-lucide-pencil' : 'i-lucide-plus'"
    :submit-label="initial ? 'Save changes' : 'Publish context'"
    :submitting="submitting"
    :error="submitError"
    size="lg"
    @submit="submit"
  >
    <FormField
      label="Title"
      name="title"
      required
      autofocus
      hint="A short name for what this context contains."
      :aside="`${form.title.length}/255`"
      :error="errors.title"
    >
      <UInput
        v-model="form.title"
        name="title"
        placeholder="API Documentation"
        class="w-full"
        maxlength="255"
      />
    </FormField>

    <FormField
      label="Authoring agent"
      name="agent_id"
      required
      hint="Who wrote this, so a reader knows which agent to trust."
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
      label="Content (markdown)"
      name="content"
      required
      hint="Markdown is supported — the app renders it with MarkdownView."
      :error="errors.content"
    >
      <UTextarea
        v-model="form.content"
        name="content"
        :rows="14"
        placeholder="# Heading&#10;&#10;Write in **markdown**…"
        class="w-full resize-y font-mono text-sm"
      />
    </FormField>

    <FormField
      label="Tags (comma separated)"
      name="tags"
      hint="Optional. Used to filter contexts; comma separated."
      :error="errors.tags"
    >
      <UInput
        v-model="tagsInput"
        name="tags"
        placeholder="api, documentation, invoices"
        class="w-full"
        @blur="commitTags"
      />
    </FormField>
  </FormModal>
</template>