import {
  createError,
  defineEventHandler,
  getRequestHeader,
  getRequestURL,
  getRouterParam,
  useRuntimeConfig
} from 'nuxt/server'
import { resolveUpstreamAuth } from '../../utils/upstreamAuth'

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
 *
 * Imports come from `nuxt/server`, not from `h3` and not from Nitro's
 * auto-imports. On Nuxt 4.6+ the auto-imported names of the same spelling are
 * h3 v1's, and `h3`'s `defineEventHandler` hands the callback a different event
 * than the `nuxt/server` helpers read. The same source keeps working on Nuxt 5,
 * where the server moves to Nitro v3 / h3 v2 and these names come from
 * `nuxt/server` there too.
 */

/**
 * Headers that describe a single hop and must not be relayed verbatim.
 *
 * `content-encoding` / `content-length` are in the list for a mechanical
 * reason: `fetch` already decoded and buffered the upstream body, so the
 * values the upstream advertised no longer describe what we are about to send.
 * Forwarding them would make the client either fail to decode or hang waiting
 * for bytes that never arrive.
 */
const HOP_BY_HOP = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
  'content-encoding',
  'content-length'
])

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const upstream = (config.apiUpstream || 'http://127.0.0.1:8080').replace(/\/+$/, '')

  // `decode: false` keeps the raw percent-encoding: this value is spliced into
  // a URL that is then dialled, so re-encoding a decoded segment could change
  // which upstream path is addressed.
  const path = getRouterParam(event, 'path', { decode: false }) ?? ''
  const search = getRequestURL(event).search
  const target = `${upstream}/api/${path}${search}`

  const headers: Record<string, string> = resolveUpstreamAuth(
    {
      authorization: getRequestHeader(event, 'authorization'),
      apiKey: getRequestHeader(event, 'x-api-key')
    },
    config.apiKey
  )

  // Propagate the request id so one trace id spans the Nuxt hop and the Go
  // hop: the Go middleware reads the same header, so both services' logs and
  // spans correlate on it.
  const requestId = getRequestHeader(event, 'x-request-id')
  if (requestId) headers['x-request-id'] = requestId

  // The body's content type is part of the request, not a credential, and the
  // Go handlers decode JSON. Dropping it made every write depend on the backend
  // ignoring the header.
  const contentType = getRequestHeader(event, 'content-type')
  if (contentType) headers['content-type'] = contentType

  // Forward the caller's API credential if it supplied one, otherwise fall back
  // to the server-side key.
  //
  // Both paths matter. A headless client (curl, an MCP host, a script) sends
  // its own key and it must survive the hop. The SPA does not: it holds no
  // key at all, so the injected NUXT_API_KEY is what lets the browser reach the
  // API. That key is server-only runtimeConfig, so it is never serialised into
  // the client bundle.
  //
  // resolveUpstreamAuth also decides what must NOT be forwarded. Behind a TLS
  // edge every browser request carries `Authorization: Basic …` for the edge's
  // own user gate; forwarding that upstream both broke the API and shipped the
  // perimeter password into the Go service's request log. See the module.
  //
  // Note the consequence: anyone who can load the UI can use the API through
  // it. The key's job is to protect the Go service's own surface (it is
  // reachable by anything that gets past the reverse proxy in front of Nuxt),
  // not to be the app's user authentication. See docs/AUTHENTICATION.md.

  // Read the body only for methods that can carry one. GET/HEAD/DELETE are
  // passed through without a body to avoid Content-Length mismatches.
  const method = event.req.method
  const body = method === 'GET' || method === 'HEAD' || method === 'DELETE'
    ? undefined
    : await event.req.text()

  let upstreamResponse: Response
  try {
    upstreamResponse = await fetch(target, {
      method,
      headers,
      body
    })
  } catch (err) {
    // Upstream unreachable: return a structured 502 so the SPA's error
    // handling reports a connection problem instead of a silently empty body.
    throw createError({
      status: 502,
      statusText: 'Bad Gateway',
      message: `MCP backend unreachable at ${upstream}: ${(err as Error).message}`
    })
  }

  // Pass the upstream status and payload through unchanged — the Go service
  // owns auth, CORS, body limits and the error envelope.
  //
  // Returning the body as a stream (`upstreamResponse.body`) rather than
  // buffering it into an ArrayBuffer keeps memory flat for large collections
  // and stops the proxy from rewriting the payload. The earlier version needed
  // `send()` to avoid Nitro JSON-stringifying the ArrayBuffer into `{}`; a
  // returned `Response` is written as-is, so that workaround is gone.
  const responseHeaders = new Headers()
  upstreamResponse.headers.forEach((value, key) => {
    if (!HOP_BY_HOP.has(key.toLowerCase())) responseHeaders.set(key, value)
  })

  return new Response(upstreamResponse.body, {
    status: upstreamResponse.status,
    statusText: upstreamResponse.statusText,
    headers: responseHeaders
  })
})
