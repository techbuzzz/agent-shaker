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

  // Backend wiring.
  runtimeConfig: {
    // Server-only. Consumed by server/routes/api/[...path].ts to forward /api/*
    // to the Go backend. Deliberately NOT under `public`, so the internal
    // hostname (e.g. http://mcp-server:8080) is never shipped to the browser.
    // Set at deploy time via NUXT_API_UPSTREAM.
    apiUpstream: process.env.NUXT_API_UPSTREAM || 'http://127.0.0.1:8080',

    // Server-only API key injected into upstream requests when the caller did
    // not supply one of their own. This is what lets the SPA reach an
    // authenticated Go service without the key ever entering the browser
    // bundle. Server-only for the same reason as apiUpstream: a key under
    // `public` would be readable by anyone who loads the page.
    apiKey: process.env.NUXT_API_KEY || '',

    // Server-only origin allow-list for the /ws handshake, enforced in
    // server/routes/ws.ts. Comma-separated. Empty means same-origin only.
    //
    // This exists because the proxy injects the API key for every caller: the
    // Go-side Origin check is bypassed for browser traffic (the upstream dial
    // carries no Origin), so without this the endpoint would be open to
    // cross-site WebSocket hijacking.
    wsAllowedOrigins: process.env.NUXT_WS_ALLOWED_ORIGINS || '',

    public: {
      // Leave EMPTY in production. An empty value makes useServerUrl resolve a
      // same-origin `/api` and `/ws`, both served by the Nitro proxy. That
      // keeps the deployment on one origin — no CORS, no cross-origin cookies.
      // Dev does not need these either: nitro.devProxy below already forwards
      // to localhost:8080.
      apiBase: process.env.NUXT_PUBLIC_API_BASE || '',
      wsBase: process.env.NUXT_PUBLIC_WS_BASE || ''
    }
  },

  // Dev: forward /api/* and /ws/* to the Go backend on :8080 (matches the old Vite proxy).
  // Prod: server/routes/api/[...path].ts proxies /api/*, server/routes/ws.ts handles /ws.
  nitro: {
    // `pinia` must be INLINED, not left external.
    //
    // @pinia/nuxt pushes its own runtime dir into `build.transpile`, so its
    // plugin (which calls createPinia) is bundled into the server entry. App
    // code importing `defineStore` would otherwise be traced as an external
    // and resolved from `.output/server/node_modules/pinia` — two copies of
    // the module, two distinct `piniaSymbol`s. `useStore()` then injects the
    // bundled copy's symbol while the app's copy looks it up, gets `null`,
    // and throws `getActivePinia() was called but there was no active Pinia`
    // on every SSR render that touches a store. Vite's dev-time dedupe hides
    // this completely: dev renders fine, production returns 500.
    externals: {
      inline: ['pinia']
    },
    experimental: {
      // Nitro v2 / h3 v1. Nuxt 5 moves to Nitro v3, where this becomes
      // `features: { websocket: true }` and `defineWebSocketHandler` is imported
      // from `nitro` rather than `h3` — see server/routes/ws.ts.
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
