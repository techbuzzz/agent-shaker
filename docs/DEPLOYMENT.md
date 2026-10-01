# Deployment

How Agent Shaker is actually put in front of users, and why it is shaped this
way. For the authentication scheme itself see [AUTHENTICATION.md](./AUTHENTICATION.md).

## Topology

```
                    ┌──────────────────────────── TLS edge ────────────────────────────┐
                    │  Caddy (opt-in, `tls` profile)                                 │
   browser ─────────►│    /            /api/*  /ws       → web:3000  + basic auth    │
                    │    /mcp*  /a2a/*  /.well-known/*  → mcp-server:8080            │
   MCP / A2A client ►│                                     (API key, no basic auth)   │
                    └───────────────────────────────────────────────────────────────┘
                                                    │
                    ┌───────────────────────────────┴─────────────────────┐
                    │  web (Nuxt SSR :3000)  ── Nitro proxy ──┐          │
                    └─────────────────────────────────────────┼──────────┘
                                                              ▼
                                            mcp-server (Go, :8080, internal only)
                                                              │
                                                              ▼
                                                        postgres:16
```

Two public entrypoints exist depending on profile:

| Profile | Public entrypoint | TLS | Machine clients (MCP/A2A) |
|---|---|---|---|
| default (`docker compose up -d`) | `web` on `${WEB_BIND}:${WEB_PORT}` | none — put your own proxy in front | **unreachable** |
| `tls` (`docker compose --profile tls up -d`) | Caddy on `:443` / `:80` | automatic Let's Encrypt | reachable at `/mcp`, `/a2a/v1/*` |

**The machine surfaces are only published under the `tls` profile.** Without an
edge there is no proxy that can route them, and the Go service publishes no host
port. If you need MCP/A2A reachable and you are not using the `tls` profile,
either enable that profile or put your own reverse proxy in front and give
`mcp-server` a port to bind.

## Quick start (TLS)

```bash
# 1. Required configuration
cp .env.example .env
$EDITOR .env          # POSTGRES_PASSWORD, API_KEYS, BASIC_AUTH_USER, BASIC_AUTH_HASH, PUBLIC_HOST

# 2. Bring it up
docker compose --profile tls up -d

# 3. Watch it converge
docker compose --profile tls ps
docker compose --profile tls logs -f edge
```

Generate the basic-auth hash once:

```bash
docker run --rm caddy:2-alpine caddy hash-password --plaintext 'your-password'
```

**See "Do not put the bcrypt hash in `.env`" below before you fill it in.**

## Do not put the bcrypt hash in `.env`

`BASIC_AUTH_USER` and `BASIC_AUTH_HASH` are **required** under the `tls`
profile and compose will refuse to start without them. This is deliberate: a
Caddyfile cannot branch on whether an environment variable is set, so the only
alternative is a placeholder credential, and a guessable one is worse than a
refusal to boot.

### Do not put the bcrypt hash in `.env`

This is the single easiest way to deploy a locked-out edge, so it is worth
stating bluntly.

Docker Compose interpolates `$VAR` **inside env-file values**. A bcrypt hash is
`$2a$14$<salt><digest>`, and the `$<salt>` segment looks exactly like a
variable reference. (Hash below is elided — it is illustrative, not a real one):

```
$2a$14$SALTSEGMENT_THAT_LOOKS_LIKE_A_VAR/dgHtC5A3nqK5OWQ3mEXAMPLEaaKqDbMxP0MKI
                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ treated as ${SALTSEGMENT_...} -> empty

becomes
$2a$14/dgHtC5A3nqK5OWQ3mEXAMPLEaaKqDbMxP0MKI
```

The result is still 60 characters in the wrong places and no longer parses as a
bcrypt hash, so the edge starts, reports healthy, and rejects **every** password
— a silent lockout with no error in any log. Export the value from the shell
instead, where nothing re-interprets it:

```bash
export BASIC_AUTH_USER=admin
export BASIC_AUTH_HASH='$2a$14$...'     # single quotes: literal, no expansion
docker compose --profile tls up -d
```

A shell variable takes precedence over `.env`, so this composes fine with a
populated `.env` file.

Things that do **not** work, all verified against this Compose version:

| Approach | Result |
|---|---|
| `BASIC_AUTH_HASH=$2a$14$...` in `.env` | salt eaten, every password rejected |
| same via `--env-file` | identical corruption |
| same via service-level `env_file:` | identical corruption |
| doubling the `$` (`$$2a$$14$$...`) | still corrupted |
| `export ...='$2a$14$...'` (single-quoted) | **works** |

`API_KEYS` and `POSTGRES_PASSWORD` are unaffected — they contain no `$`. Only
bcrypt hashes hit this.

### `TLS_TERMINATED` is a manual pairing

Compose cannot set one environment variable based on whether a profile is
active, so when you enable the `tls` profile you must also set
`TLS_TERMINATED=true` in `.env`. It does not enable TLS; it tells the Go service
that a proxy in front terminated it, so it emits HSTS. Advertising HSTS without
a real certificate is a lie the browser acts on.

## Why the edge splits traffic by caller

The alternative design is to teach the Nitro proxy to forward MCP and A2A
alongside `/api`, keeping Nuxt as the single upstream for everything. That does
not work:

1. **Path collision.** `/` is a registered MCP endpoint on the Go service (the
   legacy JSON-RPC alias) *and* it is the SPA. Only one can own it, and the SPA
   has to. A Nitro-only design leaves the root endpoint permanently dead.

2. **SSE cannot survive the Nitro proxy.** MCP streamable-HTTP and
   `POST /a2a/v1/message:stream` respond with `text/event-stream`. The proxy
   in `server/routes/api/[...path].ts` buffers the upstream body into an
   `ArrayBuffer` and writes it with `send()` — correct for JSON, fatal for a
   stream, because events only appear once the response closes. Streaming needs
   a genuine pass-through proxy, which is what Caddy's `reverse_proxy` already is.

So the split is by *who is calling*:

| Path | Upstream | Credential | For |
|---|---|---|---|
| `/`, `/api/*`, `/ws`, `/healthz` | `web:3000` | basic auth | browsers, the SPA |
| `/mcp`, `/mcp/*`, `/a2a/*`, `/.well-known/*` | `mcp-server:8080` | API key | agents, MCP hosts, scripts |
| `/metrics` | nothing — 404 | — | nobody; scrape it internally |

**Consequence worth planning around:** under the `tls` profile, `/api/*` is a
*UI* surface, not a machine one. A headless client with a valid API key still
gets a 401 there, because the edge requires basic auth on that path and a client
cannot send `Basic` and `Bearer` in the same `Authorization` header. Integrations
should use `/mcp` or `/a2a/v1/*`. If you genuinely need key-only REST access
from a script, put it on the machine branch in `deploy/Caddyfile`.

### Why the machine branch has no basic auth

MCP clients authenticate with `Authorization: Bearer <key>`. HTTP basic auth
consumes the *same header*. A client physically cannot send both, so requiring
basic auth on the MCP path would make the surface unusable rather than more
secure.

Those paths are not left open: `RequireAPIKey` guards every one of them. The
property the topology actually guarantees is that **every path requires a
credential, and no path requires two mutually exclusive ones.**

### Why `/metrics` is not routed

The catch-all sends it to Nuxt, which returns 404. Prometheus metrics at a
public origin are free reconnaissance, and nothing needs them from outside —
scrape `mcp-server:8080/metrics` on the internal network.

### The edge's basic-auth credential never reaches the Go service

Behind this edge every browser request carries `Authorization: Basic …` for the
edge's own user gate. The Nitro proxy drops it rather than forwarding it: the
Go service expects an API key, and forwarding the Basic credential would both
break the API and write the perimeter password into the Go service's request
log. `web/server/utils/upstreamAuth.ts` is the single place that decision is
made, for both `/api/*` and `/ws`.

## Health checks

| Endpoint | Checked by | Auth |
|---|---|---|
| `/healthz` (web) | `web` container healthcheck | none |
| `http://127.0.0.1:2019/config/` (edge) | `edge` container healthcheck | none |
| `/healthz` (Go) | `mcp-server` container healthcheck | none — probes are exempt |
| `/readyz` (Go) | external orchestrators | none — probes are exempt |

`/healthz`, `/readyz`, `/metrics` and `/health` are deliberately exempt from
API-key auth. An orchestrator has no credential, and a 401 there becomes a
crash-restart loop instead of a legible configuration error.

## A2A discovery

The agent card is served at `/.well-known/agent-card.json` and is reachable
under the `tls` profile.

The URLs it advertises are derived **per request** from `X-Forwarded-Proto` /
`X-Forwarded-Host` (falling back to the request's own scheme and `Host`), so the
card describes the public origin rather than the Go container's internal one. A
hardcoded `http://localhost:8080` would send every A2A client to a dead address.

Set `BASE_URL` to override this — the only case that needs it is an origin that
cannot be derived from a request, such as a separate API hostname.

The card also reports its real auth scheme. It used to hardcode `none`, which
was accurate when every surface was open and became a lie the moment API-key
auth landed: a conforming client would send no credential and collect a 401.

> `X-Forwarded-*` is client-settable, so a caller can influence the advertised
> origin. The responses are generated per request, never cached, and already
> require an API key, so the worst case is a caller poisoning a card it is the
> sole reader of. Set `BASE_URL` to remove even that.

## Environment variables

See [`.env.example`](../.env.example) for the full annotated list. The ones that
govern this topology:

| Variable | Default | Notes |
|---|---|---|
| `POSTGRES_PASSWORD` | — | **required**; compose refuses to start without it |
| `API_KEYS` | — | **required**; comma-separated, rotate by adding before removing |
| `AUTH_ENABLED` | `true` | the Go service exits `2` if true with an empty key set |
| `NUXT_API_KEY` | `${API_KEYS}` | the key Nitro injects; must be a member of `API_KEYS` |
| `BASIC_AUTH_USER` / `BASIC_AUTH_HASH` | — | **required under the `tls` profile** |
| `PUBLIC_HOST` | `localhost` | the hostname Caddy issues a certificate for |
| `ACME_EMAIL` | `admin@example.com` | Let's Encrypt contact |
| `HTTPS_PORT` / `HTTP_PORT` | `443` / `80` | |
| `WEB_BIND` | `127.0.0.1` | loopback-only, so Nuxt is not reachable around the edge |
| `WEB_PORT` | `3000` | |
| `TLS_TERMINATED` | `false` | **set to `true` under the `tls` profile** — see below |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | usually irrelevant; single origin |
| `CORS_ALLOW_CREDENTIALS` | `false` | decoupled from `AUTH_ENABLED` on purpose |

### `TLS_TERMINATED` is a manual pairing

Compose cannot set one environment variable based on whether a profile is
active, so when you enable the `tls` profile you must also set
`TLS_TERMINATED=true` in `.env`. It does not enable TLS; it tells the Go service
that a proxy in front terminated it, so it emits HSTS. Advertising HSTS without
a real certificate is a lie the browser acts on.

## Verifying a deployment

```bash
# Human surface — should be 200
curl -sk -u 'user:pass' https://your.host/healthz
curl -sk -u 'user:pass' https://your.host/api/projects

# Human surface without credentials — should be 401
curl -sk -o /dev/null -w '%{http_code}\n' https://your.host/api/projects

# Machine surface — should be 200 with the key, 401 without
curl -s -H "Authorization: Bearer $API_KEY" https://your.host/.well-known/agent-card.json
curl -s -o /dev/null -w '%{http_code}\n' https://your.host/.well-known/agent-card.json

# MCP initialize round-trip
curl -s -X POST https://your.host/mcp \
  -H "Authorization: Bearer $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'

# A2A
curl -s -H "Authorization: Bearer $API_KEY" https://your.host/a2a/v1/tasks
```

Confirm the agent card advertises your **public** origin, not the container's:

```bash
curl -s -H "Authorization: Bearer $API_KEY" https://your.host/.well-known/agent-card.json | jq .url
# "https://your.host/a2a/v1"
```

## Troubleshooting

**`/api/*` returns 401 in the browser but the key is correct.** The SPA holds no
key — the Nitro proxy injects `NUXT_API_KEY` for it. A 401 means the proxy had
no key to inject, which means the running `web` image predates the `apiKey`
runtimeConfig entry. Rebuild: `docker compose build web && docker compose up -d web`.

**`/.well-known/agent-card.json` returns 404.** You are not behind the `tls`
profile. See "Topology" above.

**Caddy serves plain HTTP when you expected HTTPS.** With `PUBLIC_HOST=localhost`
Caddy issues no certificate, but it still redirects HTTP→HTTPS. Test with
`https://localhost/` and `curl -k`.

**Everything is 502.** `web` depends on `mcp-server: healthy` and the one-shot
`migrate` job must have completed. Check
`docker compose logs migrate` — migrations are forward-only, so a failed
`up` leaves the schema where it stopped.
