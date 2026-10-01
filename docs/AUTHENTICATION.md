# Authentication

Agent Shaker uses **API-key authentication** (a shared-secret bearer scheme).
It is off by default so local development needs no credentials, and it must be
switched on for any deployment reachable from a network you do not control.

> **Read this before exposing the app.** The API-key layer protects the Go
> service. It is *not* user authentication: there are no accounts, no roles and
> no per-user identity. See [What this does not protect](#what-this-does-not-protect).

## Enabling it

```bash
cp .env.example .env
```

Generate a key:

```bash
openssl rand -hex 32
```

Then in `.env`:

```dotenv
AUTH_ENABLED=true
API_KEYS=8f14e45fceea167a5a36dedd4bea2543...   # one or more, comma-separated
NUXT_API_KEY=8f14e45fceea167a5a36dedd4bea2543...  # same value; the SPA proxy uses it
```

```bash
docker compose up -d
```

The server **refuses to boot** (exit 2) if `AUTH_ENABLED=true` while `API_KEYS`
is empty. Failing at deploy time is the point: a server that starts "protected"
with no keys would either reject every caller or, far worse, accept every
caller.

## How a credential is presented

| Surface | Accepted |
| --- | --- |
| REST (`/api/…`) | `X-API-Key: <key>` or `Authorization: Bearer <key>` |
| MCP (`/`, `/mcp`, `/mcp/message`) | same |
| A2A (`/a2a/v1/…`) | same |
| Agent card (`/.well-known/agent-card.json`) | same |
| WebSocket (`/ws`) | the same headers, **or** `?api_key=<key>` |
| Probes (`/healthz`, `/readyz`, `/metrics`, `/health`) | none — always open |

A bare token in `Authorization` with no scheme is also accepted, because some
MCP clients send it that way.

The A2A agent card at `/.well-known/agent-card.json` advertises the scheme that
is actually in force — `apiKey` when `AUTH_ENABLED=true`, `none` otherwise. It
used to hardcode `none`, which was a straight lie once this layer landed: a
conforming client would send no credential and collect a `401` on every call.

The card also advertises the origin it was reached on, taken from
`X-Forwarded-Proto` / `X-Forwarded-Host` rather than a hardcoded internal
address, so a peer agent is pointed at the public URL instead of a dead one.

### Why `/ws` is the only route that reads a query parameter

A browser cannot attach headers to a WebSocket handshake. For every other route
a credential in the query string would be written to access logs, proxy logs and
browser history, so it is rejected there. On `/ws` it is a fallback only.

## Probes stay unauthenticated on purpose

`/healthz`, `/readyz` and `/metrics` are open. An orchestrator or load balancer
has no credential to present, and a `401` there becomes a crash-restart loop
that looks like an outage rather than the configuration error it is.

`/metrics` in particular should not be public on a real deployment — either
bind it to an internal network or gate it at the reverse proxy. The bundled
Caddyfile does the latter by routing it nowhere at all: the catch-all hands it
to Nuxt, which returns `404`. Scrape `mcp-server:8080/metrics` on the internal
network instead.

## What this does not protect

This is the part that matters for a public launch.

**The web UI is not user authentication.** The Nuxt service holds `NUXT_API_KEY`
server-side and injects it when proxying to the Go service, so the browser never
sees the key and the SPA works normally. The direct consequence:

> Anyone who can load the web UI can use the API through it.

The key therefore guards the Go service's own HTTP surface — which is the
component that owns the data, and which anything that reaches the internal
network can hit directly. It is not a login.

For a single-team or self-hosted tool that is usually the right trade: one
shared secret, no user accounts to provision, and the perimeter sits at the
reverse proxy in front of the web service.

**If you need real access control**, put it in front of `web`, not in the app:

- Caddy or nginx `basic_auth` — simplest, adequate for a small team
- A VPN (Tailscale, WireGuard) — nothing is exposed to the public internet
- An authenticating proxy (oauth2-proxy, Cloudflare Access, Traefik forward-auth)

All three compose with the API key without changes. Keep `AUTH_ENABLED=true` as
defence in depth so the Go service is not usable even if the perimeter is
misconfigured.

The bundled Caddy edge (`make edge-up`) is the first option, pre-wired. See
[DEPLOYMENT.md](./DEPLOYMENT.md) for the topology and how it splits the two
kinds of caller.

## The two-surface model

With the TLS edge in front, the public origin answers two different kinds of
caller, and each path takes exactly one credential:

| Path | For | Credential |
| --- | --- | --- |
| `/`, `/api/*`, `/ws` | browsers, the SPA | basic auth (the key is injected server-side) |
| `/mcp`, `/mcp/*`, `/a2a/*`, `/.well-known/*` | agents, MCP hosts, scripts | API key |
| `/metrics` | nobody — 404 at the edge | — |

**`/api/*` is a UI surface, not a machine one.** A headless client with a valid
API key still gets a `401` there, because the edge requires basic auth on that
path and an HTTP client cannot send `Authorization: Basic` and
`Authorization: Bearer` in the same header. Integrations use `/mcp` or
`/a2a/v1/*`.

The machine branch deliberately has no basic auth, for the same reason: MCP
clients authenticate with `Bearer`, and requiring both would make the surface
unusable rather than more secure. Those paths are still guarded by
`RequireAPIKey`, so every path requires a credential and none requires two
mutually exclusive ones.

### The edge's Basic credential never reaches the Go service

Behind basic auth, every browser request arrives at the Nitro proxy carrying
`Authorization: Basic …`. The proxy **drops** it and forwards only `X-API-Key`
or `Authorization: Bearer`, injecting `NUXT_API_KEY` when the caller presented
neither. See `web/server/utils/upstreamAuth.ts`.

Two things this prevents. The Go service would otherwise receive a Basic
credential where it expects a key and answer `401` to every request; and the
perimeter password would be written into the Go service's request log on every
call, which is strictly worse than the outage.


## Rotating a key

`API_KEYS` accepts a list, and any one entry grants access. Rotation is
therefore additive and needs no downtime:

1. Add the new key alongside the old one and redeploy.
2. Move clients to the new key.
3. Remove the old key and redeploy.

Never edit a key in place on a running container — restart with a different
`API_KEYS` than a client holds and that client is locked out until it is fixed.

## Implementation notes

- **Constant-time comparison.** `internal/middleware/auth.go` compares the
  presented value against every configured key and OR-s the results, without
  short-circuiting on a match. An early return would leak through response
  timing how many leading characters were correct.
- **No echo.** A `401` never reflects the presented value, so the endpoint
  cannot be used as an oracle for guessing keys character by character.
- **Origin and credential are separate controls.** Two independent checks apply
  to a WebSocket, and both are needed:
  - The Go service checks the `Origin` header against `WS_ALLOWED_ORIGINS`. This
    alone would be insufficient here: the Nitro proxy dials upstream
    server-to-server with no `Origin`, which the Go handler treats as
    same-origin.
  - The Nitro proxy therefore checks the *browser-facing* `Origin` against
    `NUXT_WS_ALLOWED_ORIGINS` (empty = same-origin only). This is what stops
    cross-site WebSocket hijacking, which the credential injection would
    otherwise make possible — a page on any site could open
    `ws://<this-host>/ws?project_id=…` and receive another user's events.

  A valid key bypasses neither.
- **Logs never contain the key.** The startup line reports only a key count and
  a four-character fingerprint (`DescribeAuthConfig`); rejections log the path,
  method and client IP, never the credential.
- **Rate limiting** still applies to authenticated requests, keyed by client IP.
  Excluded paths are unchanged: `/ws`, `/healthz`, `/readyz`, `/metrics`.

  Behind the bundled edge, `TRUSTED_PROXY=true` makes the key come from
  `X-Forwarded-For` — otherwise every caller would share the proxy container's
  IP and one client could throttle the rest. The **rightmost** entry is read, not
  the leftmost, because proxies append: a client that forges the header gets
  entries that are ignored rather than a bucket of its own to rotate. See
  [DEPLOYMENT.md](./DEPLOYMENT.md#rate-limiting-behind-the-edge).

## Verifying it is actually enforced

```bash
# No credential -> 401
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:3000/api/projects
# -> 401

# Wrong credential -> 401
curl -s -o /dev/null -w '%{http_code}\n' -H 'X-API-Key: wrong' http://localhost:3000/api/projects
# -> 401

# Valid credential -> 200
curl -s -o /dev/null -w '%{http_code}\n' -H "X-API-Key: $API_KEY" http://localhost:3000/api/projects
# -> 200

# Probes remain open
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:3000/healthz
# -> 200
```

Through the web UI (which injects the key for you) the same endpoints return
200 — if a page starts returning `401` after enabling auth, `NUXT_API_KEY` is
missing or does not match any entry in `API_KEYS`.
