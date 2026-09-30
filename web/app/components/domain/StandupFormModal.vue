<script setup lang="ts">
import type { Standup, CreateStandupInput } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateStandupInput): Promise<void> | void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

function todayIso() {
  return new Date().toISOString().slice(0, 10)
}

const form = reactive<CreateStandupInput>({
  project_id: props.projectId,
  agent_id: '',
  date: todayIso(),
  yesterday: '',
  today: '',
  blockers: ''
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.date = todayIso()
    form.agent_id = ''
    form.yesterday = ''
    form.today = ''
    form.blockers = ''
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
  <UModal v-model:open="open" title="Submit standup" icon="i-lucide-calendar-plus">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Date" required>
            <UInput v-model="form.date" type="date" required />
          </UFormField>
          <UFormField label="Agent" required>
            <USelect
              v-model="form.agent_id"
              :items="agents?.map(a => ({ label: a.name, value: a.id })) ?? []"
              value-key="value"
              placeholder="Select an agent"
            />
          </UFormField>
        </div>
        <UFormField label="Yesterday">
          <UTextarea v-model="form.yesterday" :rows="3" placeholder="What did you ship yesterday?" />
        </UFormField>
        <UFormField label="Today">
          <UTextarea v-model="form.today" :rows="3" placeholder="What's the plan for today?" />
        </UFormField>
        <UFormField label="Blockers">
          <UTextarea v-model="form.blockers" :rows="2" placeholder="Anything in the way?" />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">Submit</UButton>
      </div>
    </template>
  </UModal>
</template>
