<script setup lang="ts">
/**
 * One labelled field.
 *
 * The distinction it draws is between a hint (always visible, tells you how to
 * fill the field in) and an error (visible only once you have tried to submit,
 * says what is wrong with what you typed). Nuxt UI collapses both into `help`,
 * which is why the old forms could not show a failure without also permanently
 * occupying space with advice.
 *
 * The label row is rendered here rather than left to Nuxt UI because the field
 * also carries two things Nuxt UI has no place for: the character counter and
 * the required marker next to the label instead of orphaned under it.
 */
const props = withDefaults(defineProps<{
  label: string
  /** Field name, used to wire the message to the control for screen readers. */
  name: string
  required?: boolean
  hint?: string
  error?: string
  /** Rendered on the right of the label row — a character counter, a unit. */
  aside?: string
}>(), {
  required: false,
  hint: undefined,
  error: undefined,
  aside: undefined
})

/**
 * `required` is read through a computed rather than straight in the template.
 * Vue's compiler resolves a bare `required` in a template to a globally
 * injected helper, so `v-if="required"` type-checks as "always defined" and the
 * asterisk would render on every field in the app.
 */
const isRequired = computed(() => props.required)
</script>

<template>
  <UFormField
    :name="name"
    :error="error"
    :description="hint"
    class="w-full"
  >
    <template #label>
      <span class="flex w-full items-baseline justify-between gap-3">
        <span class="text-sm font-medium text-default">
          {{ label }}
          <span v-if="isRequired" class="text-error" aria-hidden="true">*</span>
          <span v-if="isRequired" class="sr-only">(required)</span>
        </span>
        <span v-if="aside" class="text-xs font-normal text-muted tabular-nums">{{ aside }}</span>
      </span>
    </template>

    <slot />
  </UFormField>
</template>