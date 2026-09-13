<script setup lang="ts">
import type { NuxtLinkProps } from '#app'

interface NavItem { label: string; to: string; icon: string }
const items: NavItem[] = [
  { label: 'Dashboard',     to: '/',              icon: 'i-lucide-layout-dashboard' },
  { label: 'Projects',      to: '/projects',      icon: 'i-lucide-folder-kanban' },
  { label: 'Agents',        to: '/agents',        icon: 'i-lucide-bot' },
  { label: 'Tasks',         to: '/tasks',         icon: 'i-lucide-list-checks' },
  { label: 'Documentation', to: '/documentation', icon: 'i-lucide-book-text' },
  { label: 'Standups',      to: '/standups',      icon: 'i-lucide-calendar-days' },
  { label: 'Settings',      to: '/settings',      icon: 'i-lucide-settings' }
]

defineProps<{ collapsed?: boolean }>()
defineEmits<{ (e: 'toggle'): void }>()
</script>

<template>
  <aside
    class="sticky top-0 h-screen bg-elevated border-r border-default transition-[width] duration-200 ease-out flex flex-col"
    :class="collapsed ? 'w-16' : 'w-60'"
  >
    <!-- Brand header -->
    <div class="px-3 h-16 flex items-center border-b border-default flex-shrink-0">
      <NuxtLink to="/" class="flex items-center gap-2 overflow-hidden" :aria-label="'Home'">
        <img src="/icon.png" alt="" class="w-8 h-8 rounded-lg flex-shrink-0" />
        <Transition>
          <div v-if="!collapsed" class="flex flex-col leading-tight">
            <span class="text-sm font-semibold bg-gradient-to-r from-indigo-600 to-violet-600 bg-clip-text text-transparent">Agent Shaker</span>
            <span class="text-[10px] uppercase tracking-wider text-muted">MCP Task Tracker</span>
          </div>
        </Transition>
      </NuxtLink>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 overflow-y-auto py-4 px-2 space-y-1">
      <NuxtLink
        v-for="item in items"
        :key="item.to"
        :to="item.to as NuxtLinkProps['to']"
        class="flex items-center gap-3 px-2 py-2 rounded-md text-sm font-medium text-muted hover:text-highlighted hover:bg-elevated/60 transition-colors"
        active-class="text-primary bg-primary/10 hover:bg-primary/15"
        exact-active-class="text-primary bg-primary/10"
        :title="collapsed ? item.label : undefined"
      >
        <UIcon :name="item.icon" class="w-5 h-5 flex-shrink-0" />
        <Transition>
          <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
        </Transition>
      </NuxtLink>
    </nav>

    <!-- Collapse toggle -->
    <div class="border-t border-default p-2 flex-shrink-0">
      <UButton
        :icon="collapsed ? 'i-lucide-panel-left-open' : 'i-lucide-panel-left-close'"
        :aria-label="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
        variant="ghost"
        color="neutral"
        size="sm"
        block
        @click="$emit('toggle')"
      >
        <template v-if="!collapsed">Collapse</template>
      </UButton>
    </div>
  </aside>
</template>
