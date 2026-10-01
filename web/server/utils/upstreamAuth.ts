/**
 * Credential selection for requests the Nitro proxy makes to the Go service.
 *
 * Why this is a module and not three lines in each route: getting it wrong is
 * a security bug, not just an auth failure, and the two routes that need it
 * (`/api/*` and `/ws`) failed in exactly the same way.
 *
 * The bug it fixes. Behind a TLS edge, the browser authenticates to the *edge*
 * with HTTP basic auth. That credential arrives at Nitro as an `Authorization:
 * Basic …` header. Both proxy routes used to forward it upstream verbatim, with
 * the result that:
 *
 *   * The Go service received a Basic credential where it expected an API key,
 *     and answered 401 — the SPA was broken for every authenticated user.
 *   * The edge's shared password was shipped into the Go service on every
 *     request, where it would be captured by request logging. Leaking one
 *     deployment's perimeter credential into an internal service's log stream
 *     is strictly worse than the outage it caused.
 *
 * So: only a credential that is genuinely an API key is ever forwarded. A
 * `Basic` (or `Digest`, or any other scheme) credential belongs to the edge and
 * is dropped here — Nitro is behind the edge, so it is already past that check.
 */

export interface IncomingCredentials {
  /** Raw `Authorization` header value, if any. */
  authorization?: string | null
  /** Raw `X-API-Key` header value, if any. */
  apiKey?: string | null
}

const BEARER = /^bearer\s+(\S.*)$/i

/**
 * Decide which credential to present to the Go service.
 *
 * Precedence:
 *   1. `X-API-Key` — unambiguous, always an API key.
 *   2. `Authorization: Bearer <token>` — the form MCP clients conventionally
 *      use, and one the Go service accepts unchanged.
 *   3. The server-only `NUXT_API_KEY`, which is what lets the browser reach an
 *      authenticated API without ever holding a key.
 *
 * Anything else — no credential, or a non-Bearer scheme — falls through to (3).
 * The result is normalised to `x-api-key` so the Go service sees one shape
 * from this proxy rather than two.
 *
 * @param incoming    credentials presented by the caller, if any
 * @param configured  the server-only key from runtimeConfig; may be empty
 */
export function resolveUpstreamAuth(
  incoming: IncomingCredentials,
  configured: string,
): Record<string, string> {
  const apiKey = incoming.apiKey?.trim()
  if (apiKey) return { 'x-api-key': apiKey }

  const authz = incoming.authorization?.trim()
  if (authz) {
    const bearer = BEARER.exec(authz)
    // Capture only the token: the rest of the header is the edge's business.
    if (bearer && bearer[1]!.trim()) return { 'x-api-key': bearer[1]!.trim() }
    // Any other scheme (Basic, Digest, Negotiate, …) is deliberately dropped.
  }

  const serverKey = configured?.trim()
  if (serverKey) return { 'x-api-key': serverKey }

  return {}
}
