<script setup lang="ts">
import type { CreateGlobalContextInput, GlobalContextScope } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  agentId: string
  projectId?: string
  initialScope?: GlobalContextScope
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateGlobalContextInput): Promise<void> | void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const form = reactive<{ scope: GlobalContextScope; title: string; content: string; tagsRaw: string }>({
  scope: props.initialScope ?? 'project',
  title: '',
  content: '',
  tagsRaw: ''
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.scope = props.initialScope ?? 'project'
    form.title = ''
    form.content = ''
    form.tagsRaw = ''
  }
})

const submitting = ref(false)
async function onSubmit() {
  submitting.value = true
  try {
    const tags = form.tagsRaw.split(',').map(s => s.trim()).filter(Boolean)
    const payload: CreateGlobalContextInput = {
      scope: form.scope,
      agent_id: props.agentId,
      title: form.title,
      content: form.content,
      tags
    }
    if (form.scope === 'project') {
      if (!props.projectId) throw new Error('scope=project requires projectId')
      payload.project_id = props.projectId
    }
    await emit('submit', payload)
    open.value = false
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" title="Publish doc" icon="i-lucide-globe">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Scope">
            <USelect
              v-model="form.scope"
              :items="[
                { label: 'Project — visible only inside this project', value: 'project' },
                { label: 'Global — server-wide playbook',                value: 'global' }
              ]"
              value-key="value"
            />
          </UFormField>
          <UFormField label="Tags (comma-separated)">
            <UInput v-model="form.tagsRaw" placeholder="oncall, infra, playbook" />
          </UFormField>
        </div>
        <UFormField label="Title" required>
          <UInput v-model="form.title" placeholder="On-call playbook" required />
        </UFormField>
        <UFormField label="Content (markdown)">
          <UTextarea v-model="form.content" :rows="8" placeholder="# Heading" />
        </UFormField>
        <p v-if="form.scope === 'global'" class="text-xs text-muted">
          After publishing, any agent on this server can read this playbook via
          <code class="font-mono">read_resource("global://{{ form.title || '<title>' }}")</code>.
        </p>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">Publish</UButton>
      </div>
    </template>
  </UModal>
</template>
