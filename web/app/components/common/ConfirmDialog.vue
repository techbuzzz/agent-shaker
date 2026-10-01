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
  <UModal
    :model-value="props.modelValue"
    :title="title"
    :description="description"
    :icon="icon"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  >
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="cancel">{{ cancelLabel }}</UButton>
        <UButton :color="color" :loading="loading" @click="confirm">{{ confirmLabel }}</UButton>
      </div>
    </template>
  </UModal>
</template>
