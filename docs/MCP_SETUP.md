# Connecting an AI agent (MCP setup)

Agent Shaker is only useful if agents can actually reach it. This is how you
connect one — Copilot, Claude, or any MCP host — to a project.

The UI has a generator for this: open a project, then **Connect an AI agent →
Get config**. It produces four files you commit to the agent's repository.

> **Status:** implemented. The `docs/VS2026_*.md` set describes an earlier
> attempt that targeted the pre-Nuxt `web/src/` tree and never reached the
> backend or the UI; it is superseded by this document.

## The four generated files

| File | Read by | Purpose |
| --- | --- | --- |
| `.mcp.json` | Visual Studio 2026, MCP hosts using this convention | The server connection |
| `.vscode/mcp.json` | Visual Studio Code | Same, under VS Code's `servers` key |
| `.github/copilot-instructions.md` | GitHub Copilot | Explains what the server is for and how to work with it |
| `AGENT_SHAKER_SETUP.md` | Humans | Step-by-step setup and how to verify the connection |

## The API key is referenced, never embedded

The generated configs contain:

```json
"headers": { "Authorization": "Bearer ${env:AGENT_SHAKER_API_KEY}" }
```

not a literal key. That is deliberate: these files are meant to be **committed**
to the agent's repository, and a literal key in a tracked file is a leaked key.

The operator sets the variable in the agent's environment:

```bash
export AGENT_SHAKER_API_KEY=<one of the server's API_KEYS values>
```

Keep it in the shell or a git-ignored `.env` — never in a tracked file.

This is also why the UI has no "paste your key" field. The SPA cannot obtain a
key (it is server-only runtimeConfig), and a form that accepts a live secret is
one XSS away from exfiltrating it. Generating a reference instead keeps the
secret entirely out of the browser.

## How the connection is scoped

The generated URL binds the project, and optionally the agent:

```
https://<your-host>/mcp?project_id=<uuid>&agent_id=<uuid>
```

The Go handler reads `project_id` / `agent_id` from the query string (or the
`X-Project-ID` / `X-Agent-ID` headers). Binding them means the scoped tools work
without the agent restating its identity on every call — `get_my_tasks` returns
"my" tasks because the connection says who "my" is.

Leave the agent unbound if the agent should call `register_self` on first
connect.

### Why the URL is derived from the browser origin

The generator reads `window.location.origin` rather than a configured value.
That is what makes one config correct in every topology: with the TLS edge the
browser origin *is* the public origin, and without it the browser origin is the
Nuxt service. Either way the agent is pointed somewhere reachable from outside
the container.

## Verifying the connection

```bash
curl -sS -X POST "https://<your-host>/mcp?project_id=<uuid>" \
  -H "Authorization: Bearer $AGENT_SHAKER_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

Expected: a JSON-RPC result naming `agent-shaker` as the server.

| Symptom | Cause |
| --- | --- |
| `401` | Wrong or unset key |
| `404` | The edge is not routing `/mcp` — that path is only published under the `tls` profile. See [DEPLOYMENT.md](./DEPLOYMENT.md) |
| Connection refused | `mcp-server` publishes no host port; go through the edge |

## The tools an agent gets

`tools/list` returns 25+. The generated Copilot instructions group them:

- **Orientation** — `get_my_identity`, `get_my_project`
- **Working on tasks** — `get_my_tasks`, `claim_task`, `complete_task`, `reassign_task`, `update_my_status`
- **Tasks and planning** — `list_tasks`, `create_task`, `update_task_status`, `get_dashboard`, `list_milestones`, `create_milestone`, `assign_task_to_milestone`
- **Shared knowledge** — `list_contexts`, `add_context`, `list_global_contexts`, `read_global_context`, `publish_global_context`
- **Other agents** — `discover_a2a_agent`, `delegate_to_a2a_agent`, `get_a2a_task_status`
- **Onboarding** — `register_self`

Milestone and global-context writes are PM-gated; a non-PM agent gets a clear
refusal rather than a silent no-op.

## The working loop

The generated instructions spell this out for the agent, but it is worth stating
here because it is the intended usage:

1. `get_my_identity` — confirm the connection is bound as expected.
2. `get_my_tasks` — see what is assigned.
3. `claim_task` — mark it in progress so other agents can see.
4. Do the work.
5. `complete_task` — report back.
6. `update_my_status` — flag being blocked or waiting.
7. `list_contexts` before deciding, `add_context` after — so the next agent
   inherits the reasoning.

Live updates also arrive over the app's WebSocket. MCP is for *acting*; the UI
is for *watching*.

## Relationship to A2A

MCP is agent-to-Agent-Shaker. A2A is Agent-Shaker-to-another-agent: see
[A2A_INTEGRATION.md](./A2A_INTEGRATION.md). An MCP client can call
`discover_a2a_agent` and `delegate_to_a2a_agent` to hand work to an external A2A
agent it knows nothing about.

## Implementation

- Generator: `web/app/composables/useMcpSetup.ts` (pure builders, no Vue
  dependency in the generation path)
- UI: `web/app/components/domain/McpSetupModal.vue`
- Wiring: the project detail page, plus a persistent call-to-action above the
  tabs
- Server side: `internal/mcp/handler.go` (tool surface) and
  `cmd/server/routes.go` (routing and auth)
