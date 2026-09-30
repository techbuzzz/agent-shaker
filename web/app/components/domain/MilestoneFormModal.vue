<script setup lang="ts">
import type { CreateMilestoneInput, MilestoneStatus } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  createdBy: string
  initial?: { title: string; description?: string; status?: MilestoneStatus; target_date?: string } | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateMilestoneInput): Promise<void> | void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const form = reactive<CreateMilestoneInput>({
  project_id: props.projectId,
  title: '',
  description: '',
  status: 'planned',
  target_date: '',
  created_by: props.createdBy
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

const submitting = ref(false)

async function onSubmit() {
  submitting.value = true
  try {
    const payload: CreateMilestoneInput = {
      project_id: form.project_id,
      title: form.title,
      created_by: form.created_by
    }
    if (form.description) payload.description = form.description
    if (form.status) payload.status = form.status
    if (form.target_date) payload.target_date = form.target_date
    await emit('submit', payload)
    open.value = false
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="initial ? 'Edit milestone' : 'Create milestone'" icon="i-lucide-flag">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Title" required>
          <UInput v-model="form.title" placeholder="M1 — Auth MVP" required />
        </UFormField>
        <UFormField label="Description">
          <UTextarea v-model="form.description" :rows="3" placeholder="What does this milestone deliver?" />
        </UFormField>
        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Status">
            <USelect
              v-model="form.status"
              :items="[
                { label: 'Planned', value: 'planned' },
                { label: 'Active',  value: 'active' },
                { label: 'Done',    value: 'done' },
                { label: 'Dropped', value: 'dropped' }
              ]"
              value-key="value"
            />
          </UFormField>
          <UFormField label="Target date">
            <UInput v-model="form.target_date" type="date" />
          </UFormField>
        </div>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">
          {{ initial ? 'Save changes' : 'Create milestone' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
