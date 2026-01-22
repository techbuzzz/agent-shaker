<template>
  <div id="app" class="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50">
    <nav class="sticky top-0 z-50 bg-white/80 backdrop-blur-md border-b border-slate-200 shadow-soft">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex flex-col sm:flex-row justify-between items-center py-4 gap-3">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
              <span class="text-white font-bold text-sm">🚀</span>
            </div>
            <h1 class="text-lg sm:text-xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
              MCP Task Tracker
            </h1>
          </div>
          <div class="flex flex-wrap justify-center gap-1 sm:gap-2">
            <router-link
              to="/"
              class="nav-link"
              active-class="nav-link-active"
            >
              <span class="hidden sm:inline">Dashboard</span>
              <span class="sm:hidden">🏠</span>
            </router-link>
            <router-link
              to="/projects"
              class="nav-link"
              active-class="nav-link-active"
            >
              <span class="hidden sm:inline">Projects</span>
              <span class="sm:hidden">📁</span>
            </router-link>
            <router-link
              to="/agents"
              class="nav-link"
              active-class="nav-link-active"
            >
              <span class="hidden sm:inline">Agents</span>
              <span class="sm:hidden">🤖</span>
            </router-link>
            <router-link
              to="/tasks"
              class="nav-link"
              active-class="nav-link-active"
            >
              <span class="hidden sm:inline">Tasks</span>
              <span class="sm:hidden">📋</span>
            </router-link>
            <a
              href="https://github.com/techbuzzz/agent-shaker"
              target="_blank"
              rel="noopener noreferrer"
              class="nav-link"
            >
              <span class="hidden sm:inline">GitHub</span>
              <span class="sm:hidden">⭐</span>
            </a>
          </div>
        </div>
      </div>
    </nav>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
      <router-view v-slot="{ Component }">
        <transition name="page" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>

    <footer class="bg-white/50 backdrop-blur-sm border-t border-slate-200 py-6 sm:py-8 mt-12 sm:mt-16">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <div class="flex flex-col sm:flex-row items-center justify-center gap-4">
          <div class="flex items-center gap-2">
            <div class="w-6 h-6 bg-gradient-to-br from-blue-500 to-purple-600 rounded-md flex items-center justify-center">
              <span class="text-white font-bold text-xs">🚀</span>
            </div>
            <span class="text-slate-600 text-sm">© 2026 MCP Task Tracker</span>
          </div>
          <span class="text-slate-500 text-sm">AI Agent Coordination System</span>
        </div>
      </div>
    </footer>
  </div>
</template>

<script>
export default {
  name: 'App'
}
</script>

<style>
.nav-link {
  color: rgb(55 65 81);
  padding: 0.5rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s ease;
  text-decoration: none;
  display: inline-block;
}

.nav-link:hover {
  color: rgb(37 99 235);
  background-color: rgb(239 246 255);
}

.nav-link-active {
  background-color: rgb(37 99 235);
  color: white;
}

.nav-link-active:hover {
  background-color: rgb(29 78 216);
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes fadeOut {
  from {
    opacity: 1;
  }
  to {
    opacity: 0;
  }
}

.page-enter-active, .page-leave-active {
  transition: opacity 0.3s ease;
}

.page-enter {
  opacity: 0;
}

.page-leave-to {
  opacity: 0;
}
</style>
