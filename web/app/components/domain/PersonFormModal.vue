<script setup lang="ts">
import { statusItems } from '~/types/api'
import { PERSON_KINDS, PERSON_KIND_LABELS } from '~/types/admin'
import type { CreatePersonInput, PersonKind } from '~/types/admin'
import { compose, maxLength, required } from '~/utils/formValidation'

const props = defineProps<{
  modelValue: boolean
  /**
   * Runs the write. A prop rather than a `submit` event so the dialog can wait
   * for the answer — see the note in useFormModal.
   */
  onSubmit: (payload: CreatePersonInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

// UModal binds `open` in Nuxt UI v4; the parent components in this app all
// speak `modelValue`, so the two are bridged here rather than at the call site.
const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const kindItems = statusItems(PERSON_KINDS, PERSON_KIND_LABELS)

type PersonForm = { display_name: string; email: string; kind: PersonKind }

const form = reactive<PersonForm>({
  display_name: '',
  email: '',
  kind: 'human'
})

const validate = compose<PersonForm>(
  // 255 and 320 are the column widths in migrations/009_people.up.sql
  // (display_name VARCHAR(255), email VARCHAR(320)); the limit belongs next to
  // the constraint, not only in the database.
  required('display_name', 'A display name'),
  maxLength('display_name', 255, 'The display name'),
  maxLength('email', 320, 'The email')
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  // The API input treats an absent email as a distinct case, so the always-present
  // string from the form is dropped here rather than sent as an empty address.
  onSubmit: (payload) => {
    const input: CreatePersonInput = {
      kind: payload.kind,
      display_name: payload.display_name.trim()
    }
    const email = payload.email.trim()
    if (email) input.email = email
    return props.onSubmit(input)
  },
  close: () => { open.value = false },
  failureMessage: 'Could not add the principal'
})

// Reset on every open so a cancelled form never leaks half-typed values into
// the next one.
watch(open, (v) => {
  if (!v) return
  form.display_name = ''
  form.email = ''
  form.kind = 'human'
})
</script>

<template>
  <FormModal
    v-model:open="open"
    title="Add principal"
    description="Someone or something the system attributes action to."
    icon="i-lucide-user-plus"
    submit-label="Add principal"
    :submitting="submitting"
    :error="submitError"
    size="sm"
    @submit="submit"
  >
    <FormField
      label="Display name"
      name="display_name"
      required
      hint="How this principal is named everywhere it is referenced."
      :aside="`${form.display_name.length}/255`"
      :error="errors.display_name"
    >
      <UInput
        v-model="form.display_name"
        name="display_name"
        autofocus
        placeholder="Victor Buzin"
        class="w-full"
        maxlength="255"
      />
    </FormField>

    <FormField
      label="Kind"
      name="kind"
      required
      hint="Only a human may hold approval rights; a service is a machine caller."
      :error="errors.kind"
    >
      <USelect
        v-model="form.kind"
        name="kind"
        class="w-full"
        :items="kindItems"
        value-key="value"
      />
    </FormField>

    <FormField
      label="Email"
      name="email"
      hint="Optional. Used for identification only — the server does not send mail to it."
      :error="errors.email"
    >
      <UInput
        v-model="form.email"
        name="email"
        type="email"
        placeholder="name@example.com"
        class="w-full"
        maxlength="320"
      />
    </FormField>

    <p class="text-xs text-muted">
      <UIcon name="i-lucide-info" class="w-3.5 h-3.5 inline" />
      Only a <span class="font-medium">human</span> principal may hold approval rights; a
      <span class="font-medium">service</span> principal is a machine caller and cannot approve.
    </p>
  </FormModal>
</template>