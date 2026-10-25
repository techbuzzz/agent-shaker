# Connecting an AI agent (MCP setup)

Agent Shaker is only useful if agents can actually reach it. This is how you
connect one — Claude Code, Codex, Cursor, Copilot, or any MCP host — to a
project.

There are two halves to it, and they are separable on purpose:

1. **The configuration** — which file goes where, and which URL it points at.
   Generated for one agent by `GET /api/agents/{id}/config`.
2. **The credential** — a key bound to that agent.
   Issued by `POST /api/agents/{id}/connection`, shown once.

In the UI they are one flow: **Agents → an agent → the plug icon → Create
connection**. You get the files and the key together, and the key is copied into
the file in your browser.

> **Status:** implemented. The `docs/VS2026_*.md` set describes an earlier
> attempt that targeted the pre-Nuxt `web/src/` tree and never reached the
> backend or the UI; it is superseded by this document.

## Choosing the tool

Every agent has a `tool_type` — the client it runs in — set at registration.
That field is what picks the file layout, so "the file we told you to write" and
"what this agent actually is" cannot drift apart.

| `tool_type` | File generated |
| --- | --- |
| `claude-code` | `.claude/mcp.json` |
| `codex` | `~/.codex/mcp.json` (user-level — Codex has no per-project config) |
| `cursor` | `.cursor/mcp.json` |
| `opencode` | `opencode.json` (OpenCode keeps MCP servers in its one settings document) |
| `copilot-vs` | `.vscode/mcp.json` + `.github/copilot-instructions.md` |
| `copilot-cli` | `.vscode/mcp.json` + `.github/copilot-instructions.md` |
| anything else | `.mcp.json`, the convention most MCP hosts read |

Windsurf and `a2a` are recognised values without a dedicated layout yet; those
agents get the generic file, which connects.

Every bundle also carries two documents: `AGENT_SHAKER.md` — or
`.github/copilot-instructions.md` for Copilot — which tells the agent how the
server is meant to be used, and `AGENT_SHAKER_SETUP.md`, which says which file
goes where, how to set the key, and how to check that the connection is alive.

### VS Code and Copilot read different keys

VS Code reads `servers`; everything else reads `mcpServers`. The two are not
interchangeable and the failure is silent — a file with the wrong top-level key
is valid JSON that VS Code ignores without a word, so the agent connects to
nothing.

Copilot in VS Code reads its MCP servers from `.vscode/mcp.json` as well, and
reads repository instructions from `.github/copilot-instructions.md`
automatically. The `copilot` format ships that pair together: a Copilot user
given only the instructions would have a bundle that connects to nothing.

### The credential spelling is per client

Not a style preference — it is the difference between the client expanding the
value and handing the agent the literal string:

| Client | Spelling |
| --- | --- |
| Claude Code, Cursor, Codex, `mcpServers` convention | `${env:AGENT_SHAKER_API_KEY}` |
| [OpenCode](https://opencode.ai/docs/mcp-servers/) | `{env:AGENT_SHAKER_API_KEY}` |

OpenCode also differs in two structural ways: its servers live under an `mcp`
key rather than `mcpServers`, and every entry must declare `"type": "remote"` —
without it OpenCode treats the server as a local stdio process and silently runs
nothing. The generator emits all three per format, and the prose in
`AGENT_SHAKER.md` carries the same spelling as the config beside it, so an agent
copying the instruction writes the form its own client expands.

Alongside the config, the generator emits `AGENT_SHAKER.md`: how the agent is
expected to work with the server. It is not decoration — the most common real
failure is not a broken connection but a *working* one in which the agent never
calls `send_heartbeat`, because nothing told it a lease existed.

## The API key is referenced, never embedded

The generated configs contain:

```json
"headers": { "Authorization": "Bearer ${env:AGENT_SHAKER_API_KEY}" }
```

not a literal key. That is deliberate: these files are meant to be **committed**
to the agent's repository, and a literal key in a tracked file is a leaked key.

The operator sets the variable in the agent's environment:

```bash
export AGENT_SHAKER_API_KEY=<the key issued for this agent>
```

Keep it in the shell or a git-ignored `.env` — never in a tracked file.

This holds even in the connection modal, where the plaintext *is* present in the
browser: **Copy** substitutes it into the file you are about to paste, and
**Download** deliberately keeps the placeholder, so the file you save is still
safe to commit. The plaintext lives in one ref for as long as the modal is open
and is dropped when it closes — the server keeps only a hash and cannot return
it again.

## How the connection is scoped

The generated URL binds the project and the agent:

```
https://<your-host>/mcp?project_id=<uuid>&agent_id=<uuid>
```

The Go handler reads `project_id` / `agent_id` from the query string (or the
`X-Project-ID` / `X-Agent-ID` headers). Binding them means the scoped tools work
without the agent restating its identity on every call — `get_my_tasks` returns
"my" tasks because the connection says who "my" is.

### The URL identity is a claim, not the identity

With a key issued for an agent, the **`agent_id` in the URL is checked, not
believed**. The key names the agent; a URL naming somebody else is refused with
a 403:

```
403  agent identity does not match the credential: this credential is bound to
     agent <uuid>; drop the agent_id from the connection URL
```

That rule is the reason the connection exists at all. A key scoped only to a
project, plus an `agent_id` in a query string, let any holder of that key act as
any agent in the project.

A key with **no** agent binding — every key issued before this existed, and the
`API_KEYS` shared secrets — still supplies `agent_id` in the URL and is still
trusted, which is what keeps existing setups working. See
[AUTHENTICATION.md](./AUTHENTICATION.md).

### Where the URL comes from

The origin is resolved server-side: `BASE_URL` if set, otherwise the request's
own hop honouring `X-Forwarded-Proto` / `X-Forwarded-Host`, otherwise the
request's scheme and host. An agent runs outside the compose network, so a
config advertising `localhost:8080` would send it to a dead address. Set
`BASE_URL` for a deployment exposed directly to the internet.

## Verifying the connection

```bash
curl -sS -X POST "https://<your-host>/mcp?project_id=<uuid>&agent_id=<uuid>" \
  -H "Authorization: Bearer $AGENT_SHAKER_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

Expected: a JSON-RPC result naming `agent-shaker` as the server.

| Symptom | Cause |
| --- | --- |
| `401` | Wrong or unset key |
| `403` + "agent identity does not match" | The key is bound to a different agent than the URL names. Drop `agent_id` from the URL, or use the key issued for that agent |
| `404` | The edge is not routing `/mcp`. That path is served by the `tls` profile's Caddy; see [DEPLOYMENT.md](./DEPLOYMENT.md) |
| Connection refused | The Go service publishes to loopback only. Check `docker compose ps` for `127.0.0.1:8080->8080`, or set `MCP_BIND=0.0.0.0` |
| `no such host: mcp-server` | The config is using the compose-internal name. Agents run on your machine and cannot resolve it — use the published host port, or set `BASE_URL` and re-generate |

## Reaching the server

MCP clients do not run inside the compose network, so the Go service publishes
its port to the host rather than merely exposing it to its neighbours:

```yaml
ports:
  - "${MCP_BIND:-127.0.0.1}:${MCP_PORT:-8080}:8080"
```

Loopback by default, like `web`. `MCP_BIND=0.0.0.0` lets agents on other
machines connect, and carries no TLS of its own — behind the `tls` profile, keep
it on loopback and let Caddy be the entrypoint.

An internal address like `http://mcp-server:8080` is not a hostname an agent can
resolve; that name exists only inside the compose network. The generator never
emits one, because it derives the origin from `BASE_URL` or from the request's
own host.

## The tools an agent gets

`tools/list` returns 28. The generated instructions group them:

- **Orientation** — `get_my_identity`, `get_my_project`
- **Working on tasks** — `get_my_tasks`, `claim_task`, `complete_task`, `complete_task_with_handoff`, `reassign_task`, `update_my_status`
- **Staying visible** — `send_heartbeat`
- **Blocked** — `raise_blocker`, `check_blocker_status`, `get_my_blockers`
- **Dependencies** — `add_task_dependency`, `get_task_dependencies`
- **Tasks and planning** — `list_tasks`, `create_task`, `update_task_status`, `get_dashboard`, `list_milestones`, `create_milestone`, `assign_task_to_milestone`
- **Shared knowledge** — `list_contexts`, `search_contexts`, `add_context`, `list_global_contexts`, `read_global_context`, `publish_global_context`
- **Other agents** — `discover_a2a_agent`, `delegate_to_a2a_agent`, `get_a2a_task_status`
- **Onboarding** — `register_self`

Milestone and global-context writes are PM-gated; a non-PM agent gets a clear
refusal rather than a silent no-op.

The PM agent itself cannot be created by an agent: `register_self` refuses
`role: "pm"`. The PM gate reads the role of the agent on the connection, and
`register_self` creates the row that connection would then use, so allowing it
would let an agent promote itself. A human creates the PM agent in the UI and
issues it a key. Every other role is open to an agent — none of them carries
approval authority.

## The working loop

The generated instructions spell this out for the agent, but it is worth stating
here because it is the intended usage:

1. `get_my_identity` — confirm the connection is bound as expected.
2. `get_my_tasks` — see what is assigned.
3. `claim_task` — mark it in progress so other agents can see.
4. Do the work, sending `send_heartbeat` throughout.
5. `complete_task_with_handoff` — report back, passing on context if it helps.
6. `raise_blocker` the moment you are stuck. Silence is not a report.
7. `list_contexts` before deciding, `add_context` after — so the next agent
   inherits the reasoning.

Live updates also arrive over the app's WebSocket. MCP is for *acting*; the UI
is for *watching*.

## Rotating a key

Mint a new one from the agent's Connect action, update the agent's environment,
then revoke the old key (**Access**). Minting is not idempotent — a retry after a
timeout leaves a second valid credential, so revoke the one you did not use.

Revoking takes effect on the agent's very next call; the credential cache is
invalidated rather than waited out.

## Relationship to A2A

MCP is agent-to-Agent-Shaker. A2A is Agent-Shaker-to-another-agent: see
[A2A_INTEGRATION.md](./A2A_INTEGRATION.md). An MCP client can call
`discover_a2a_agent` and `delegate_to_a2a_agent` to hand work to an external A2A
agent it knows nothing about.

## Implementation

- Generator: `internal/configgen` — pure, no database and no HTTP, and the
  single source of truth for a generated file's contents
- Endpoints: `internal/handlers/agent_config.go`
  (`GET /api/agents/{id}/config`, `POST /api/agents/{id}/connection`)
- Origin resolution: `internal/publicurl`, shared with the A2A agent card
- Identity rule: `internal/mcp/handler.go` (`resolveIdentity`,
  `applyPrincipalIdentity`)
- UI: `web/app/components/domain/AgentConnectionModal.vue` (issue a
  connection), `McpSetupModal.vue` (fetch and download the files)
- Secret substitution in the browser:
  `web/app/composables/useAgentConfig.ts` — `split`/`join`, never
  `String.replace`, because the placeholder contains `$` and `$&` / `` $` ``
  would be interpreted as substitution patterns
- Routing and scopes: `cmd/server/routes.go`

## Beyond the generated config

The generated files are enough for a host that only speaks MCP. Two hosts in
this project's own toolchain can do more, because they have an extension
point that lets them act rather than only observe.

### OpenCode: the plugin package

`plugins/opencode` is a published npm package (`@techbuzzz/opencode-agent-shaker`)
that installs into OpenCode and adds the things MCP cannot reach:

- the session's **heartbeat lease**, so the server knows the agent is alive;
- the **compaction bridge**, which writes the summary back into the shared
  contexts and re-injects the shared truth into the new, smaller context;
- the **policy gate**, which can block a tool call before it runs.

It exports both the v1 and the v2 plugin shapes from one entry point, because
OpenCode has two plugin APIs and the v1 object form is only supported from
1.18.29 onward.

**Two configuration formats, and they are not interchangeable.** `mcp` is one
key with two shapes, so one file cannot serve both majors: OpenCode 1 reads
the servers directly under `mcp`, OpenCode 2 reads them under `mcp.servers`.
Ask for the one that matches the host:

```
GET /api/agents/{id}/config?format=opencode      # OpenCode 1.x
GET /api/agents/{id}/config?format=opencode-v2   # OpenCode 2.x
```

The v2 document also sets `"oauth": false`. OpenCode 2 turns OAuth on for
every remote server unless the file says otherwise, and this server
authenticates with a header — without the field the client tries to
authenticate against a server that does not speak OAuth, and the agent
connects to nothing.

```jsonc
// opencode.json — the v2 shape
{
  "mcp": {
    "servers": {
      "agent-shaker": {
        "type": "remote",
        "url": "https://agents.example.internal/mcp",
        "disabled": false,
        "headers": { "Authorization": "Bearer {env:AGENT_SHAKER_API_KEY}" },
        "oauth": false,
        "protocol": "legacy"
      }
    }
  },
  "plugins": [
    { "package": "@techbuzzz/opencode-agent-shaker",
      "options": { "endpoint": "https://agents.example.internal/mcp" } }
  ]
}
```

### The organisation manifest

`GET /.well-known/opencode` publishes the same document for a whole
organisation. OpenCode fetches it by itself, before any credential exists, and
merges it underneath every other configuration source — so one endpoint hands
a fleet of agents the same coordination server and the same plugin.

It is unauthenticated on purpose, and it carries no secret: every credential in
it is the `{env:AGENT_SHAKER_API_KEY}` placeholder. The connection it publishes
is unbound, because an organisation does not know which agent is about to
connect.

### What a client says about itself

An agent running the plugin announces once, at setup, what its client can do
for the server — `compaction`, `context_injection`, `policy_gate`,
`evidence_capture`. The server needs the answer before it routes work: an
agent whose client cannot accept a pushed context, or author a compaction
summary, is an agent the server must not ask to.

A bare MCP client announces nothing, and that is a different fact from
announcing an empty list: the first means "I have not told you what I am", the
second means "I have, and the answer is nothing".

### MiniMax Code: a project hook

MiniMax Code has hooks of its own, so it needs no plugin. The repository
carries one:

- `.minimax/hooks.json` — `SessionStart`, `SessionEnd` and `PostCompact`.
- `.minimax/agent-shaker-hook.mjs` — the script they call.

`SessionStart` renews the agent's lease, `SessionEnd` releases it, and
`PostCompact` reads the transcript and pushes it into the shared contexts —
the same compaction bridge the OpenCode plugin implements.

The hook file is inert until the operator enables it in
**Settings → Hooks → Project**. The script reads:

```
AGENT_SHAKER_URL         the API base URL
AGENT_SHAKER_API_KEY     the agent's credential
AGENT_SHAKER_PROJECT_ID  the project the session belongs to
AGENT_SHAKER_AGENT_ID    the agent this session is
```

It is fail-open and side-effect only: it never blocks a turn, never writes to
stdout, and always exits 0.

### The other direction: the server runs a command

The hook layer inverts the relationship. Instead of an agent reporting to the
server, a **task lifecycle event runs a command on the server** — for example
a reviewer agent that reads the finished task and writes its verdict back as
a context.

See [PHASE3_OPENCODE.md](./PHASE3_OPENCODE.md) for the design, and
`scripts/hook-e2e.sh` for a proof that runs the whole path against a
throwaway stack.