<script setup lang="ts">
import { statusItems } from '~/types/api'
import { KEY_SCOPES, KEY_SCOPE_LABELS } from '~/types/admin'
import type { CreateKeyInput, KeyScope, Person } from '~/types/admin'
import { NO_SELECTION, optionalId } from '~/utils/select'
import type { Project } from '~/types/api'
import type { FormValidator } from '~/utils/formValidation'
import { compose, requiredChoice } from '~/utils/formValidation'

const props = defineProps<{
  modelValue: boolean
  people: Person[]
  projects: Project[]
  /** Pre-selected principal, e.g. when minting from a person's "Keys" button. */
  principalId?: string
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer — see the note in useFormModal.
   */
  onSubmit: (payload: CreateKeyInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const scopeItems = statusItems(KEY_SCOPES, KEY_SCOPE_LABELS)

const principalItems = computed(() =>
  props.people.map(p => ({ label: `${p.display_name} · ${p.kind}`, value: p.id }))
)

const projectItems = computed(() => [
  { label: 'All projects', value: NO_SELECTION },
  ...props.projects.map(p => ({ label: p.name, value: p.id }))
])

type KeyForm = { principal_id: string; project_id: string; scopes: KeyScope[] }

const form = reactive<KeyForm>({
  principal_id: '',
  project_id: NO_SELECTION,
  scopes: ['read']
})

/**
 * A key with no scopes would authenticate nothing, so it is a mistake rather
 * than a minimal request. This used to be a silently disabled button, which
 * left the person looking at a form that refused to respond with no reason.
 */
const atLeastOneScope: FormValidator<KeyForm> = values =>
  values.scopes.length === 0 ? { scopes: 'Pick at least one scope' } : {}

const validate = compose<KeyForm>(
  requiredChoice('principal_id', 'the principal this key authenticates as'),
  atLeastOneScope
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  // "All projects" is the sentinel, not an empty id; the API distinguishes an
  // omitted project_id (server-wide key) from a project-bound one.
  onSubmit: (payload) => {
    const input: CreateKeyInput = {
      principal_id: payload.principal_id,
      scopes: [...payload.scopes]
    }
    const projectId = optionalId(payload.project_id)
    if (projectId) input.project_id = projectId
    return props.onSubmit(input)
  },
  close: () => { open.value = false },
  failureMessage: 'Could not issue the key'
})

watch(open, (v) => {
  if (!v) return
  form.principal_id = props.principalId ?? ''
  form.project_id = ''
  form.scopes = ['read']
})
</script>

<template>
  <FormModal
    v-model:open="open"
    title="Issue API key"
    description="A key acts as its principal on every call it makes."
    icon="i-lucide-key-round"
    submit-label="Issue key"
    :submitting="submitting"
    :error="submitError"
    @submit="submit"
  >
    <FormField
      label="Principal"
      name="principal_id"
      required
      autofocus
      :hint="people.length
        ? 'Who this key authenticates as.'
        : 'The directory is empty — add a principal first.'"
      :error="errors.principal_id"
    >
      <USelect
        v-model="form.principal_id"
        name="principal_id"
        class="w-full"
        :items="principalItems"
        value-key="value"
        :disabled="people.length === 0"
        placeholder="Select a principal"
      />
    </FormField>

    <FormField
      label="Scopes"
      name="scopes"
      required
      hint="At least one. `admin` grants access to this very page."
      :error="errors.scopes"
    >
      <USelectMenu
        v-model="form.scopes"
        name="scopes"
        :items="scopeItems"
        value-key="value"
        multiple
        placeholder="Select scopes"
        class="w-full"
      />
    </FormField>

    <FormField
      label="Project scope"
      name="project_id"
      hint="Optional. Limits the key to one project."
      :error="errors.project_id"
    >
      <USelect
        v-model="form.project_id"
        name="project_id"
        class="w-full"
        :items="projectItems"
        value-key="value"
      />
    </FormField>
  </FormModal>
</template>