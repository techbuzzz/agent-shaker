# Agent Shaker — Web UI (Nuxt 4 + Nuxt UI v3)

Modern Nuxt frontend for the Agent Shaker MCP backend. Replaces the
previous Vite + Vue 3 SPA with SSR, dark mode, a real component library,
and live WebSocket updates.

## Stack

- **Nuxt 4** with the `app/` directory layout
- **Nuxt UI v3** (Tailwind v4 + Reka UI) for components
- **@pinia/nuxt** for cross-page state
- **@vueuse/nuxt** for composables
- **marked + DOMPurify** for markdown rendering
- **Nitro** for the server (in production, proxies `/api/*` and `/ws`
  to the MCP backend)

## Project layout

```
web/
├── app/
│   ├── app.vue, error.vue, app.config.ts
│   ├── assets/css/main.css
│   ├── components/        # layout, common, domain
│   ├── composables/       # useApi, useServerUrl, useRealtime, useToastBus
│   ├── layouts/           # default, blank
│   ├── middleware/        # connection.global.ts
│   ├── pages/             # index, projects/[index|id], agents, tasks, documentation, standups, settings
│   ├── stores/            # settings, projects
│   ├── types/             # api.ts, ws.ts
│   └── utils/             # formatters.ts
├── public/                # icon.png, logo.png, favicon.ico
├── server/
│   └── routes/
│       ├── api/[...path].ts   # prod proxy /api/* → MCP backend
│       └── ws.ts              # prod WebSocket proxy /ws → MCP backend
├── nuxt.config.ts
├── Dockerfile
└── package.json
```

## Development

Prerequisites: Node 20+, the MCP backend running on `:8080`.

```bash
cd web
npm install
npm run dev
```

Open http://localhost:3000. The first time you visit, open **Settings**
and confirm the MCP server URL (default `http://localhost:8080`).

`/api/*` and `/ws` are proxied to the backend by `nitro.devProxy`
(no nginx, no extra config).

## Build & preview

```bash
npm run build      # → .output/
npm run preview    # serve the built app on :3000
```

## Docker

```bash
docker compose up -d --build
```

The `web` service in `docker-compose.yml` builds `./web`, exposes `:80`,
and runs `node .output/server/index.mjs` inside the container on `:3000`.

In production, the Nitro server proxies `/api/*` and `/ws` to the
`mcp-server` service (hostname resolved by docker-compose).

## Configuration

| Env var                  | Purpose                              | Default                  |
| ------------------------ | ------------------------------------ | ------------------------ |
| `NUXT_PUBLIC_API_BASE`   | Default MCP API base URL             | `http://localhost:8080`  |
| `NUXT_PUBLIC_WS_BASE`    | Default MCP WebSocket base URL       | `ws://localhost:8080`    |
| `NUXT_API_UPSTREAM`      | Prod proxy target (api)              | `http://mcp-server:8080/api` |
| `NUXT_API_UPSTREAM_WS`   | Prod proxy target (ws)               | `ws://mcp-server:8080`   |

Server URL is also persisted in a per-user cookie (`mcp-server-url`)
and can be changed at runtime in **Settings**.

## Type checking

```bash
npm run typecheck
```

## License

Same as the main Agent Shaker project.
