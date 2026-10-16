# MCP Multi-Agent Architecture

## System Overview

The deployment is single-origin by design. The browser only ever talks to the
Nuxt service; the Go API is never published to the host, and everything it
serves is reached over the internal compose network.

```
┌─────────────────────────────────────────────────────────────────┐
│                              Browser                            │
└───────────────────────┬─────────────────────────────────────────┘
                        │  HTTP(S), one origin
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Caddy  (only under the `tls` profile)         │
│      public entrypoint · TLS · basic auth on the human branch    │
│   HUMAN    /, /api/*, /ws, /healthz            → web:3000       │
│   MACHINE  /mcp*, /a2a/*, /.well-known/*       → mcp-server:8080│
└───────────────────────┬─────────────────────────────────────────┘
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│              web — Nuxt 4 SSR, listens on :3000                  │
│  • serves the UI from web/app/ (pages, components, composables)  │
│  • Nitro proxy forwards /api and /ws to the Go service           │
│  • injects NUXT_API_KEY server-side when the caller sent none    │
└───────────────────────┬─────────────────────────────────────────┘
                        │  server-to-server, internal network only
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│         mcp-server — Go API, listens on :8080                    │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  REST  /api/*        projects, agents, tasks, contexts,     │ │
│  │                      standups, heartbeats, milestones,     │ │
│  │                      project_repos, global_contexts,       │ │
│  │                      people*, keys*, dashboard              │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │  MCP   /mcp, /mcp/message, /                               │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │  A2A   /a2a/v1/message, message:stream, tasks, artifacts   │ │
│  │        /.well-known/agent-card.json                        │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │  WS    /ws            live task and agent events            │ │
│  └────────────────────────────────────────────────────────────┘ │
│  `expose: 8080` only — no host port is published                │
└───────────────────────┬─────────────────────────────────────────┘
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│                    PostgreSQL 16                                 │
│                                                                  │
│   projects · agents · tasks · contexts · daily_standups         │
│   agent_heartbeats · a2a_tasks · milestones · project_repos     │
│   global_contexts · people · api_keys                           │
└─────────────────────────────────────────────────────────────────┘
```

Notes on the shape:

- **No published database port.** `postgres` has no `ports:` mapping; a
  `127.0.0.1:5433:5432` binding exists only as a commented-out line for local
  `psql` access. Nothing in the running topology listens on 5433.
- **No published API port.** `mcp-server` declares `expose: "8080"`, which
  documents the port on the compose network without binding it on the host.
  TLS termination belongs in front of the service.
- **`web` binds loopback only** (`127.0.0.1:3000` by default) so that when the
  `tls` profile is active, Caddy is the sole public entrypoint.
- **Two caller branches, split by who calls.** The machine branch cannot go
  through Nuxt: `/` is a registered MCP alias, and SSE replies (`message:stream`)
  need a real pass-through proxy. See `deploy/Caddyfile` for the full rationale.
- **Migrations run as a one-shot `migrate` job** that exits; `mcp-server`
  depends on `service_completed_successfully`, so the API never starts against a
  stale schema.

## Repository Layout

```
cmd/server/          API + MCP + A2A + WebSocket HTTP server (:8080)
cmd/migrate/         standalone migration runner (the `migrate` compose job)
internal/a2a/        A2A protocol: server, outbound client, models, mapper
internal/authkey/    key minting, hashing and well-formedness helpers
internal/configgen/  generated MCP configuration (the single source of truth)
internal/database/   connection, queries, task store implementations
internal/handlers/   REST handlers, one file per resource
internal/httpx/      shared HTTP response and error envelopes
internal/mcp/        MCP tool definitions, schemas and runtime validation
internal/middleware/ auth, scopes, CORS, rate limit, logging, headers
internal/models/     domain types and the status vocabularies (status.go)
internal/observability/ Prometheus metrics and OpenTelemetry wiring
internal/publicurl/  origin resolution for anything a client outside reads
internal/task/       A2A task lifecycle (Manager, Store, TaskExecutor)
internal/validator/  request payload validation
internal/websocket/  hub and connection lifecycle
migrations/          numbered *.up.sql, applied by cmd/migrate
web/app/             Nuxt 4 application (the current frontend)
deploy/Caddyfile     TLS terminator and public perimeter
```

The earlier Vue SPA (`web/src/views`, Pinia stores, Vue router) is not part of
the build. It is preserved only as `web/.deprecated-spa/`.

## Data Model

`migrations/` is the source of truth; the tables are:

| Table | Purpose |
|---|---|
| `projects` | Top-level workspace. Every other scoped table hangs off `project_id`. |
| `agents` | Registered agents belonging to a project, with role and presence status. |
| `tasks` | Coordination units of work, with agent, status, priority and dependencies. |
| `contexts` | Per-task documentation shared between agents. Full-text searchable. |
| `daily_standups` | Per-project, per-day standup entries. |
| `agent_heartbeats` | Presence signal, written by `POST /api/heartbeats`. |
| `a2a_tasks` | A2A-protocol task state: message, result and artifacts as JSONB. |
| `milestones` | Project milestones with their own status, created by an agent. |
| `project_repos` | Repositories attached to a project, optionally owned by an agent. |
| `global_contexts` | Context that spans every project rather than one task. |
| `people` | The subject action is attributed to: `human` or `service`. |
| `api_keys` | Hashed credentials, each bound to a person, with scopes and optional project scoping. |

Two migrations are worth calling out because they are not merely additive:

- **`013_context_search`** adds a generated `tsvector` column over `contexts`
  and a GIN index, so an agent can select the right context instead of listing
  and reading every one.
- **`014_status_constraints`** normalises legacy status values and then adds
  `CHECK` constraints on `tasks.status` and `agents.status`, plus a
  `(project_id, status)` index.

### Status vocabularies

The vocabularies are defined once, in `internal/models/status.go`, and every
surface derives from them — the REST validator, the MCP tool schemas, MCP
runtime validation, the SQL queries, and the migration 014 constraints. Adding a
status means adding it in that file and in the migration, never at a call site.

| `tasks.status` | `agents.status` |
|---|---|
| `pending` | `active` |
| `in_progress` | `idle` |
| `blocked` | `blocked` |
| `done` | `offline` |
| `cancelled` | |

`done` and `cancelled` are terminal: terminal tasks are not claimable and do not
count towards open work.

The A2A task vocabulary in `internal/a2a/models` is a separate, protocol-fixed
set (`submitted`, `working`, `completed`, `failed`, `canceled`) and is
deliberately not shared with the `tasks` table.

## Authentication and Authorization

A request is authenticated once at the edge and carries a **Principal** for the
rest of its life. A Principal is the answer to three separate questions — who
this is (attribution), what they may do (permissions), and are they a human
(approval authority) — which a bare "is this key valid?" check cannot express.

There are two credential paths:

1. **Database-backed keys (the main path).** `StoreKeyResolver` implements the
   `KeyResolver` interface: it hashes the presented credential with
   `internal/authkey`, looks the digest up in `api_keys`, and builds a Principal
   from the row's `principal_type`, `principal_id`, `scopes` and optional
   `project_id`. Only the SHA-256 hash is stored; the plaintext exists exactly
   once, in the `POST /api/keys` response. Decisions are cached for 15 s
   (5 s for negatives) so that a flood of distinct wrong keys costs a bounded
   number of probes, and a revocation is propagated immediately via
   `InvalidateKey`. A store outage is reported as unavailable rather than as a
   rejected credential, so an incident is not misread as an attack.

2. **`API_KEYS` from the environment (the bootstrap path).** A flat shared
   secret still authenticates, so an existing deployment keeps working
   unchanged. It resolves to a synthetic `LegacyPrincipal` marked `Synthetic`,
   which is how the audit trail keeps "a human did this" distinguishable from
   "somebody holding a shared secret did this". This path is intended to get you
   onto the database-backed path, not to stay there.

Authorization is scope-based and enforced outside the handlers, so a new route
cannot ship unauthorised and a deleted route cannot silently lose its
authorisation:

- `/api/keys` and `/api/people` require the `admin` scope, applied by path
  prefix (`RequireScopeOnPaths`) regardless of method. Minting a key is the only
  endpoint that returns a live secret, which is why the gate is not per-handler.
- Every other `/api` route is graded by HTTP method (`RequireMethodScope`): safe
  methods need `read`, everything else needs `write`, and an unrecognised verb
  fails closed as a mutation.
- A key with an empty `scopes` array is unrestricted, which is what lets an
  existing deployment migrate without a flag day.
- A request that reaches a guarded route with no principal in context is refused
  and logged: that combination means a wiring mistake, not an anonymous user.

The same `Principal` governs the MCP, A2A and WebSocket surfaces; `/ws`
additionally accepts `?api_key=`, which is the only way a browser can
authenticate a handshake, and it is the only route on which a credential is
accepted from a query string.

### Agent identity on the MCP surface

A key may name an agent (`api_keys.agent_id`). When it does, that binding is the
identity: the `agent_id` in an MCP connection URL is checked against it rather
than believed, and a disagreement is refused with a 403.

Without this, the MCP handler took `agent_id` from the query string and used it
as-is, while the key behind the request was scoped only to a project — so any
holder of a project key could act as any agent in it. Every identity-aware tool
(`get_my_identity`, `get_my_tasks`, `claim_task`, `send_heartbeat`, the blocker
tools) reads the value once resolved in `HandleMCP`, which is what keeps the rule
in one place instead of one per tool.

Keys with no binding — every key issued before migration 017, and the
`API_KEYS` shared secrets — keep supplying `agent_id` in the URL and keep being
trusted, because that is the only identity they have. See
[AUTHENTICATION.md](AUTHENTICATION.md) for the full precedence table.

## Configuration Generation

`internal/configgen` is the single source of truth for what a generated MCP
configuration contains. It is a pure package: an origin in, strings out, with no
database, no HTTP and no environment lookups.

It lives on the server rather than in the browser because the generator would
otherwise exist twice — once in the Nuxt bundle and once in any CLI or plugin —
and two implementations of "what should this config say" drift apart. One gets a
tool type added, the other does not, and a developer receives a config for the
wrong client with nothing pointing at the cause.

Three invariants, each pinned by a test:

- **No credential in a rendered file.** Every header references
  `${env:AGENT_SHAKER_API_KEY}`. The plaintext is substituted by the browser at
  copy time, in memory, from the one response that carried it — which is what
  makes a generated file safe to commit.
- **The URL shape is unchanged** from the client-side generator this replaced, so
  a config generated here works against a server that has not been upgraded.
- **Formats are chosen from `agents.tool_type`** (migration 017), which is why
  that column exists: the operator picks the tool once, at registration, and
  every later request renders the right layout.

`internal/publicurl` resolves the origin those files point at, shared with the
A2A agent card. An agent runs outside the compose network, so a config
advertising `localhost:8080` sends it to a dead address; `BASE_URL` overrides,
otherwise the request's own hop (honouring `X-Forwarded-*`) decides.

## A2A Boundary: Coordination, Not Execution

Agent Shaker is a coordination plane. It does not run agents, and the inbound
path is honest about that:

- **Outbound delegation is implemented.** `internal/a2a/client` discovers a
  remote agent from its card, sends messages, follows and lists tasks, streams
  updates, and reads artifacts; `internal/a2a/mapper` translates between MCP
  and A2A models. This is covered by `tests/a2a/external_agent_test.go`.
- **Inbound task execution is deliberately not implemented.** The task `Manager`
  is built without a `TaskExecutor`, so a submitted inbound task reaches the
  terminal `failed` state carrying the explicit reason `ErrNoExecutor` — "no
  task executor configured: this deployment does not execute A2A tasks". The
  agent card says the same thing in the `task_submission` skill description.
  This replaced an earlier behaviour that echoed the message back and reported
  `completed`, which told a remote agent that work had been done when none had.

Use the MCP surface to assign work to registered agents.

## Workflow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     Developer Workflow                           │
└─────────────────────────────────────────────────────────────────┘

1. Setup Phase
   ┌─────────────────────────────────────────────────────────┐
   │ $ docker compose up -d                                  │
   │ ✓ migrate runs once, then exits                        │
   │ ✓ mcp-server starts on :8080 (internal only)           │
   │ ✓ postgres starts                                       │
   │ ✓ web (Nuxt SSR) starts on :3000                        │
   └─────────────────────────────────────────────────────────┘
                           │
                           ▼
   ┌─────────────────────────────────────────────────────────┐
   │ $ .\scripts\setup-agent.ps1 -Interactive                 │
   │ → Create Project: "InvoiceAI"                           │
   │ → Register Agent: "InvoiceAI-Frontend"                  │
   │ → Role: frontend                                         │
   │ ✓ agent-config.json created                             │
   │ ✓ agent-helper script created                           │
   └─────────────────────────────────────────────────────────┘

2. Development Phase
   ┌─────────────────────────────────────────────────────────┐
   │ $ . .\agent-helper-InvoiceAI-Frontend.ps1                │
   │ ✓ Helper functions loaded                               │
   └─────────────────────────────────────────────────────────┘
                           │
                           ▼
   ┌─────────────────────────────────────────────────────────┐
   │ $ Get-MyTasks                                           │
   │ → Fetches tasks assigned to this agent                  │
   └─────────────────────────────────────────────────────────┘
                           │
                           ▼
   ┌─────────────────────────────────────────────────────────┐
   │ $ New-Task -Title "Build invoice form" -Priority "high" │
   │ → Creates new task in the system                        │
   │ ✓ Task created with ID                                  │
   └─────────────────────────────────────────────────────────┘
                           │
                           ▼
   ┌─────────────────────────────────────────────────────────┐
   │ $ Update-TaskStatus -TaskId "xxx" -Status "in_progress" │
   │ → Agent starts working on the task                      │
   └─────────────────────────────────────────────────────────┘
                           │
                           ▼
   ┌─────────────────────────────────────────────────────────┐
   │ Developer works with AI assistant (Copilot)             │
   │ • Copilot knows the agent identity                      │
   │ • Can check task list                                   │
   │ • Aware of project context                              │
   └─────────────────────────────────────────────────────────┘
                           │
                           ▼
   ┌─────────────────────────────────────────────────────────┐
   │ $ Add-TaskContext -TaskId "xxx" -Context "..."          │
   │ → Documents implementation details                       │
   │ → Shares knowledge with other agents                    │
   └─────────────────────────────────────────────────────────┘
                           │
                           ▼
   ┌─────────────────────────────────────────────────────────┐
   │ $ Update-TaskStatus -TaskId "xxx" -Status "done"        │
   │ → Marks task as complete                                │
   │ ✓ Other agents notified (via WebSocket)                 │
   └─────────────────────────────────────────────────────────┘

3. Collaboration Phase
   ┌─────────────────────────────────────────────────────────┐
   │          Frontend Agent          Backend Agent           │
   │                                                           │
   │  Task: Build Invoice UI    →    Task: Create Invoice API│
   │  Status: Pending                 Status: In Progress     │
   │  Waiting for API...              Working...              │
   │                                                           │
   │                            ←    API Complete!            │
   │                                  Status: Done            │
   │                                  Context: API docs shared│
   │                                                           │
   │  Task Status: In Progress                                │
   │  Searches the shared contexts instead of listing them    │
   │  Builds UI components                                    │
   │                                                           │
   │  Task Complete!                                          │
   │  Context: UI patterns shared                             │
   └─────────────────────────────────────────────────────────┘
```

## Agent Communication Flow

```
┌──────────────┐                                    ┌──────────────┐
│   Agent 1    │                                    │   Agent 2    │
│  (Frontend)  │                                    │  (Backend)   │
└──────┬───────┘                                    └──────┬───────┘
       │                                                   │
       │ POST /api/tasks                                   │
       │ { title: "Build Invoice Form" }                  │
       ├──────────────────────►│                          │
       │                       │                          │
       │     201 Created       │                          │
       │   { id: "task-123" }  │                          │
       │◄──────────────────────┤                          │
       │                       │                          │
       │                       │  POST /api/tasks         │
       │                       │  { title: "Create API" } │
       │                       │◄─────────────────────────┤
       │                       │                          │
       │                       │      201 Created         │
       │                       ├─────────────────────────►│
       │                       │                          │
       │ WebSocket Connection  │  WebSocket Connection    │
       ├──────────────────────►│◄─────────────────────────┤
       │                       │                          │
       │                       │ PUT /api/tasks/xxx/status│
       │                       │ { status: "done" }       │
       │                       │◄─────────────────────────┤
       │                       │                          │
       │ ◄─── WS Event ────────┤                          │
       │ { event: "done", task_id: "xxx" }                │
       │                       │                          │
       │ GET /api/contexts/search                           │
       │ ?q=invoice+api                                     │
       ├──────────────────────►│                          │
       │                       │                          │
       │  200 OK (matching contexts)                        │
       │◄──────────────────────┤                          │
       │                       │                          │
       │ Now can start UI work │                          │
       │                       │                          │
```

## Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                         Project Level                            │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Project: InvoiceAI                                          │ │
│  │ ID: 2da7e7e6-0f8c-4f33-9ae8-f3dfb6f0ec8a                   │ │
│  │ Status: active                                              │ │
│  └────────────────────────────────────────────────────────────┘ │
│                           │                                      │
│         ┌─────────────────┼──────────────────┐                   │
│         ▼                 ▼                  ▼                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐        │
│  │ Agent:       │  │ Agent:       │  │ Milestone:       │        │
│  │ Frontend     │  │ Backend      │  │ "Invoices live"  │        │
│  │ Role:        │  │ Role:        │  │ Status: in       │        │
│  │ frontend     │  │ backend      │  │ progress         │        │
│  │ Status:      │  │ Status:      │  └──────────────────┘        │
│  │ active       │  │ idle         │                             │
│  └────────┬─────┘  └────────┬─────┘                             │
│           │                 │                                    │
│  ┌────────┴─────────────────┴──────┐   ┌────────────────────┐   │
│  ▼               ▼                ▼   │ project_repos:     │   │
│ ┌──────┐    ┌──────┐    ┌──────┐     │  repo URL, branch, │   │
│ │Task 1│    │Task 2│    │Task 3│     │  role, owning agent│   │
│ │UI    │    │Forms │    │API   │     └────────────────────┘   │
│ └──┬───┘    └──┬───┘    └──┬───┘                             │
│    │           │           │                                 │
│    ▼           ▼           ▼                                 │
│ ┌──────────────────────────────────────────────────────┐      │
│ │              Context (searchable)                     │      │
│ │  • Implementation notes                               │      │
│ │  • Code patterns                                      │      │
│ │  • API contracts                                      │      │
│ │  • Full-text search over the tsvector index           │      │
│ │  • Plus project-wide global_contexts                  │      │
│ └──────────────────────────────────────────────────────┘      │
│                                                                 │
│ ┌──────────────────────────────────────────────────────┐      │
│ │  people  →  api_keys  →  Principal on every request    │      │
│ │  InvoiceAI-Frontend (service) ── key, scopes          │      │
│ │  InvoiceAI-Backend  (service) ── key, scopes          │      │
│ └──────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────┘
```

## Task Dependencies Example

```
InvoiceAI Project Timeline

Backend Tasks (InvoiceAI-Backend)
├─ [✓] Setup Database Schema
├─ [✓] Create Invoice API Endpoints
│   └─ POST   /api/invoices
│   └─ GET    /api/invoices
│   └─ GET    /api/invoices/:id
│   └─ PUT    /api/invoices/:id
│   └─ DELETE /api/invoices/:id
└─ [⏳] Add Invoice Validation

Frontend Tasks (InvoiceAI-Frontend)
├─ [⏸️] Set up the Nuxt frontend (pending backend API)
│   ↑ depends on: "Create Invoice API Endpoints"
├─ [⏸️] Build Invoice List Component (pending backend API)
│   ↑ depends on: "Create Invoice API Endpoints"
├─ [⏸️] Build Invoice Form (pending backend API)
│   ↑ depends on: "Create Invoice API Endpoints"
└─ [ ] Add Invoice Export Feature

Legend:
[✓] Done
[⏳] In Progress
[⏸️] Blocked (waiting on dependency)
[ ] Pending
```

## Real-World Example: Multi-Agent Session

```
Time: 10:00 AM
┌─────────────────────────────────────────────────────────────────┐
│ Backend Agent (InvoiceAI-Backend)                               │
│ $ Get-MyTasks                                                   │
│ → Task: "Create Invoice REST API" [in_progress]                │
│                                                                  │
│ [Working with Copilot to generate API endpoints...]              │
└─────────────────────────────────────────────────────────────────┘

Time: 10:30 AM
┌─────────────────────────────────────────────────────────────────┐
│ Backend Agent completes API                                      │
│ $ Update-TaskStatus -TaskId "xxx" -Status "done"                │
│ $ Add-TaskContext -TaskId "xxx" -Context "API Documentation..." │
│                                                                  │
│ ✓ WebSocket notification sent to all agents                     │
└─────────────────────────────────────────────────────────────────┘
                           │
                           ▼ (Real-time notification)
┌─────────────────────────────────────────────────────────────────┐
│ Frontend Agent (InvoiceAI-Frontend)                              │
│ 🔔 Notification: "InvoiceAI-Backend completed Create Invoice API"│
│                                                                  │
│ $ Get-MyTasks                                                   │
│ → Task: "Build Invoice List" [pending] ✓ Can now start!         │
│                                                                  │
│ $ Update-TaskStatus -TaskId "yyy" -Status "in_progress"         │
│ [Starts working with Copilot using API documentation...]         │
└─────────────────────────────────────────────────────────────────┘

Time: 11:00 AM
┌─────────────────────────────────────────────────────────────────┐
│ Frontend Agent                                                   │
│ $ Add-TaskContext -TaskId "yyy" -Context "UI Components..."     │
│ $ Update-TaskStatus -TaskId "yyy" -Status "done"                │
│                                                                  │
│ Both agents have shared knowledge base!                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Benefits of MCP Architecture

1. **Coordination**: Agents know what others are working on
2. **Context Sharing**: Documentation shared between agents, searchable without
   scraping
3. **Dependency Management**: Tasks blocked until dependencies complete
4. **Real-time Updates**: WebSocket notifications for collaboration
5. **Accountability**: Keys resolve to a named principal, so it is possible to
   answer who did what and when
6. **Knowledge Base**: Accumulated context helps onboarding new agents

---

## Related Documents

- [API.md](./API.md) — REST and MCP endpoint reference
- [AUTHENTICATION.md](./AUTHENTICATION.md) — credential paths and scopes
- [MCP_SETUP.md](./MCP_SETUP.md) — connecting an agent to the server
- [MIGRATIONS.md](./MIGRATIONS.md) — how the schema evolves
- [DEPLOYMENT.md](./DEPLOYMENT.md) — compose topology and the TLS profile
- [AGENT_SETUP_GUIDE.md](./AGENT_SETUP_GUIDE.md) — agent onboarding walkthrough
