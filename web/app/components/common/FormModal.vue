<script setup lang="ts">
/**
 * The dialog every create/edit form in the app sits in.
 *
 * What it guarantees, so no individual form has to:
 *   - one column, full-width controls. The old forms laid fields out at
 *     roughly half the dialog width and left the right half empty, which read
 *     as broken rather than airy;
 *   - a heading that says what will happen ("Create project"), not a category
 *     ("Project");
 *   - one place for a failed request to be reported, above the fields, so the
 *     message cannot be missed by someone who has already looked at the
 *     inputs;
 *   - a submit button that reflects the real state of the request, and a
 *     dismiss path that is disabled while the write is in flight, so a dialog
 *     cannot vanish underneath an unanswered request.
 */
const props = withDefaults(defineProps<{
  open: boolean
  title: string
  /** One line under the title: what this dialog is for. */
  description?: string
  icon?: string
  submitLabel: string
  submitting?: boolean
  /** Message for a request that reached the server and was refused. */
  error?: string | null
  /** 'sm' for a 2-3 field form, 'md' default, 'lg' for markdown-length input. */
  size?: 'sm' | 'md' | 'lg'
}>(), {
  description: undefined,
  icon: undefined,
  submitting: false,
  error: null,
  size: 'md'
})

const emit = defineEmits<{
  (e: 'update:open', v: boolean): void
  (e: 'submit'): void
}>()

function setOpen(v: boolean) {
  // Refuse to close mid-flight: the write would finish against a dialog that
  // no longer exists and its result would have nowhere to go.
  if (!v && props.submitting) return
  emit('update:open', v)
}

const proxyOpen = computed({
  get: () => props.open,
  set: setOpen
})

const widths = { sm: 'sm:max-w-md', md: 'sm:max-w-xl', lg: 'sm:max-w-3xl' } as const
</script>

<template>
  <UModal
    v-model:open="proxyOpen"
    :title="title"
    :description="description"
    :icon="icon"
    :class="widths[size]"
    :close-on-click-away="!submitting"
    :close-on-escape="!submitting"
  >
    <template #body>
      <form
        novalidate
        class="flex flex-col gap-5"
        @submit.prevent="emit('submit')"
      >
        <!--
          The request failed as a whole, not per field: the payload was fine but
          the server said no. It sits above the inputs, next to the button that
          caused it, and is announced so it is not a purely visual addition.
        -->
        <div
          v-if="error"
          role="alert"
          class="flex items-start gap-3 rounded-lg border border-error/40 bg-error/10 px-4 py-3 text-sm text-error"
        >
          <UIcon name="i-lucide-circle-alert" class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <p class="leading-snug">{{ error }}</p>
        </div>

        <slot />
      </form>
    </template>

    <template #footer>
      <div class="flex w-full items-center justify-end gap-2">
        <UButton
          variant="ghost"
          color="neutral"
          :disabled="submitting"
          @click="setOpen(false)"
        >
          Cancel
        </UButton>
        <UButton
          type="submit"
          color="primary"
          :loading="submitting"
          @click="emit('submit')"
        >
          {{ submitLabel }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>