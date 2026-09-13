<script setup lang="ts">
import type { Task, CreateTaskInput, TaskPriority } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
  initial?: Task | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateTaskInput): Promise<void> | void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const form = reactive<CreateTaskInput>({
  project_id: props.projectId,
  title: '',
  description: '',
  priority: 'medium' as TaskPriority,
  assigned_to: '',
  created_by: ''
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.title = props.initial?.title ?? ''
    form.description = props.initial?.description ?? ''
    form.priority = props.initial?.priority ?? 'medium'
    form.assigned_to = props.initial?.assigned_to ?? ''
    form.created_by = props.initial?.created_by ?? ''
  }
})

const submitting = ref(false)

async function onSubmit() {
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
  <UModal v-model:open="open" :title="initial ? 'Edit task' : 'Create task'" :icon="initial ? 'i-lucide-pencil' : 'i-lucide-plus'">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Title" required>
          <UInput v-model="form.title" placeholder="Implement user authentication" required />
        </UFormField>
        <UFormField label="Description">
          <UTextarea v-model="form.description" :rows="4" placeholder="What needs to be done?" />
        </UFormField>
        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Priority">
            <USelect v-model="form.priority" :items="[
              { label: 'Low', value: 'low' },
              { label: 'Medium', value: 'medium' },
              { label: 'High', value: 'high' }
            ]" value-key="value" />
          </UFormField>
          <UFormField label="Assigned to">
            <USelect
              v-model="form.assigned_to"
              :items="[{ label: 'Unassigned', value: '' }, ...(agents?.map(a => ({ label: a.name, value: a.id })) ?? [])]"
              value-key="value"
            />
          </UFormField>
        </div>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">
          {{ initial ? 'Save changes' : 'Create task' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
