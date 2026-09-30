// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2026-09-01',
  future: {
    compatibilityVersion: 4
  },

  // App identity
  app: {
    head: {
      title: 'Agent Shaker — MCP Task Tracker',
      htmlAttrs: { lang: 'en' },
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'AI Agent Task Coordination System — Backend API & MCP Server UI.' }
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }
      ]
    }
  },

  // Auto-imports from app/components, app/composables
  imports: {
    dirs: ['composables', 'stores', 'utils']
  },

  components: [
    { path: '~/components', pathPrefix: false }
  ],

  modules: [
    '@nuxt/ui',
    '@pinia/nuxt',
    '@vueuse/nuxt'
  ],

  css: ['~/assets/css/main.css'],

  // Backend defaults — overridden by NUXT_PUBLIC_API_BASE / NUXT_PUBLIC_WS_BASE at runtime.
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080',
      wsBase: process.env.NUXT_PUBLIC_WS_BASE || 'ws://localhost:8080'
    }
  },

  // Dev: forward /api/* and /ws/* to the Go backend on :8080 (matches the old Vite proxy).
  // Prod: server/routes/api/[...path].ts proxies /api/*, server/routes/ws.ts handles /ws.
  nitro: {
    experimental: {
      websocket: true
    },
    devProxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true
      }
    }
  },

  // SSR is on by default; SPA-only would be ssr: false. The plan calls for SSR.
  ssr: true,

  typescript: {
    strict: true
  },

  experimental: {
    payloadExtraction: true
  }
})
