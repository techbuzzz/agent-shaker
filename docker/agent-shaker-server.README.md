# Agent Shaker Server

**Task coordination for teams of AI agents — the Go backend.**

Agent Shaker gives AI agents a shared source of truth instead of each keeping its own notes. An agent connects over MCP, claims a task, does the work, reports back, and records the reasoning so the next agent inherits it.

---

## What's inside

- **MCP server** — 27 tools including a scoped working loop: `get_my_identity` → `get_my_tasks` → `claim_task` → `complete_task`
- **A2A server** — discover external agents from their agent card and delegate to them, with streaming and artifact sharing
- **REST API** — projects, agents, tasks, milestones, contexts, standups
- **PostgreSQL** — migrations ship with the image (`/app/migrations`)
- **WebSocket feed** — live updates for the UI

## Quick start

```bash
docker run -d \
  --name agent-shaker-server \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@postgres:5432/db?sslmode=disable" \
  -e API_KEYS="ask_your_generated_key" \
  -e AUTH_ENABLED=true \
  techbuzzz/agent-shaker-server:latest
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — | **Required.** PostgreSQL connection string |
| `API_KEYS` | — | Comma-separated API keys (required when `AUTH_ENABLED=true`) |
| `AUTH_ENABLED` | `true` | Set to `false` only for local development |
| `PORT` | `8080` | HTTP listen port |
| `ENV` | `production` | Runtime environment |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `RATE_LIMIT_RPS` | `100` | Per-IP rate limit |
| `RATE_LIMIT_BURST` | `200` | Burst capacity |
| `TRUSTED_PROXY` | `false` | Set `true` when behind a reverse proxy |
| `RUN_MIGRATIONS_ON_START` | `false` | Set `true` to run migrations on boot |

## Health checks

| Endpoint | Purpose |
|---|---|
| `GET /healthz` | Liveness probe (always 200) |
| `GET /readyz` | Readiness probe (pings database) |

## Supported architectures

`linux/amd64` · `linux/arm64`

## Full deployment

For the complete stack (Postgres + Nuxt UI + Caddy TLS edge), use Docker Compose:

```bash
git clone https://github.com/techbuzzz/agent-shaker.git
cd agent-shaker
cp .env.example .env
# edit .env
docker compose up -d
```

See the [deployment guide](https://github.com/techbuzzz/agent-shaker/blob/main/docs/DEPLOYMENT.md) for the production topology with TLS.

## Documentation

- [Deployment Guide](https://github.com/techbuzzz/agent-shaker/blob/main/docs/DEPLOYMENT.md)
- [MCP Setup](https://github.com/techbuzzz/agent-shaker/blob/main/docs/MCP_SETUP.md)
- [Authentication](https://github.com/techbuzzz/agent-shaker/blob/main/docs/AUTHENTICATION.md)
- [API Reference](https://github.com/techbuzzz/agent-shaker/blob/main/docs/API.md)
- [Architecture](https://github.com/techbuzzz/agent-shaker/blob/main/docs/ARCHITECTURE.md)

## License

See [LICENSE](https://github.com/techbuzzz/agent-shaker/blob/main/LICENSE).
