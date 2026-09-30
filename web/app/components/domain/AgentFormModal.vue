<script setup lang="ts">
import type { Agent, CreateAgentInput, AgentStatus } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  projectId: string
  initial?: Agent | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateAgentInput): Promise<void> | void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const form = reactive<CreateAgentInput>({
  project_id: props.projectId,
  name: '',
  role: 'backend',
  team: '',
  status: 'active' as AgentStatus
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.name = props.initial?.name ?? ''
    form.role = props.initial?.role ?? 'backend'
    form.team = props.initial?.team ?? ''
    form.status = props.initial?.status ?? 'active'
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
  <UModal v-model:open="open" title="Register agent" icon="i-lucide-bot">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Name" required>
          <UInput v-model="form.name" placeholder="Backend-Copilot" required />
        </UFormField>
        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Role" required>
            <USelect
              v-model="form.role"
              :items="[
                { label: 'Project Manager', value: 'pm' },
                { label: 'Backend',         value: 'backend' },
                { label: 'Frontend',        value: 'frontend' },
                { label: 'DevOps',          value: 'devops' },
                { label: 'Design',          value: 'design' },
                { label: 'QA',              value: 'qa' }
              ]"
              value-key="value"
            />
          </UFormField>
          <UFormField label="Team">
            <UInput v-model="form.team" placeholder="Backend Team" />
          </UFormField>
        </div>
        <UFormField label="Initial status">
          <USelect v-model="form.status" :items="[
            { label: 'Active', value: 'active' },
            { label: 'Idle', value: 'idle' },
            { label: 'Offline', value: 'offline' }
          ]" value-key="value" />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">Register agent</UButton>
      </div>
    </template>
  </UModal>
</template>
