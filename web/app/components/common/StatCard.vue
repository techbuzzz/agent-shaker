<script setup lang="ts">
interface Breakdown { label: string; value: number; color?: string }

const props = withDefaults(defineProps<{
  title: string
  value: number | string
  icon?: string
  breakdown?: Breakdown[]
  href?: string
  accent?: 'primary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
}>(), {
  accent: 'primary',
  icon: 'i-lucide-activity',
  breakdown: () => [],
  href: undefined
})

const accentColor = computed(() => {
  switch (props.accent) {
    case 'success': return 'text-emerald-600 dark:text-emerald-400'
    case 'info':    return 'text-sky-600 dark:text-sky-400'
    case 'warning': return 'text-amber-600 dark:text-amber-400'
    case 'error':   return 'text-rose-600 dark:text-rose-400'
    case 'neutral': return 'text-slate-600 dark:text-slate-400'
    default:        return 'text-indigo-600 dark:text-indigo-400'
  }
})

const bgGradient = computed(() => {
  switch (props.accent) {
    case 'success': return 'from-emerald-500/15 to-emerald-500/0'
    case 'info':    return 'from-sky-500/15 to-sky-500/0'
    case 'warning': return 'from-amber-500/15 to-amber-500/0'
    case 'error':   return 'from-rose-500/15 to-rose-500/0'
    case 'neutral': return 'from-slate-500/15 to-slate-500/0'
    default:        return 'from-indigo-500/15 to-indigo-500/0'
  }
})
</script>

<template>
  <component
    :is="href ? resolveComponent('NuxtLink') : 'div'"
    v-bind="href ? { to: href } : {}"
    class="block rounded-xl bg-elevated border border-default hover:border-primary/30 transition-all hover:shadow-sm group"
  >
    <div class="p-5">
      <div class="flex items-start justify-between mb-3">
        <div class="flex items-center gap-3">
          <div
            class="w-10 h-10 rounded-lg flex items-center justify-center bg-gradient-to-br"
            :class="bgGradient"
          >
            <UIcon :name="icon" class="w-5 h-5" :class="accentColor" />
          </div>
          <span class="text-sm font-medium text-muted">{{ title }}</span>
        </div>
        <UIcon
          v-if="href"
          name="i-lucide-arrow-right"
          class="w-4 h-4 text-muted opacity-0 group-hover:opacity-100 transition-opacity"
        />
      </div>

      <div class="text-3xl font-semibold tabular-nums">{{ value }}</div>

      <div v-if="breakdown.length" class="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs">
        <div
          v-for="row in breakdown"
          :key="row.label"
          class="inline-flex items-center gap-1.5"
        >
          <span
            class="inline-block w-2 h-2 rounded-full"
            :style="{ backgroundColor: row.color }"
          />
          <span class="text-muted">{{ row.label }}:</span>
          <span class="font-medium">{{ row.value }}</span>
        </div>
      </div>
    </div>
  </component>
</template>
