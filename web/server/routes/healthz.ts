import { defineEventHandler } from 'nuxt/server'

/**
 * Liveness probe for the Nuxt service.
 *
 * Deliberately does NOT touch the Go backend: orchestrators use this to decide
 * whether to restart the web container. Backend health is a separate concern
 * and is covered by mcp-server's own HEALTHCHECK, so coupling the two would
 * make a transient database blip restart the frontend too.
 *
 * `defineEventHandler` is imported from `nuxt/server` rather than relying on
 * Nitro's auto-imports — see server/routes/api/[...path].ts for why.
 */
export default defineEventHandler(() => {
  return { status: 'ok', service: 'web' }
})
