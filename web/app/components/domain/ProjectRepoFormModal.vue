<script setup lang="ts">
import type { CreateProjectRepoInput } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
  initial?: { url: string; branch?: string; role?: string; agent_id?: string } | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateProjectRepoInput): Promise<void> | void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const form = reactive<CreateProjectRepoInput>({
  project_id: props.projectId,
  url: '',
  branch: 'main',
  role: 'code',
  agent_id: ''
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.url = props.initial?.url ?? ''
    form.branch = props.initial?.branch ?? 'main'
    form.role = props.initial?.role ?? 'code'
    form.agent_id = props.initial?.agent_id ?? ''
  }
})

const submitting = ref(false)
async function onSubmit() {
  submitting.value = true
  try {
    const payload: CreateProjectRepoInput = {
      project_id: form.project_id,
      url: form.url
    }
    if (form.branch) payload.branch = form.branch
    if (form.role) payload.role = form.role
    if (form.agent_id) payload.agent_id = form.agent_id
    await emit('submit', payload)
    open.value = false
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" title="Add repository" icon="i-lucide-git-branch">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Git URL" required>
          <UInput v-model="form.url" placeholder="git@github.com:org/repo.git" required />
        </UFormField>
        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Branch">
            <UInput v-model="form.branch" placeholder="main" />
          </UFormField>
          <UFormField label="Role">
            <USelect
              v-model="form.role"
              :items="[
                { label: 'Code',   value: 'code' },
                { label: 'Infra',  value: 'infra' },
                { label: 'Docs',   value: 'docs' },
                { label: 'Design', value: 'design' }
              ]"
              value-key="value"
            />
          </UFormField>
        </div>
        <UFormField label="Owning agent (optional)">
          <USelect
            v-model="form.agent_id"
            :items="[{ label: '— unassigned —', value: '' }, ...(agents?.map(a => ({ label: a.name, value: a.id })) ?? [])]"
            value-key="value"
          />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">Add repo</UButton>
      </div>
    </template>
  </UModal>
</template>
