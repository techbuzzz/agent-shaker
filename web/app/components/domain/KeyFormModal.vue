<script setup lang="ts">
import { statusItems } from '~/types/api'
import { KEY_SCOPES, KEY_SCOPE_LABELS } from '~/types/admin'
import type { CreateKeyInput, KeyScope, Person } from '~/types/admin'
import type { Project } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  people: Person[]
  projects: Project[]
  /** Pre-selected principal, e.g. when minting from a person's "Keys" button. */
  principalId?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreateKeyInput): Promise<void> | void
}>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const scopeItems = statusItems(KEY_SCOPES, KEY_SCOPE_LABELS)

const principalItems = computed(() =>
  props.people.map(p => ({ label: `${p.display_name} · ${p.kind}`, value: p.id }))
)

const projectItems = computed(() => [
  { label: 'All projects', value: '' },
  ...props.projects.map(p => ({ label: p.name, value: p.id }))
])

const form = reactive<{ principal_id: string; project_id: string; scopes: KeyScope[] }>({
  principal_id: '',
  project_id: '',
  scopes: ['read']
})

const submitting = ref(false)

watch(open, (v) => {
  if (!v) return
  form.principal_id = props.principalId ?? ''
  form.project_id = ''
  form.scopes = ['read']
})

async function onSubmit() {
  if (!form.principal_id || form.scopes.length === 0) return
  submitting.value = true
  try {
    const payload: CreateKeyInput = {
      principal_id: form.principal_id,
      scopes: [...form.scopes]
    }
    if (form.project_id) payload.project_id = form.project_id
    await emit('submit', payload)
    open.value = false
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" title="Issue API key" icon="i-lucide-key-round">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <UFormField
          label="Principal"
          required
          :help="people.length ? 'Who this key authenticates as.' : 'The directory is empty — add a principal first.'"
        >
          <USelect
            v-model="form.principal_id"
            :items="principalItems"
            value-key="value"
            :disabled="people.length === 0"
            placeholder="Select a principal"
          />
        </UFormField>

        <UFormField label="Scopes" required help="At least one. `admin` grants access to this very page.">
          <USelectMenu
            v-model="form.scopes"
            :items="scopeItems"
            value-key="value"
            multiple
            placeholder="Select scopes"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Project scope" help="Optional. Limits the key to one project.">
          <USelect v-model="form.project_id" :items="projectItems" value-key="value" />
        </UFormField>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton
          color="primary"
          :loading="submitting"
          :disabled="!form.principal_id || form.scopes.length === 0"
          @click="onSubmit"
        >
          Issue key
        </UButton>
      </div>
    </template>
  </UModal>
</template>