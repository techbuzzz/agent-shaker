<script setup lang="ts">
import { statusItems } from '~/types/api'
import { PERSON_KINDS, PERSON_KIND_LABELS } from '~/types/admin'
import type { CreatePersonInput, PersonKind } from '~/types/admin'

const props = defineProps<{ modelValue: boolean }>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'submit', payload: CreatePersonInput): Promise<void> | void
}>()

// UModal binds `open` in Nuxt UI v4; the parent components in this app all
// speak `modelValue`, so the two are bridged here rather than at the call site.
const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const kindItems = statusItems(PERSON_KINDS, PERSON_KIND_LABELS)

const form = reactive<{ display_name: string; email: string; kind: PersonKind }>({
  display_name: '',
  email: '',
  kind: 'human'
})

const submitting = ref(false)

// Reset on every open so a cancelled form never leaks half-typed values into
// the next one.
watch(open, (v) => {
  if (!v) return
  form.display_name = ''
  form.email = ''
  form.kind = 'human'
})

async function onSubmit() {
  submitting.value = true
  try {
    const payload: CreatePersonInput = { kind: form.kind, display_name: form.display_name.trim() }
    const email = form.email.trim()
    if (email) payload.email = email
    await emit('submit', payload)
    open.value = false
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" title="Add principal" icon="i-lucide-user-plus">
    <template #body>
      <UForm :state="form" class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Display name" required help="Shown everywhere this principal is referenced.">
          <UInput v-model="form.display_name" placeholder="Victor Buzin" required />
        </UFormField>

        <UFormField label="Kind" required>
          <USelect v-model="form.kind" :items="kindItems" value-key="value" />
        </UFormField>

        <UFormField
          label="Email"
          help="Optional. Used for identification only — the server does not send mail to it."
        >
          <UInput v-model="form.email" type="email" placeholder="name@example.com" />
        </UFormField>

        <p class="text-xs text-muted">
          <UIcon name="i-lucide-info" class="w-3.5 h-3.5 inline" />
          Only a <span class="font-medium">human</span> principal may hold approval rights; a
          <span class="font-medium">service</span> principal is a machine caller and cannot approve.
        </p>
      </UForm>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">Cancel</UButton>
        <UButton color="primary" :loading="submitting" @click="onSubmit">Add principal</UButton>
      </div>
    </template>
  </UModal>
</template>