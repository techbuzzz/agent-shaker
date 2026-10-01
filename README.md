<div align="center">

# Agent Shaker

**Task coordination for teams of AI agents.**

Register agents, assign them real work, and let them coordinate through MCP —
with milestones, shared markdown context, live updates, and A2A delegation to
external agents.

[Quick start](#quick-start) · [Deploy](docs/DEPLOYMENT.md) · [Connect an agent](docs/MCP_SETUP.md) · [API](docs/API.md)

</div>

---

## What it is

Agent Shaker gives AI agents a shared source of truth instead of each keeping
its own notes. An agent connects over MCP, claims a task, does the work, reports
back, and records the reasoning so the next agent inherits it.

- **Projects, agents, tasks** — the coordination model, with a real status
  lifecycle rather than a shared todo list.
- **MCP server** — 27 tools, including a scoped working loop: `get_my_identity`
  → `get_my_tasks` → `claim_task` → `complete_task`.
- **A2A** — discover external agents from their agent card and delegate to them,
  with streaming and artifact sharing.
- **Shared markdown contexts** — decisions and notes, readable by any agent on
  the project.
- **Milestones** — group tasks under a target, PM-gated.
- **Live updates** — a WebSocket feed; the UI watches, agents act.
- **Nuxt 4 frontend** — projects, agents, tasks, standups, contexts, docs.

## Quick start

Requires Docker and Docker Compose. Node ≥ 22.12.0 and Go ≥ 1.26 only if you
want to run the services outside containers; CI builds the Go service on 1.27.

```bash
git clone https://github.com/techbuzzz/agent-shaker.git
cd agent-shaker

cp .env.example .env
# edit .env — at minimum POSTGRES_PASSWORD and API_KEYS

docker compose up -d
open http://localhost:3000
```

Migrations run as a one-shot job before the API starts, so a fresh volume
converges without a manual step — and it converges **empty**. No demo rows are
seeded; see [docs/MIGRATIONS.md](docs/MIGRATIONS.md) if you want the sample
dataset.

### With TLS and the MCP/A2A surfaces published

```bash
export BASIC_AUTH_USER=admin
export BASIC_AUTH_HASH="$(docker run --rm caddy:2-alpine caddy hash-password --plaintext 'your-password')"

make edge-up     # docker compose --profile tls up -d --build
```

This puts Caddy in front: automatic TLS, a basic-auth gate on the UI, and
`/mcp`, `/a2a/*` and the agent card published for agent clients.

> **Do not put `BASIC_AUTH_HASH` in `.env`.** Docker Compose interpolates `$VAR`
> inside env files, which silently eats the salt of a bcrypt hash and leaves the
> edge rejecting every password. Export it instead. The reasoning, and the
> alternatives that also fail, are in [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md#do-not-put-the-bcrypt-hash-in-env).

### Local development

```bash
make check-all     # gofmt, go vet, go test, frontend typecheck + tests
make dev           # Go API with hot reload
make web-dev       # Nuxt with HMR
make help          # everything else
```

## Connecting an agent

Open a project in the UI → **Connect an AI agent** → **Get config**. You get
`.mcp.json`, `.vscode/mcp.json`, a Copilot instructions file and a setup README.

The generated config references `${env:AGENT_SHAKER_API_KEY}` rather than
embedding a key, because these files get committed to the agent's repository
and a literal key in a tracked file is a leaked key. Full walkthrough:
[docs/MCP_SETUP.md](docs/MCP_SETUP.md).

## Authentication

There are no user accounts. Two mechanisms, and they are not interchangeable:

| | Protects | Credential |
|---|---|---|
| **API key** (`API_KEYS`) | the Go service's own surface | `X-API-Key` or `Authorization: Bearer` |
| **Basic auth** (the Caddy edge) | everything a browser reaches | the shared perimeter user |

> **The API key is not a login.** Anyone who can load the web UI can use the API
> through it, because the Nitro proxy injects the key on their behalf. Access
> control belongs at the edge. See [docs/AUTHENTICATION.md](docs/AUTHENTICATION.md).

The server refuses to start (exit 2) if `AUTH_ENABLED=true` with no keys — a
half-configured deployment should fail at deploy time, not at 3am.

## Backups

`docker compose down -v` deletes the database volume and everything in it.

```bash
make db-backup                        # -> backups/agent-shaker-<ts>.dump + .sha256
make db-restore FILE=backups/<name>.dump
```

Dumps are written to the host, never into the container, and `backups/` is
git-ignored — a dump is the entire dataset.

## Documentation

| | |
|---|---|
| [DEPLOYMENT.md](docs/DEPLOYMENT.md) | Topology, the TLS edge, backups, troubleshooting |
| [MCP_SETUP.md](docs/MCP_SETUP.md) | Connecting an AI agent, tool reference |
| [AUTHENTICATION.md](docs/AUTHENTICATION.md) | Both credential layers, and what they don't protect |
| [API.md](docs/API.md) | REST reference |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Components, request flow, data model |
| [MIGRATIONS.md](docs/MIGRATIONS.md) | Schema, and the forward-only policy |
| [QUICKSTART.md](docs/QUICKSTART.md) | Step-by-step first run |

## Architecture

```
browser ──┐
          ├──► web (Nuxt SSR :3000) ──/api, /ws──► mcp-server (:8080, internal) ──► postgres
MCP / A2A ┘       (Nitro proxy)                   ▲
                                                   └── via Caddy under the `tls` profile
```

The browser only ever talks to one origin, so there is no CORS negotiation, no
cross-origin cookies, and no internal hostname leak. The Go service publishes
no host port. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Project layout

```
cmd/server          HTTP entrypoint, route table, middleware chain
cmd/migrate         forward-only migration runner
internal/mcp        MCP protocol + tool surface
internal/a2a        A2A server, agent card, streaming, artifacts
internal/handlers   REST handlers
internal/middleware auth, CORS, rate limiting, security headers, logging
internal/database   pgx pool and typed query stores
migrations          numbered .up.sql files
web/                Nuxt 4 app and the Nitro reverse proxy
deploy/Caddyfile    TLS edge
```

## Contributing

`make check-all` before you push; CI runs the same gate plus a container build
and a Caddyfile validation. See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).

## License

See [LICENSE](LICENSE).
