# Agent Shaker — Roadmap

Agent Shaker is a coordination plane: a self-hosted store of shared truth where AI
agents and humans read and write the same projects, agents, tasks, contexts,
milestones and standups through MCP and A2A, over the same API. It deliberately
does not execute agents and is not a framework.

This roadmap describes what the product needs next, in dependency order, as of
`main` @ `69d064f` (2026-10-01; `develop`, `main` and `feat/vnext` all point at
this commit, and GitHub shows 1 star). Every task below is scoped to specific
files and has a verifiable acceptance criterion. Nothing here is implemented
yet; the task IDs are stable and meant to be copied into issues without
rewriting.

## Where this picks up

Four facts about the current build frame everything that follows.

- The MCP server answers `protocolVersion: "2024-11-05"`, the first revision of
  the specification. The current revision is `2026-07-28`, three major revisions
  later, and it removed the session model the server still depends on.
- A2A tasks do not execute. `cmd/server/main.go` builds the task manager with a
  nil executor, and with a nil executor the manager echoes the incoming message
  back and marks the task completed. A remote agent delegating work receives a
  success signal for work that was never performed.
- Exactly one span exists in the codebase (`websocket.handle` in
  `internal/websocket/hub.go`). The OpenTelemetry provider, the W3C propagator
  and the otelslog bridge are all wired correctly, so the service is
  instrumentation-ready and uninstrumented.
- The schema has ten tables and none of them records who acted, what a human
  approved, or what it cost. There is no audit journal, no human principal, no
  approvals, no usage.

The rest of this document is the work that closes those gaps in an order where
each step is cheaper after the one before it.

## Milestones at a glance

| Milestone | Goal | Depends on | Size | Blocked by |
|---|---|---|---|---|
| M0 | Prerequisites, no code | — | days | — |
| M1 | Retire the A2A stub | M0 | days | open decision 1 |
| M2 | Humans as a data subject | M1 | weeks | open decision 2 |
| M3 | Append-only event journal | M2 | weeks | — |
| M4 | Approvals | M3 | weeks | — |
| M5 | Target MCP revision 2026-07-28 | M3 | weeks | open decision 6 |
| M6 | Observability | M3 | days | — |
| M7 | Context retrieval | M3 | days | — |
| M8 | Cost tracking | M3 | days | open decision 3 |
| M9 | Distribution | M1–M8 | days + ops | open decision 5 |

M5, M6 and M7 are independent of each other and can run in parallel once M3
lands. M9 comes last on purpose: being discovered by an agent host and handing
that agent a server that reports synthetic success is worse than not being
discovered at all.

## M0 — Prerequisites

Not code. These are the items that must be settled before milestone work is
worth starting.

| ID | Task | Acceptance |
|---|---|---|
| T0.1 | Merge PR #42 (`develop` → `main`) — **done**, `main` is at `69d064f` | Verified 2026-10-02 |
| T0.2 | Decide on the public deployment: domain, DNS, secrets | `PUBLIC_HOST` is fixed, or there is an explicit decision not to run a public instance |
| T0.3 | Create GitHub milestones M1–M9 and move the task IDs into them | All nine milestones exist, each with a due date and an issue list |

## M1 — Retire the A2A stub

A product that cannot be trusted about what happened cannot be positioned around
trust. This is the smallest task in the roadmap and the one with the highest
cost of leaving in place.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T1.1 | Implement `TaskExecutor` as an adapter to an external runtime, or delete the stub | `internal/task/executor_*.go`, `internal/task/manager.go` | An implementation exists and is wired in `cmd/server/main.go`, **or** a nil executor yields `StatusFailed` with an error instead of a completed task |
| T1.2 | Remove the `Task received: %s` echo path | `internal/task/manager.go` | No code path completes a task without executing it |
| T1.3 | Test the manager across nil executor, success, error and cancellation | `internal/task/manager_test.go` (new) | Table-driven; the nil-executor case asserts `failed` |
| T1.4 | Rewrite the tautological status assertion | `tests/a2a/integration_test.go` | The assertion depends on an observed outcome, not on the literal `completed` |
| T1.5 | Bring the A2A documentation in line with actual behaviour | `docs/A2A_INTEGRATION.md`, `README.md` | The described delegation path matches the code |

The default resolution is open decision 1: keep the inbound direction and the
agent card, drop outbound delegation. Maintaining somebody else's execution
runtime is not a commitment a single maintainer can keep.

## M2 — Humans as a data subject

Authentication today is a shared-secret scheme, and the code says so plainly.
There is no "who", which makes attribution, permissions and approval authority
impossible to express. `daily_standups` is the one capability no coordination
competitor has, and it is wasted while the data model cannot record a human.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T2.1 | Migration `009`: `people` (id, kind human/service, display_name, email, created_at) | `migrations/009_people.up.sql` | Applies to a virgin database and to one that already holds data |
| T2.2 | Migration `010`: `api_keys` (id, key_hash, principal_type, principal_id, project_id nullable, scopes, revoked_at, last_used_at) | `migrations/010_scoped_keys.up.sql` | A key is issued against a principal and revoked by setting `revoked_at` |
| T2.3 | Models and stores for both entities | `internal/models/people.go`, `internal/models/api_key.go`, `internal/database/queries/people.go`, `internal/database/queries/api_keys.go` | Store methods are covered by tests |
| T2.4 | Resolve a principal in middleware, preserving the flat-key configuration | `internal/middleware/auth.go` | The existing `API_KEYS` setting yields a synthetic principal; a missing key is 401, a missing scope is 403 |
| T2.5 | Store only the key hash | `internal/database/queries/api_keys.go` | The plaintext key appears neither in the database nor in logs; hash comparison is constant-time |
| T2.6 | Management endpoints | `cmd/server/routes.go` | `GET`/`POST /api/people`, `GET`/`POST`/`DELETE /api/keys` |
| T2.7 | Management UI | `web/app/pages/settings.vue` | A People & Keys section; `make web-typecheck` and `make web-test` pass |
| T2.8 | Extend the authentication tests | `internal/middleware/auth_test.go` | Valid key, revoked key, wrong scope, legacy key |

New columns are added alongside the existing ones and old values are preserved:
`tasks.created_by` keeps referencing `agents(id)` so historical rows stay valid.

## M3 — Append-only event journal

Every capability after this one — audit, cost, approvals, meaningful tracing —
needs a record of what happened. Retrofitting a journal into a system that has
been running is always more expensive than starting it now.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T3.1 | Migration `011`: `audit_events` (id, occurred_at, actor_type, actor_id, project_id, entity_type, entity_id, action, before, after, trace_id, request_id) | `migrations/011_audit_log.up.sql` | Indexes on `(project_id, occurred_at DESC)` and `(entity_type, entity_id)` |
| T3.2 | An `internal/audit` package: `Recorder` interface, Postgres implementation, no-op for tests | `internal/audit/recorder.go`, `internal/audit/postgres.go`, `internal/audit/noop.go` | A write failure is logged through `slog.ErrorContext` rather than panicking |
| T3.3 | Wrap the mutating paths | `internal/handlers/*.go`, `internal/mcp/handler.go` | Every create, update and delete records an event with `before` and `after` |
| T3.4 | Forbid mutation of the journal | `internal/audit` plus review | No `UPDATE` or `DELETE` against `audit_events` exists, and a test pins that |
| T3.5 | Read the journal in the UI | `web/app/pages/activity.vue` (new) | Filter by project and entity; typecheck passes |
| T3.6 | Tests | `internal/audit/recorder_test.go` | Success, database failure, absent actor |

A journal write failing must not fail the business operation. The recorder logs
and returns the error to the caller, which may choose to roll back. This is a
deliberate trade in favour of read availability, recorded here so it is not
later mistaken for an oversight.

## M4 — Approvals

Approvals are table stakes everywhere in the category and are absent here. The
nearest thing in the codebase is a milestone convention, not an entity, and the
UI needs a real record to act on.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T4.1 | Migration `012`: `approvals` (id, project_id, entity_type, entity_id, requested_by_type, requested_by_id, decided_by_type, decided_by_id, state, decision_note, requested_at, decided_at, expires_at) | `migrations/012_approvals.up.sql` | A CHECK constraint on `state` and a partial index on pending rows |
| T4.2 | Model, store and HTTP layer | `internal/models/approval.go`, `internal/database/queries/approvals.go`, `internal/handlers/approvals.go` | CRUD is covered by tests |
| T4.3 | Endpoints | `cmd/server/routes.go` | `POST /api/approvals`, `POST /api/approvals/{id}/decide`, `GET /api/approvals?state=pending` |
| T4.4 | Compare-and-set on decision | `internal/database/queries/approvals.go` | A single `UPDATE ... WHERE state = 'pending' RETURNING`; concurrent decisions yield exactly one winner, the loser gets 409 |
| T4.5 | Push the events over WebSocket | call `hub.BroadcastToProject` from `internal/handlers/approvals.go` | `approval.requested` and `approval.decided` reach the project's clients |
| T4.6 | UI | `web/app/pages/approvals/index.vue`, a card on `web/app/pages/projects/[id].vue` | Three actions: approve, reject, edit-with-note |
| T4.7 | Agent-facing tools | `internal/mcp/handler.go` | `request_approval` and `list_pending_approvals`, fully annotated |
| T4.8 | Tests | `internal/database/queries/approvals_test.go` | A race on `decide`, expiry by `expires_at`, and a repeated decision as a no-op |

The protocol offers a cheaper path to the same outcome. MCP Apps
(`2026-07-28`, SEP-1865) lets a server render an interactive approval card inside
the agent's own host, and every UI action travels the same JSON-RPC path — and
therefore the same audit and consent path — as a direct tool call. That is a
stronger fit for the "one shared truth" thesis than a browser dashboard, and it
belongs in M5.

## M5 — Target MCP revision 2026-07-28

The current revision is the largest change to the protocol since launch. It is
also, for this product, unusually well-timed: the reworked transport is what
makes the journal, the approval card and the trace story possible at all.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T5.1 | Remove `Mcp-Session-Id` and the `initialize`/`initialized` handlers, add `server/discover` | `internal/mcp/handler.go` | A request with no handshake is served; `server/discover` returns capabilities |
| T5.2 | Require `Mcp-Method` and `Mcp-Name`, reject a mismatch with the body | `internal/mcp/handler.go` | A mismatch is a protocol error, not a silent execution |
| T5.3 | Send `MCP-Protocol-Version` in responses and read `clientInfo` from `_meta` on every request | `internal/mcp/handler.go` | The hardcoded `2024-11-05` is gone from the source |
| T5.4 | Return `ttlMs` and `cacheScope` from `tools/list` | `internal/mcp/handler.go` | A client can cache the tool list for a known interval |
| T5.5 | `outputSchema` and `structuredContent` for the `list_*` and `get_*` tools | `internal/mcp/handler.go` | Full JSON Schema 2020-12; `structuredContent` is not restricted to objects |
| T5.6 | `InputRequiredResult` (SEP-2322), or an explicit and correct error in its place | `internal/mcp/handler.go` | The behaviour is documented and covered by a test |
| T5.7 | `resources/list` and `prompts/list` return empty lists | `internal/mcp/handler.go` | Not "method not found" — a hard requirement of every server directory |
| T5.8 | `title`, `readOnlyHint`, `destructiveHint`, `openWorldHint` on all 24 tools | `internal/mcp/handler.go` | A test asserts every tool carries the annotations |
| T5.9 | Parse `traceparent` from `_meta` | `internal/mcp/handler.go` | The incoming trace is linked to the tool-call span |
| T5.10 | Contract tests | `internal/mcp/handler_test.go` (new) | Stateless request, headers, structured output, annotations |

MCP server-side Tasks is worth evaluating here too: the server may return a task
handle from `tools/call` and the client drives it with `get`/`update`/`cancel`.
That is the native way to express a long operation, and it is the honest
successor to the A2A task path removed in M1.

The 63 `gen_ai.*` attribute keys are all still at Development stability, so none
of them appear in this plan. Where GenAI semantics are wanted, they are modelled
on a private `shaker.*` schema over the stable OTel core.

## M6 — Observability

The infrastructure is already correct and unused. This milestone is
instrumentation, not a new dependency.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T6.1 | Wrap the external HTTP handlers | `cmd/server/routes.go` | Every inbound request opens a server span |
| T6.2 | Span the database calls | `internal/database/queries/*.go` | `span.RecordError` on failure; context threaded through `QueryContext` |
| T6.3 | Span `tools/call` under a private schema | `internal/mcp/handler.go` | Attributes prefixed `shaker.*`; zero `gen_ai.*` attributes |
| T6.4 | Move to context-aware logging | `internal/**/*.go` | `slog.*Context(ctx, ...)` wherever a context is in scope |
| T6.5 | Audit metric cardinality | `internal/observability/metrics.go` | No unbounded label values; histograms rather than summaries |
| T6.6 | Attach exemplars to latency histograms | `internal/observability/metrics.go` | A latency spike links to the trace that caused it |
| T6.7 | Put the PromQL query above each declaration | `internal/observability/metrics.go` | Every metric carries its query as a comment |
| T6.8 | Span tests | `internal/observability/tracing_test.go` (new) | A `tracetest` exporter confirms span presence and status |

`OTEL_TRACES_SAMPLER` stays the single sampling switch: full rate on the demo
instance, reduced in production.

## M7 — Context retrieval

The product promises that an agent gets fresh context without guessing or
scraping. With `list_contexts` as the only read path and no full-text index, the
agent has to pull the list and read everything to find anything.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T7.1 | Migration `013`: GIN index over `to_tsvector('simple', title || ' ' || content)`, maintained from the write path | `migrations/013_context_search.up.sql` | Search returns relevant rows against real data |
| T7.2 | `Search` in the store | `internal/database/queries/contexts.go` | Scoped by project, limited, ranked |
| T7.3 | A `search_contexts` tool | `internal/mcp/handler.go` | Schema, annotations and `structuredContent` |
| T7.4 | `GET /api/contexts/search` | `cmd/server/routes.go`, `internal/handlers/contexts.go` | `q`, `project_id` and `limit` are validated |
| T7.5 | Search in the UI | `web/app/pages/projects/[id].vue` | Finding a context no longer loads all of them |
| T7.6 | Tests | `internal/database/queries/contexts_test.go` | Empty query, non-matching query, limit |

PostgreSQL full-text search on the `simple` configuration is the default choice.
There is no ratified standard for agent memory to align an embedding scheme
with, and embeddings would drag in a model and a vector store for a corpus that
is measured in markdown documents.

## M8 — Cost tracking

Budgets and per-agent cost are what the mature players in this category
converge on, and the existing per-agent and per-task entities make it cheap to
add here.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T8.1 | Migration `014`: usage records | `migrations/014_agent_usage.up.sql` | Aggregates available by agent, project and task |
| T8.2 | An opt-in `report_usage` tool | `internal/mcp/handler.go` | The agent reports usage; the server never infers it |
| T8.3 | Export aggregates to Prometheus | `internal/observability/metrics.go` | Bounded cardinality: `agent_id` and `project_id` only |
| T8.4 | Reporting UI | `web/app/pages/dashboard` or `settings.vue` | Breakdown by agent and period |
| T8.5 | Tests | `internal/observability/metrics_test.go` | Reported usage reaches the aggregates correctly |

Usage is opt-in. No data means no data, never zero — otherwise a partial
reporting regime produces a false picture of agent cost, which is worse than no
picture.

## M9 — Distribution

This milestone is not a checklist. It is a decision, because the current
deployment model and the discovery channel are structurally incompatible.

The official MCP registry is the only route to discoverability, and it accepts
only servers whose installation method is publicly available or whose endpoint
is publicly reachable. It explicitly does not accept private servers — those
reachable by a narrow set of users. Agent Shaker is a self-hosted service
behind basic auth with per-instance API keys, so as built it cannot be listed
at all.

Either the project runs a public demo instance over HTTPS and opens that
channel, or it remains a tool for people who already know it exists. The second
option is a legitimate choice, but it should be chosen rather than discovered.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| T9.1 | Run a public demo instance | deployment configuration | `PUBLIC_HOST` resolves to a reachable instance |
| T9.2 | Label the image | `Dockerfile` | `LABEL io.modelcontextprotocol.server.name="io.github.techbuzzz/agent-shaker"` |
| T9.3 | Publish to the official registry | `server.json`, then `mcp-publisher validate` / `login github` / `publish` | The registry API returns the record |
| T9.4 | Ship an install manifest | `llms-install.md` in the repository root | The file exists and carries installation instructions |
| T9.5 | Homepage backlink | `README.md` | The first line links to the project |
| T9.6 | Realign the migration help text | `Makefile`, `docs/MIGRATIONS.md` | `MIGRATION_VERSION` matches the latest migration |
| T9.7 | Correct the advertised tool count | `README.md` | The real number is stated: 24 tools plus `ping` |

Listing has a second tier worth doing even without a demo instance: submissions
to the major vendor directories, which additionally require a privacy policy URL
and, for one of them, a reviewed organisation account.

## Out of scope, deliberately

- **Runtime adapters and agent execution.** The two established projects in this
  category both ship them. They are a support liability across every vendor's
  API, and declining them is a position worth stating publicly rather than
  leaving to be inferred.
- **Comparison with execution frameworks.** Agent Shaker does not run agents, so
  counting orchestration patterns against LangGraph or the Microsoft Agent
  Framework compares different layers.
- **A full Git integration layer with webhooks and synchronisation.** That
  turns the product into a Git tool. If it happens, it happens as a link on the
  existing `project_repos`, not as a data pipeline.
- **Multi-tenancy and billing.** Single-tenant self-hosting stays the product.
- **Onboarding material, templates, tutorials.** Worthless before discovery
  exists; revisit after M9.

## Open decisions

Six forks need an owner. The defaults below are what this roadmap assumes
unless told otherwise.

1. **Outbound A2A delegation** — blocks M1. Either implement `TaskExecutor` as
   an HTTP adapter to an external runtime, or keep the inbound direction and the
   agent card and drop outbound delegation. *Default: drop it.* It is more
   honest than a synthetic success and removes an obligation to maintain
   somebody else's execution layer.
2. **Key compatibility** — blocks M2. *Default:* the existing `API_KEYS` setting
   keeps working as a synthetic principal with full rights, and the move to
   scoped keys is documented without a deadline.
3. **Source of cost data** — blocks M8. *Default:* an opt-in `report_usage` tool
   rather than deriving usage from traces.
4. **Context search mechanism** — *Default:* PostgreSQL full-text search, no
   embeddings, for the reason given in M7.
5. **Public demo instance** — blocks M9. Needs a domain, DNS and secrets from the
   owner.
6. **Transition mode for 2026-07-28** — whether the old session-based semantics
   are accepted for a while. *Default: no.* A clean cut is acceptable because the
   twelve-month deprecation window applies to clients regardless.
