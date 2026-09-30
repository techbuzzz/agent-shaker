<script setup lang="ts">
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = withDefaults(defineProps<{
  source?: string | null
  compact?: boolean
}>(), {
  source: '',
  compact: false
})

marked.setOptions({ gfm: true, breaks: false })

const html = computed(() => {
  if (!props.source) return ''
  // marked.parse is sync for non-async sources
  const raw = marked.parse(props.source) as string
  if (typeof window === 'undefined') {
    // Server-side rendering: minimal sanitization via a regex strip for script tags.
    // DOMPurify's jsdom setup requires extra config; we keep this conservative.
    return raw.replace(/<script[\s\S]*?<\/script>/gi, '').replace(/on[a-z]+="[^"]*"/gi, '')
  }
  return DOMPurify.sanitize(raw, { USE_PROFILES: { html: true } })
})
</script>

<template>
  <article
    class="markdown-body"
    :class="compact ? 'text-sm' : ''"
    v-html="html"
  />
</template>
