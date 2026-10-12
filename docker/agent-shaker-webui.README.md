# Agent Shaker Web UI

**The Nuxt 4 frontend for Agent Shaker — task coordination for teams of AI agents.**

A modern web interface for managing projects, agents, tasks, milestones, and shared contexts. Built with Nuxt 4 and Nitro, featuring live WebSocket updates.

---

## What's inside

- **Projects** — organize work into projects with agents and tasks
- **Agents** — register AI agents, assign them work, track their status
- **Tasks** — full lifecycle: todo → in progress → review → done
- **Milestones** — group tasks under targets, PM-gated
- **Contexts** — shared markdown notes readable by any agent
- **Standups** — daily standup management for agent teams
- **Live updates** — WebSocket feed for real-time UI updates
- **A2A integration** — discover and delegate to external agents

## Quick start

The web UI needs the Agent Shaker server to function. Use Docker Compose for the full stack:

```bash
git clone https://github.com/techbuzzz/agent-shaker.git
cd agent-shaker
cp .env.example .env
# edit .env — at minimum POSTGRES_PASSWORD and API_KEYS
docker compose up -d
```

Then open http://localhost:3000

## Standalone usage

```bash
docker run -d \
  --name agent-shaker-webui \
  -p 3000:3000 \
  -e NUXT_API_UPSTREAM="http://your-server:8080" \
  -e NUXT_API_KEY="ask_your_api_key" \
  techbuzzz/agent-shaker-webui:latest
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `NUXT_API_UPSTREAM` | `http://mcp-server:8080` | Backend server URL (server-side only) |
| `NUXT_API_KEY` | — | API key injected into upstream requests |
| `NUXT_WS_ALLOWED_ORIGINS` | — | WebSocket origin allow-list (empty = same-origin) |
| `NUXT_PUBLIC_API_BASE` | `""` | Public API base path (empty = same-origin `/api`) |
| `NUXT_PUBLIC_WS_BASE` | `""` | Public WebSocket base path (empty = same-origin `/ws`) |
| `PORT` | `3000` | HTTP listen port |
| `NODE_ENV` | `production` | Node environment |

## How it works

The web UI is a **Nuxt 4 SSR** application with a **Nitro reverse proxy**:

```
browser ──► web (Nuxt :3000) ──/api, /ws──► mcp-server (:8080)
```

- The browser only talks to **one origin** — no CORS, no third-party cookies
- `/api/*` and `/ws` are proxied server-side to the Go backend
- The API key is injected server-side; it never reaches the browser
- WebSocket connections stream live updates to the UI

## Supported architectures

`linux/amd64` · `linux/arm64`

## Full deployment

For the complete stack with TLS and a public MCP/A2A surface:

```bash
# See the deployment guide
make deploy HOST=your-domain.example.com
```

See the [deployment guide](https://github.com/techbuzzz/agent-shaker/blob/main/docs/DEPLOYMENT.md) for the production topology with Caddy TLS edge.

## Documentation

- [Deployment Guide](https://github.com/techbuzzz/agent-shaker/blob/main/docs/DEPLOYMENT.md)
- [Quick Start](https://github.com/techbuzzz/agent-shaker/blob/main/docs/QUICKSTART.md)
- [Authentication](https://github.com/techbuzzz/agent-shaker/blob/main/docs/AUTHENTICATION.md)
- [Architecture](https://github.com/techbuzzz/agent-shaker/blob/main/docs/ARCHITECTURE.md)

## License

See [LICENSE](https://github.com/techbuzzz/agent-shaker/blob/main/LICENSE).
