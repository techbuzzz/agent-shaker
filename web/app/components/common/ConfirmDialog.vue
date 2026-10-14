<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: boolean
  title?: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  color?: 'primary' | 'error' | 'warning'
  icon?: string
  loading?: boolean
}>(), {
  title: 'Are you sure?',
  description: '',
  confirmLabel: 'Confirm',
  cancelLabel: 'Cancel',
  color: 'error',
  icon: 'i-lucide-triangle-alert',
  loading: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

function close() { emit('update:modelValue', false) }
function confirm() { emit('confirm'); emit('update:modelValue', false) }
function cancel() { emit('cancel'); emit('update:modelValue', false) }
</script>

<template>
  <!--
    `open`, not `model-value`. UModal (Nuxt UI v4) takes its visibility from the
    `open` prop and emits `update:open`; passing model-value forwarded an
    unrecognised attribute to the dialog root and left the modal permanently
    closed. The component's own public API stays v-model so its caller is
    unaffected.
  -->
  <UModal
    :open="props.modelValue"
    :title="title"
    :description="description"
    :icon="icon"
    @update:open="(v: boolean) => emit('update:modelValue', v)"
  >
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="cancel">{{ cancelLabel }}</UButton>
        <UButton :color="color" :loading="loading" @click="confirm">{{ confirmLabel }}</UButton>
      </div>
    </template>
  </UModal>
</template>
