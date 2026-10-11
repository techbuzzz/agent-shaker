<script setup lang="ts">
import { renderMarkdown } from '~/utils/markdown'

const props = withDefaults(defineProps<{
  source?: string | null
  compact?: boolean
}>(), {
  source: '',
  compact: false
})

// Sanitisation lives in the utility and runs on both sides of the render — see
// app/utils/markdown.ts for why the server pass is the security boundary and
// why the previous regex strip was not one.
const html = computed(() => renderMarkdown(props.source))
</script>

<template>
  <article
    class="markdown-body"
    :class="compact ? 'text-sm' : ''"
    v-html="html"
  />
</template>
