<script setup lang="ts">
import type { Task, CreateTaskInput, TaskPriority, Milestone } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
  milestones?: Milestone[]
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

const form = reactive<{ title: string; description: string; priority: TaskPriority; assigned_to: string; created_by: string; milestone_id: string; tagsRaw: string; project_id: string }>({
  project_id: props.projectId,
  title: '',
  description: '',
  priority: 'medium',
  assigned_to: '',
  created_by: '',
  milestone_id: '',
  tagsRaw: ''
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.title = props.initial?.title ?? ''
    form.description = props.initial?.description ?? ''
    form.priority = props.initial?.priority ?? 'medium'
    form.assigned_to = props.initial?.assigned_to ?? ''
    form.created_by = props.initial?.created_by ?? ''
    form.milestone_id = props.initial?.milestone_id ?? ''
    form.tagsRaw = (props.initial?.tags ?? []).join(', ')
  }
})

const submitting = ref(false)

async function onSubmit() {
  submitting.value = true
  try {
    const tags = form.tagsRaw.split(',').map(s => s.trim()).filter(Boolean)
    const payload: CreateTaskInput = {
      project_id: form.project_id,
      title: form.title,
      description: form.description,
      priority: form.priority,
      assigned_to: form.assigned_to || undefined,
      created_by: form.created_by || undefined,
      milestone_id: form.milestone_id || undefined,
      tags: tags.length ? tags : undefined
    }
    await emit('submit', payload)
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
        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Milestone">
            <USelect
              v-model="form.milestone_id"
              :items="[{ label: '— none —', value: '' }, ...(milestones?.map(m => ({ label: m.title, value: m.id })) ?? [])]"
              value-key="value"
            />
          </UFormField>
          <UFormField label="Tags (comma-separated)" :help="'Use feature:<name> to roll this task up on the Features view.'">
            <UInput v-model="form.tagsRaw" placeholder="feature:auth, backend" />
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
