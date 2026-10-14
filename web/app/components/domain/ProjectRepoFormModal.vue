<script setup lang="ts">
import type { CreateProjectRepoInput } from '~/types/api'
import { compose, required } from '~/utils/formValidation'
import { NO_SELECTION, optionalId } from '~/utils/select'

/**
 * A repository URL is the one field here where a typo is invisible until
 * something tries to clone it, so it gets its own check instead of the
 * database's "not null".
 */
function gitUrl(value: unknown): string | null {
  const url = String(value ?? '').trim()
  if (url === '') return 'A repository URL is required'
  // Accepts the two forms an operator actually pastes: an https/ssh remote and
  // an scp-style git@host:org/repo.git.
  const looksLikeRemote = /^(https?:\/\/|git@|ssh:\/\/|git:\/\/)/i.test(url)
  if (!looksLikeRemote) return 'Use a full git remote, for example git@github.com:org/repo.git'
  if (!/\.[a-z]{2,}([:/]|$)/i.test(url) && !/@[^:]+:/.test(url)) {
    return 'That does not look like a repository remote'
  }
  return null
}

const props = defineProps<{
  modelValue: boolean
  projectId: string
  agents?: Array<{ id: string; name: string }>
  initial?: { url: string; branch?: string; role?: string; agent_id?: string } | null
  onSubmit: (payload: CreateProjectRepoInput) => Promise<unknown>
}>()

const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const form = reactive<CreateProjectRepoInput>({
  project_id: props.projectId,
  url: '',
  branch: 'main',
  role: 'code',
  agent_id: NO_SELECTION
})

const validate = compose<CreateProjectRepoInput>(
  required('url', 'A repository URL'),
  (values) => {
    const message = gitUrl(values.url)
    return message ? { url: message } : {}
  }
)

const { errors, submitting, submitError, submit } = useFormModal({
  form,
  validate,
  onSubmit: (values) => {
    const payload: CreateProjectRepoInput = { project_id: values.project_id, url: values.url }
    if (values.branch) payload.branch = values.branch
    if (values.role) payload.role = values.role
    const agentId = optionalId(values.agent_id)
    if (agentId) payload.agent_id = agentId
    return props.onSubmit(payload)
  },
  close: () => { open.value = false },
  failureMessage: 'Could not add the repository'
})

watch(() => props.modelValue, (v) => {
  if (v) {
    form.project_id = props.projectId
    form.url = props.initial?.url ?? ''
    form.branch = props.initial?.branch ?? 'main'
    form.role = props.initial?.role ?? 'code'
    form.agent_id = props.initial?.agent_id ?? NO_SELECTION
  }
})
</script>

<template>
  <FormModal
    v-model:open="open"
    :title="initial ? 'Edit repository' : 'Add repository'"
    description="Repositories tell agents where the work for this project actually lives."
    icon="i-lucide-git-branch"
    :submit-label="initial ? 'Save changes' : 'Add repository'"
    :submitting="submitting"
    :error="submitError"
    @submit="submit"
  >
    <FormField
      label="Git URL"
      name="url"
      required
      :hint="'The remote agents clone. ssh and https both work.'"
      :error="errors.url"
    >
      <UInput
        v-model="form.url"
        name="url"
        autofocus
        placeholder="git@github.com:org/repo.git"
        class="w-full font-mono"
      />
    </FormField>

    <div class="grid gap-5 sm:grid-cols-2">
      <FormField
        label="Branch"
        name="branch"
        :hint="'The branch agents work from.'"
        :error="errors.branch"
      >
        <UInput v-model="form.branch" name="branch" placeholder="main" class="w-full font-mono" />
      </FormField>

      <FormField
        label="Role"
        name="role"
        :hint="'What this repository is to the project.'"
        :error="errors.role"
      >
        <USelect
          v-model="form.role"
          name="role"
          class="w-full"
          :items="[
            { label: 'Code',   value: 'code' },
            { label: 'Infra',  value: 'infra' },
            { label: 'Docs',   value: 'docs' },
            { label: 'Design', value: 'design' }
          ]"
          value-key="value"
        />
      </FormField>
    </div>

    <FormField
      label="Owning agent"
      name="agent_id"
      hint="Optional. Defaults to whoever picks the task up."
      :error="errors.agent_id"
    >
      <USelect
        v-model="form.agent_id"
        name="agent_id"
        class="w-full"
        :items="[
          { label: 'Unassigned', value: NO_SELECTION },
          ...(agents?.map(a => ({ label: a.name, value: a.id })) ?? [])
        ]"
        value-key="value"
      />
    </FormField>
  </FormModal>
</template>