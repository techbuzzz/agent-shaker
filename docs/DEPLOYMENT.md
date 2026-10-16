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
| default (`docker compose up -d`) | `web` on `${WEB_BIND}:${WEB_PORT}`, Go on `${MCP_BIND}:${MCP_PORT}` | none | reachable on loopback at `/mcp`, `/a2a/v1/*` |
| `tls` (`docker compose --profile tls up -d`) | Caddy on `:443` / `:80` | automatic Let's Encrypt | reachable from the LAN at `/mcp`, `/a2a/v1/*` |

**The machine surfaces are reachable without an edge, on loopback.** The Go
service publishes `MCP_BIND:MCP_PORT` (default `127.0.0.1:8080`), so an agent on
the same machine connects directly and there is nothing to configure. What the
edge adds is reachability from *other* machines, with TLS in front.

Set `MCP_BIND=0.0.0.0` to expose it on the LAN without the edge — on a trusted
network only, because the endpoint has no TLS of its own. Under the `tls` profile
keep it on loopback and let Caddy be the sole entrypoint.

## Quick start (TLS)

```bash
make deploy HOST=agent-shaker.example.com
```

That is the whole procedure. `scripts/deploy.sh` checks the ports, resolves
secrets (prompting, or generating with `GENERATE=1`), builds, starts, waits for
the certificate, and then verifies the live public origin — the agent card's
advertised URL, an MCP `initialize`, that `/metrics` is 404, that the human
surface rejects an unauthenticated request, and that the WebSocket route is
present rather than silently falling through to the SPA shell. It installs a
daily backup timer and prints the URLs.

Read [`scripts/deploy.sh`](../scripts/deploy.sh) before running it if you want to
know exactly what it touches. The steps it performs are, equivalently:

```bash
# 1. Required configuration
cp .env.example .env
$EDITOR .env          # POSTGRES_PASSWORD, API_KEYS, PUBLIC_HOST, ACME_EMAIL

# 1b. PUBLIC_HOST must be the real domain, not empty and not localhost.
#     Both compose and the Caddyfile default it to localhost, and Caddy issues
#     no certificate for localhost: it serves plain HTTP, reports itself
#     healthy, and puts your basic-auth password on the wire in cleartext.
#     Nothing errors and nothing appears in any log.
grep '^PUBLIC_HOST=' .env

# 2. Export the basic-auth credential (see the warning below for why it is
#    not in .env)
export BASIC_AUTH_USER=admin
export BASIC_AUTH_HASH="$(docker run --rm caddy:2-alpine caddy hash-password --plaintext 'your-password')"

# 3. Bring it up — `make edge-up` is `docker compose --profile tls up -d --build`
make edge-up

# 4. Watch it converge
docker compose --profile tls ps
docker compose --profile tls logs -f edge
```

The first run starts with an **empty** database — migrations create the schema
and nothing else. Sample projects are not seeded; load them deliberately with
`scripts/seed_demo_data.sql` if you want them. See
[MIGRATIONS.md](./MIGRATIONS.md#demo-data-is-not-a-migration).

Validate before you deploy:

```bash
make docker-config     # both compose profiles
make caddy-validate    # the Caddyfile parses
make caddy-fmt-check   # the Caddyfile is in canonical form
```

CI runs all three on every push.

Generate the basic-auth hash once:

```bash
docker run --rm caddy:2-alpine caddy hash-password --plaintext 'your-password'
```

**See "Do not put the bcrypt hash in `.env`" below before you fill it in.**

## What the deploy script decides for you

Three choices are worth stating, because each one exists to avoid a failure that
looks like something else.

**`BASIC_AUTH_HASH` never touches `.env`.** The hash is generated inside the
script, held in a shell variable, and exported for the `docker compose` call
only. Shell variables take precedence over `.env`, so this composes with a
populated `.env` without the interpolation corruption described below.

**Generated passwords are hex.** `POSTGRES_PASSWORD` is interpolated into
`postgres://user:PASSWORD@postgres:5432/db`. An alphabet containing `/`, `@`,
`:` or `#` produces a DSN that parses as a different host or a different
database, and that surfaces as a connection error rather than as a bad password.
`openssl rand -hex 32` cannot produce any of them. `API_KEYS` travels in a
header and has no such constraint, but hex keeps it comma-safe for rotation.

**The basic-auth password is prompted for, not read from `.env`.** It is a
login, so it should be one the operator chose. It is typed without echo, and
cleared from memory by an exit trap. Only the hash survives, and only in the
process environment.

Generate everything non-interactively on a throwaway host with `GENERATE=1`;
generated secrets are printed exactly once, because there is no recovery path
for a lost API key.

## Do not put the bcrypt hash in `.env`

`BASIC_AUTH_USER` and `BASIC_AUTH_HASH` are **required for the edge to boot**.
The `edge` service refuses to start without them, exiting 1 with an actionable
message. This is deliberate: a Caddyfile cannot branch on whether an environment
variable is set, so the only alternative is a placeholder credential, and a
guessable one is worse than a refusal to boot.

The check lives in the `edge` service's command rather than in compose's `:?`
syntax. Compose interpolates the whole file regardless of which profiles are
active, so `:?` would make these variables mandatory for a plain
`docker compose up -d` — forcing every developer without a TLS edge to invent a
bcrypt hash. Guarding at startup keeps the requirement scoped to the profile
that needs it.

```
FATAL: the tls profile requires BASIC_AUTH_USER and BASIC_AUTH_HASH.
  The API key is not a login, so the edge is the only access
  control in this topology. Generate a hash with:
    docker run --rm caddy:2-alpine caddy hash-password --plaintext 'your-password'
  ...
```

### Why interpolation destroys it

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

## Rate limiting behind the edge

`mcp-server` sets `TRUSTED_PROXY=true`, and the reason is specific to this
topology.

Without it, every request reaches the Go service from the **Caddy container's
own IP**. The "per-IP" buckets collapse into one, so `RATE_LIMIT_RPS=100`
becomes a cap for the entire deployment rather than for each caller — a single
noisy client throttles everyone, including the UI.

The flag is safe on loopback: the published port binds `127.0.0.1`, so the only
processes that can reach it are the ones already on the machine, and they are
trusted to the same degree as the container network was.

Two things it is no longer safe for. The port used to be internal-only, so a
request provably arrived through a proxy; now a local caller can send
`X-Forwarded-For` directly and be believed, which is fine at loopback and is not
fine once `MCP_BIND=0.0.0.0`. And for a bare binary on a public host, leave it
unset — an arbitrary remote caller would be able to forge its own client IP and
dodge the per-IP limit.

### Why the rightmost `X-Forwarded-For` entry, not the leftmost

This is the part worth copying if you put another proxy in front. Reverse
proxies **append** to `X-Forwarded-For`. A client that sends its own header:

```
X-Forwarded-For: 1.2.3.4
```

comes out of Caddy as:

```
X-Forwarded-For: 1.2.3.4, <the address Caddy actually saw>
```

Reading the **leftmost** entry hands the caller their own bucket identity.
Rotating a forged value per request then defeats rate limiting completely with
one header — the opposite of the point of having it.

The service reads the **rightmost** entry, the one the trusted proxy itself
appended, i.e. the peer it actually observed. The port is stripped as well, so
varying the source port cannot mint buckets either.

Verified through the running edge: 600 concurrent requests carrying a rotating
forged `X-Forwarded-For` are throttled into the same bucket the honest requests
use, not into fresh ones.

### What the access log shows

The same resolution is applied to logging, and deliberately shares one instance
across every route so the two agree. An access log whose `remote` field is the
proxy's address for every line is the one thing an access log cannot be.

Each line carries both:

- `remote` — the client, as the rate limiter sees it. This is the field to
  group by.
- `peer` — the socket address. It is the proxy's in this topology, and keeping
  it means a misconfiguration is diagnosable: if `remote` ever stops varying
  while `peer` does, `X-Forwarded-For` is not arriving and the edge is not
  forwarding it.

Joining a `429` to the request that caused it is `remote`, and that join works
only because both subsystems resolve the same value.

## Backups

A live deployment holds real projects, agents, tasks and contexts, and
`docker compose down -v` deletes all of it. These targets are the supported
way in and out of that state.

```bash
make db-backup                      # -> backups/agent-shaker-<ts>.dump + .sha256
make db-backups                     # list, newest first
make db-restore FILE=backups/<name>.dump
```

The dump is written to the **host**, never into the container. A file on the
postgres volume dies with the volume it was meant to protect you from.

`backups/` is git-ignored, and it must stay that way. A dump is the entire
dataset; `-Fc` is compression, not encryption.

### `db-restore` replaces, it does not merge

It drops and recreates the public schema, then replays the dump. Everything
created since the dump is gone. The target prints what it is about to do and
sleeps three seconds so an accidental invocation can be interrupted.

### Why the checksum is compared by hand

Each backup writes a `.sha256` alongside the dump, and `db-restore` refuses to
proceed on a mismatch. The comparison is done by extracting the recorded hash
and hashing the file being restored — deliberately **not** `sha256sum -c`.

`sha256sum -c` verifies whatever *filename* the checksum file names, not the
file you passed in. Move or rename a dump and it either fails confusingly or,
worse, cheerfully verifies a different intact file while the corrupt one you
actually asked to restore sails through.

The case that matters is invisible: a corrupt dump of the same length as the
original, with a `.sha256` naming some other intact file. A size check misses
it, `sha256sum -c` passes it, and the restore destroys the live database with
garbage. The direct comparison catches it.

Verified: same-length byte corruption and a truncated dump are both refused; an
intact dump passes; and a full backup → delete → restore cycle brings the data
back with the app healthy.

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
| `TLS_TERMINATED` | `false` | override only — HSTS is per-request. See below |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | usually irrelevant; single origin |
| `CORS_ALLOW_CREDENTIALS` | `false` | decoupled from `AUTH_ENABLED` on purpose |

### HSTS needs no configuration

HSTS is decided **per request**, from `r.TLS` or `X-Forwarded-Proto`. Caddy sets
the latter, so enabling the `tls` profile gives you HSTS with no extra step.

This used to require `TLS_TERMINATED=true` set by hand alongside the profile.
Because compose cannot set an env var based on whether a profile is active,
that made it a second flag to remember — and omitting it was a *silent*
downgrade: the service started healthy, served traffic, and never sent the
header. That failure shape is now unreachable through configuration.

Set `TLS_TERMINATED=true` only if a fronting proxy does not set
`X-Forwarded-Proto` (a raw TCP passthrough, or a hand-rolled one). It does not
enable TLS; it only tells the API to assume the client-facing hop is encrypted.

A client that forges `X-Forwarded-Proto: https` on a cleartext request gains
nothing: per RFC 6797 a browser only honours HSTS received over a secure
transport, so the header is inert and cannot be used to downgrade anything.

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

### Two checks that need a credential, not a `curl -o /dev/null`

`/metrics` and `/ws` sit on the edge's human branch, so the basic-auth gate
answers **before the routing table is consulted**. Unauthenticated they return
`401` — the gate — regardless of whether the route exists. Probing them without
a credential therefore tells you nothing, and asserting `404` there fails on a
perfectly healthy deployment:

```bash
# 404 — not routed, so metrics are not public. Needs the credential.
curl -s -o /dev/null -w '%{http_code}\n' -u "$UI_USER:$UI_PASS" https://your.host/metrics

# 426 — the route exists and correctly refuses a non-upgrade GET.
# 200 instead would mean no route matched and the SPA shell was served,
# which breaks live updates without ever raising an error.
curl -s -o /dev/null -w '%{http_code}\n' -u "$UI_USER:$UI_PASS" https://your.host/ws
```

`scripts/deploy.sh` asserts exactly these values; they are measured, not assumed.

Confirm the agent card advertises your **public** origin, not the container's:

```bash
curl -s -H "Authorization: Bearer $API_KEY" https://your.host/.well-known/agent-card.json | jq .url
# "https://your.host/a2a/v1"
```

## Releasing an image to Docker Hub

`.github/workflows/publish-docker.yml` publishes two images, and **only from a
version tag**:

| image | built from |
|---|---|
| `techbuzzz/agent-shaker-server` | `./Dockerfile` (Go) |
| `techbuzzz/agent-shaker-webui` | `./web/Dockerfile` (Nuxt) |

```bash
git tag v1.2.3
git push origin v1.2.3
```

That single tag push is the release. There is no separate publish step, and
merging to `main` publishes nothing — `ci.yml` already builds both images on
every push and pull request, so a second build on branch pushes would only
repeat work while adding a way to overwrite a published `latest` by accident.

`v1.2.3` publishes three tags per image: `1.2.3`, `1.2` and `latest`. A
prerelease (`v1.2.3-rc1`) publishes **only** `1.2.3-rc1` — `1.2` and `latest`
are left pointing at the last stable build, because those are the tags
consumers pull and an RC silently taking them over is discovered from someone
else's bug report.

Both images are built for `linux/amd64` and `linux/arm64`.

### Before the first release

Two repository settings, neither of which the workflow can create for itself:

- **Secrets** `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` — an access token with
  read/write on the namespace, not the account password. Same names the
  sibling projects use.
- **Environment** `dockerhub-prod`. Enable *Required reviewers* on it if you
  want a human to approve each push: the `v*` tag filter stops an ordinary
  merge from publishing, but it does nothing about a mistyped tag or about
  anyone who can push a `v*` tag, and a `latest` that moves is not undone by
  re-tagging.

### Docker Hub repository overviews

Each repository's "Overview" tab on Docker Hub is rendered from a markdown
file. The publish workflow updates both overviews automatically after a
successful push, reading from:

| repository | overview source |
|---|---|
| `techbuzzz/agent-shaker-server` | `docker/agent-shaker-server.README.md` |
| `techbuzzz/agent-shaker-webui` | `docker/agent-shaker-webui.README.md` |

Edit those files to change what appears on Docker Hub. The update runs as the
last step of the publish workflow and uses the same `DOCKERHUB_TOKEN` as the
push. A failed overview update logs a warning but does **not** fail the
release — the images are already published and verified by then.

### If a publish fails halfway

Re-running is safe: `workflow_dispatch` on the same workflow takes the tag as
input, and `concurrency` cancels an in-flight run for the same tag rather than
queueing behind it. The post-push steps re-verify both manifests for both
platforms, so a half-published tag is caught rather than discovered on pull.

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
