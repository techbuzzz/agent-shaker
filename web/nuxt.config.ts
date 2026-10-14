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
        // SVG rather than a .ico: public/ ships no favicon.ico, so the previous
        // declaration 404ed on every page load and left the tab icon blank.
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }
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

  // Disable route prerendering. Crawling at build time snapshots whatever the
  // database state happens to be then (typically empty) and serves that stale
  // HTML until the next rebuild — wrong for an app whose entire UI is
  // fetched live from the API. SSR at request time is correct here.
  routeRules: {
    '/**': {
      prerender: false,
      // Response security headers.
      //
      // This is the server that renders HTML in every supported topology, so
      // the CSP belongs here rather than on the Go API — a policy attached to
      // the API would protect no document the browser actually renders.
      //
      // script-src carries 'unsafe-inline' because Nuxt inlines two executable
      // scripts into the SSR output: the runtime config (window.__NUXT__.config)
      // and the colour-mode bootstrap. Verified against real build output, not
      // assumed — a strict 'self' policy breaks rendering. The directives that
      // actually stop an injected payload (object-src, frame-ancestors,
      // base-uri, form-action) need no such allowance and are enforced.
      //
      // Kept byte-identical to internal/middleware.DefaultSPA_CSP so the Go SPA
      // fallback and this server cannot disagree.
      headers: {
        'Content-Security-Policy': [
          "default-src 'self'",
          "script-src 'self' 'unsafe-inline'",
          "style-src 'self' 'unsafe-inline'",
          "img-src 'self' data:",
          "font-src 'self' data:",
          "connect-src 'self' ws: wss:",
          "object-src 'none'",
          "base-uri 'self'",
          "frame-ancestors 'none'",
          "form-action 'self'"
        ].join('; '),
        // These must be repeated here, not just on the Go service. The Go
        // middleware sets them on API responses, but the document the browser
        // actually renders is served by Nuxt — and a header that never reaches
        // that response protects nothing. Values match
        // internal/middleware.SecurityHeaders exactly; change both together.
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'DENY',
        'Referrer-Policy': 'no-referrer',
        'Permissions-Policy': 'camera=(), microphone=(), geolocation=()'
      }
    }
  },

  typescript: {
    strict: true
  },

  experimental: {
    payloadExtraction: true
  }
})
