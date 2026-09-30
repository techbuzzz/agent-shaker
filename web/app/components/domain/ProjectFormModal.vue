<script setup lang="ts">
import type { Project, CreateProjectInput } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  initial?: Project | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateProjectInput): Promise<void> | void
}>()

const isEdit = computed(() => !!props.initial)
const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const form = reactive<CreateProjectInput>({
  name: '',
  description: '',
  status: 'active'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.name = props.initial?.name ?? ''
    form.description = props.initial?.description ?? ''
    form.status = props.initial?.status ?? 'active'
  }
})

const submitting = ref(false)
const formRef = ref<{ clear: () => void } | null>(null)

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
  <UModal v-model:open="open" :title="isEdit ? 'Edit project' : 'Create project'" :icon="isEdit ? 'i-lucide-pencil' : 'i-lucide-plus'">
    <template #body>
      <UForm ref="formRef" :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Name" required>
          <UInput v-model="form.name" placeholder="InvoiceAI" required />
        </UFormField>
        <UFormField label="Description">
          <UTextarea v-model="form.description" placeholder="What is this project about?" :rows="3" />
        </UFormField>
        <UFormField label="Status">
          <USelect v-model="form.status" :items="[
            { label: 'Active', value: 'active' },
            { label: 'Archived', value: 'archived' }
          ]" value-key="value" />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">
          {{ isEdit ? 'Save changes' : 'Create project' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
