<script setup lang="ts">
import type { Context, CreateContextInput } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
  initial?: Context | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateContextInput): Promise<void> | void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const form = reactive<CreateContextInput>({
  project_id: props.projectId,
  agent_id: '',
  task_id: '',
  title: '',
  content: '',
  tags: []
})

const tagsInput = ref('')

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.agent_id = props.initial?.agent_id ?? ''
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

const submitting = ref(false)

async function onSubmit() {
  commitTags()
  submitting.value = true
  try {
    await emit('submit', { ...form })
    open.value = false
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="initial ? 'Edit context' : 'Share context'" :icon="initial ? 'i-lucide-pencil' : 'i-lucide-plus'" :ui="{ content: 'sm:max-w-3xl' }">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Title" required>
          <UInput v-model="form.title" placeholder="API Documentation" required />
        </UFormField>
        <UFormField label="Authoring agent" required>
          <USelect
            v-model="form.agent_id"
            :items="agents?.map(a => ({ label: a.name, value: a.id })) ?? []"
            value-key="value"
            placeholder="Select an agent"
          />
        </UFormField>
        <UFormField label="Content (Markdown)" required>
          <UTextarea v-model="form.content" :rows="10" placeholder="# Heading&#10;&#10;Write in **markdown**…" />
        </UFormField>
        <UFormField label="Tags (comma separated)">
          <UInput v-model="tagsInput" placeholder="api, documentation, invoices" @blur="commitTags" />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">
          {{ initial ? 'Save changes' : 'Publish' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
