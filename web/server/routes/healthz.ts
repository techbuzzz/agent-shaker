/**
 * Liveness probe for the Nuxt service.
 *
 * Deliberately does NOT touch the Go backend: orchestrators use this to decide
 * whether to restart the web container. Backend health is a separate concern
 * and is covered by mcp-server's own HEALTHCHECK, so coupling the two would
 * make a transient database blip restart the frontend too.
 */
export default defineEventHandler((event) => {
  setResponseStatus(event, 200)
  return { status: 'ok', service: 'web' }
})
