<script setup lang="ts">
import type { CreateGlobalContextInput, GlobalContextScope } from '~/types/api'
import { compose, maxLength, required, requiredChoice } from '~/utils/formValidation'

const props = defineProps<{
  modelValue: boolean
  agentId: string
  projectId?: string
  initialScope?: GlobalContextScope
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer — see the note in useFormModal.
   */
  onSubmit: (payload: CreateGlobalContextInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

type GlobalContextForm = {
  scope: GlobalContextScope
  title: string
  content: string
  tagsRaw: string
}

const form = reactive<GlobalContextForm>({
  scope: props.initialScope ?? 'project',
  title: '',
  content: '',
  tagsRaw: ''
})

const validate = compose<GlobalContextForm>(
  // 255 is global_contexts.title VARCHAR(255) in
  // migrations/007_global_contexts.up.sql; content is TEXT with no limit.
  requiredChoice('scope', 'a scope'),
  required('title', 'A title'),
  maxLength('title', 255, 'The title')
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  onSubmit: (payload) => {
    // The database enforces scope='project' ⇒ project_id is not null, so a
    // missing project id is reported as a failed submit rather than throwing
    // inside the dialog where nothing would show it.
    if (payload.scope === 'project' && !props.projectId) {
      throw new Error('This document is project-scoped but no project was supplied.')
    }
    const input: CreateGlobalContextInput = {
      scope: payload.scope,
      agent_id: props.agentId,
      title: payload.title,
      content: payload.content,
      tags: payload.tagsRaw.split(',').map(s => s.trim()).filter(Boolean)
    }
    if (payload.scope === 'project') input.project_id = props.projectId
    return props.onSubmit(input)
  },
  close: () => { open.value = false },
  failureMessage: 'Could not publish the document'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.scope = props.initialScope ?? 'project'
    form.title = ''
    form.content = ''
    form.tagsRaw = ''
  }
})
</script>

<template>
  <FormModal
    v-model:open="open"
    title="Publish doc"
    description="A playbook or reference the agents in scope can read."
    icon="i-lucide-globe"
    submit-label="Publish document"
    :submitting="submitting"
    :error="submitError"
    size="lg"
    @submit="submit"
  >
    <FormField
      label="Scope"
      name="scope"
      required
      autofocus
      hint="Project keeps the doc inside this project; global publishes it server-wide."
      :error="errors.scope"
    >
      <USelect
        v-model="form.scope"
        name="scope"
        class="w-full"
        :items="[
          { label: 'Project — visible only inside this project', value: 'project' },
          { label: 'Global — server-wide playbook',                value: 'global' }
        ]"
        value-key="value"
      />
    </FormField>

    <FormField
      label="Title"
      name="title"
      required
      hint="Also the resource name agents read it by, so keep it stable."
      :aside="`${form.title.length}/255`"
      :error="errors.title"
    >
      <UInput
        v-model="form.title"
        name="title"
        placeholder="On-call playbook"
        class="w-full"
        maxlength="255"
      />
    </FormField>

    <FormField
      label="Content (markdown)"
      name="content"
      hint="Markdown is supported — the app renders it with MarkdownView."
      :error="errors.content"
    >
      <UTextarea
        v-model="form.content"
        name="content"
        :rows="14"
        placeholder="# Heading"
        class="w-full resize-y font-mono text-sm"
      />
    </FormField>

    <FormField
      label="Tags (comma-separated)"
      name="tagsRaw"
      hint="Optional. Used to find the document again."
      :error="errors.tagsRaw"
    >
      <UInput
        v-model="form.tagsRaw"
        name="tagsRaw"
        placeholder="oncall, infra, playbook"
        class="w-full"
      />
    </FormField>

    <p v-if="form.scope === 'global'" class="text-xs text-muted">
      After publishing, any agent on this server can read this playbook via
      <code class="font-mono">read_resource("global://{{ form.title || '<title>' }}")</code>.
    </p>
  </FormModal>
</template>