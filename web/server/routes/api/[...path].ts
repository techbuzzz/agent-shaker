import { send } from 'h3'

/**
 * Server-side reverse proxy for the Go MCP backend.
 *
 * Why this exists: the browser bundle resolves `apiBase` from the
 * `mcp-server-url` cookie. When that points at a *different* origin than the
 * page, the browser calls the Go server directly and needs a matching CORS
 * allow-list entry on both sides. Production does not want that coupling — one
 * public origin, no CORS negotiation, no cookies to a second host.
 *
 * So the production default is an empty server URL, which makes `apiBase`
 * resolve to `/api` and land here, where Nitro forwards to the Go service over
 * the internal network. `devProxy` in nuxt.config.ts does the same thing during
 * `nuxt dev`, so the application code is identical in both modes.
 *
 * The upstream is read from server-only runtimeConfig (NUXT_API_UPSTREAM),
 * deliberately NOT under `public`, so the internal hostname is never serialised
 * into the browser bundle.
 */
export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const upstream = (config.apiUpstream || 'http://127.0.0.1:8080').replace(/\/+$/, '')
  const path = getRouterParam(event, 'path') ?? ''
  const target = `${upstream}/api/${path}`
  const search = getRequestURL(event).search

  // Propagate the request id so one trace id spans the Nuxt hop and the Go
  // hop: the Go middleware reads the same header, so both services' logs and
  // spans correlate on it.
  const headers: Record<string, string> = {}
  const requestId = getHeader(event, 'x-request-id')
  if (requestId) headers['x-request-id'] = requestId

  // Forward the caller's credential if it supplied one, otherwise fall back to
  // the server-side key.
  //
  // Both paths matter. A headless client (curl, an MCP host, a script) sends
  // its own key and it must survive the hop. The SPA does not: it holds no
  // key at all, so the injected NUXT_API_KEY is what lets the browser reach the
  // API. That key is server-only runtimeConfig, so it is never serialised into
  // the client bundle.
  //
  // Note the consequence: anyone who can load the UI can use the API through
  // it. The key's job is to protect the Go service's own surface (it is
  // reachable by anything that gets past the reverse proxy in front of Nuxt),
  // not to be the app's user authentication. See docs/AUTHENTICATION.md.
  const incomingAuth = getHeader(event, 'authorization')
  const incomingKey = getHeader(event, 'x-api-key')
  if (incomingAuth) {
    headers['authorization'] = incomingAuth
  } else if (incomingKey) {
    headers['x-api-key'] = incomingKey
  } else if (config.apiKey) {
    headers['x-api-key'] = config.apiKey
  }

  // Forward the body only for methods that can carry one. GET/HEAD/DELETE are
  // passed through without a body to avoid Content-Length mismatches.
  const body =
    event.method === 'GET' || event.method === 'HEAD' || event.method === 'DELETE'
      ? undefined
      : await readRawBody(event, false)

  let upstream_: Response
  try {
    upstream_ = await fetch(target + search, {
      method: event.method,
      headers,
      body: body as BodyInit | undefined,
    })
  } catch (err) {
    // Upstream unreachable: return a structured 502 so the SPA's error
    // handling reports a connection problem instead of a silently empty body.
    throw createError({
      statusCode: 502,
      statusMessage: 'Bad Gateway',
      message: `MCP backend unreachable at ${upstream}: ${(err as Error).message}`,
    })
  }

  // Pass the upstream status and payload through unchanged — the Go service
  // owns auth, CORS, body limits and the error envelope.
  setResponseStatus(event, upstream_.status)
  const contentType = upstream_.headers.get('content-type')
  if (contentType) setResponseHeader(event, 'content-type', contentType)

  // `send` is required: returning the ArrayBuffer directly makes Nitro treat
  // it as a payload value and JSON-stringify it, which turns every upstream
  // response into the literal `{}`. Writing the bytes as the response body
  // keeps the upstream JSON byte-for-byte intact.
  return send(event, Buffer.from(await upstream_.arrayBuffer()))
})
