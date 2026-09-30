/**
 * Global middleware: if the server-URL cookie is unset, route to /settings
 * for first-time visitors. The cookie has a default value, so this only
 * fires if the user has explicitly cleared it (via Settings → Reset).
 */
export default defineNuxtRouteMiddleware((to) => {
  const { url } = useServerUrl()
  const cookie = useCookie<string | null>('mcp-server-url', { default: () => null })
  if (!cookie.value && to.path !== '/settings') {
    // Set the default so subsequent navigations don't loop
    cookie.value = url.value
  }
})
