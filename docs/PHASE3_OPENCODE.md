# Phase 3 — OpenCode deep integration, and the hook layer that goes with it

Status: design, not started. Branch baseline: `feat/coordinate-plan`.
Supersedes the one-line Phase 3 entry in [ROADMAP.md](./ROADMAP.md) ("npm package,
compaction hook, heartbeat on session lifecycle").

That one-liner was written on 2026-10-01. It is still roughly right, and it is
missing the thing that matters most: **the host that is being integrated into has
changed shape since it was written, and the plugin API it now exposes is the real
integration surface — not the config file.** This document rebuilds Phase 3
against the code as it stands and against the platform as it stands in late
October 2026.

---

## 1. Scope

Phase 3 is two halves that meet in the middle.

**Part A — the OpenCode client.** A published npm package that installs into
OpenCode and does three things: it turns an OpenCode *session* into a live
Agent Shaker *agent*, it survives context compaction by pushing the durable
summary back into shared truth, and it enforces a local policy gate at the
moment a tool is about to run.

**Part B — the server hook layer.** A subsystem where a task-lifecycle event
runs a command on the server. The reference executor is a headless agent CLI,
`mcode exec`, which means the server can call an agent — a reviewer, a
standup writer, a verifier — at exactly the moment a task changes state, with
no agent in the loop asking for it.

The two halves are deliberately asymmetric:

| | Part A (plugin) | Part B (hooks) |
|---|---|---|
| Runs | inside the agent's host | on the server |
| Trigger | the agent's own session lifecycle | a domain state change |
| Enforcement | preventive, synchronous — it can *block* a tool call | reactive, asynchronous — it can only *react* |
| Needs the agent | yes | no |

A hook never blocks the write that caused it. A plugin gate never runs a
command. If a rule is preventive, it belongs in Part A; if it is a reaction to
work already done, it belongs in Part B.

### What Phase 3 is not

- It is **not** an agent runtime. Agent Shaker does not execute agents, and
  Phase 3 does not change that.
- It is **not** a Claude Code bridge. Phase 4 owns the Copilot bridge and the
  `agsh` CLI. The hook layer is host-neutral on purpose — the first executor
  happens to be a MiniMax command, but nothing in the design knows that.
- It does **not** wait for M5 (the stateless MCP transport rewrite). See §7.

---

## 2. Why the plan changed since 2026-10-01

| Assumption on 2026-10-01 | Reality in late October 2026 | Consequence |
|---|---|---|
| An OpenCode integration means a generated `opencode.json` | The generated `opencode.json` already exists (`internal/configgen/configgen.go:181-192`) and already works | Part A is **not** about config generation. It is about a plugin package that config generation only *installs*. |
| OpenCode has one plugin API | OpenCode has **two**: a v1 object-returning plugin API and a v2 `Plugin.define({ id, setup(ctx) })` API with domain-scoped hooks | The package must export both from one entry point (§4.1). |
| Compaction is a hook you add | v1 has `experimental.session.compacting`; v2 has `ctx.session.hook("compaction")` | The bridge is now a first-class capability, and its API differs per major version. |
| The interesting host surface is MCP | OpenCode v2 exposes `ctx.session`, `ctx.tool`, `ctx.permission`, `ctx.mcp`, `ctx.storage`, `ctx.options`, `ctx.event` | The plugin can do things no MCP server can: inject context into a session, block a tool call before it runs. |
| The server needs a session entity to track OpenCode sessions | It does not — the credential is already bound to one agent (`api_keys.agent_id`, migration 017) | The core lifecycle bridge needs **no new server tables**. That is the single most useful finding in this document. |
| Hooks would be a client-side concern | The server already broadcasts 27 state-change events over WebSocket; nothing consumes them as commands | The hook layer is a **server-side** subscriber to an event source that already exists (§5.2). |
| The plugin observes the host | OpenCode V2's `compaction` hook lets a plugin **author** the compaction summary (`SessionCompactionResult.summary`). No other host exposes that write path | The plugin stops watching and starts *producing* shared truth into the model's context window. This is the strongest differentiator available in Phase 3 (§4.5). |
| The OpenCode MCP config has one shape | V2 nests servers under `mcp.servers`; V1 used `mcp`. V2 also exposes a `protocol` selector (`legacy` / `auto` / `2026-07-28`) | The generator emits the V1 shape today. Phase 3 must emit a config a V2 host actually reads, and say which protocol revision it wants (§4.9). |
| OpenCode is a host, like Claude Code or Codex | OpenCode V2 ships a hosted **Console**: workspaces, SSO/SCIM, budgets as policy rules with HTTP 429 on breach, team-wide policies with final authority over local configuration, and Harnesses that run Claude Code, Codex, Hermes, Pi and Raycast | The integration target is becoming the competitor. Phase 3 must be the vendor-neutral layer that such a console assumes exists — not a second console (§3.5). |

---

## 3. Trend inputs

Primary sources, checked 2026-10-24. The in-repo market scan
(`docs/research/agent-shaker-community-ideas-2026-09.md`) covers the
competitive landscape; this section covers the platform.

### 3.1 Protocol

- **MCP 2026-07-28 is a stateless core.** The `initialize`/`initialized`
  handshake and the `Mcp-Session-Id` header are gone; `server/discover` is
  mandatory; `Mcp-Method` and `Mcp-Name` headers are required on Streamable
  HTTP; list results carry `ttlMs` and `cacheScope`; `_meta` carries
  `io.modelcontextprotocol/protocolVersion`, `clientCapabilities` and
  `clientInfo`; tool results may carry `structuredContent` against an
  `outputSchema`. Sources: MCP specification 2026-07-28; SEP-2243, SEP-2549,
  SEP-2567, SEP-2575.
- **MCP Apps (SEP-1865) is stable since 2026-01-26** — `ui://` resources,
  `_meta.ui.resourceUri` on tools, `text/html;profile=mcp-app`, sandboxed
  iframes, negotiated through the `extensions` capability map under
  `io.modelcontextprotocol/ui`. Source: `modelcontextprotocol/ext-apps`,
  `specification/2026-01-26/apps.mdx`.
- **MCP Apps host support does not include OpenCode.** The documented hosts are
  Claude (web and desktop — *not* Claude Code), ChatGPT, VS Code with Copilot,
  Goose and Postman. Source: MCP Apps overview, mcp-framework.com.
  **Consequence:** an approval or status UI rendered as an MCP App will not
  appear in OpenCode. Part A's human-facing surface uses OpenCode's own TUI
  plugin API (`@opencode/plugin/tui`) instead. This is a design constraint, not
  a preference.
- **A2A v1.0 is the stable line** — agent card at
  `/.well-known/agent-card.json`, `A2A-Version` request header, signed agent
  cards (JWS over RFC 8785 canonical JSON), `supportedInterfaces[]` carrying
  `protocolBinding` and `protocolVersion`, extension negotiation with
  `ExtensionSupportRequiredError`. Sources: a2a-protocol.org v1.0 / v1.0.1
  specification; `a2aproject/A2A` `docs/specification.md`.
- **The server's position today:** it serves `2024-11-05` by default and accepts
  `2024-11-05`, `2025-03-26`, `2025-06-18`, `2025-11-25`
  (`internal/mcp/protocol.go:25`, `:30-35`). `2026-07-28` is named in the file's
  own comment but is **not** in the accepted set, so a client that asks for it
  gets HTTP 400 and `-32022`. `server/discover` already exists
  (`internal/mcp/handler.go:509-514`, `:541`). `Mcp-Method`/`Mcp-Name`
  validation, `MCP-Protocol-Version` in responses, `outputSchema` /
  `structuredContent` and `ttlMs`/`cacheScope` do **not** exist.

### 3.2 The host

- **OpenCode v2 exists and is the current line** (`opencode.ai/v2` banner on the
  v1 docs, last updated 2026-10-23). Its plugin API is `Plugin.define` with a
  stable `id` and a `setup(ctx)` that registers hooks, transforms, tools and
  subscriptions through a **domain-scoped context**: `ctx.session.hook(...)`,
  `ctx.tool.hook(...)`, `ctx.tool.transform(...)`, `ctx.event.subscribe()`,
  `ctx.storage`, `ctx.options`, `ctx.location`. Source: `opencode.ai/v2/docs/build/plugins`,
  `.../migrate-v1/`.
- **One package can serve both.** A default export may expose `setup()` for v2
  and `server()` for v1; v1 object entrypoints are supported from OpenCode
  **1.18.29** onward. The npm scopes differ: v1 is `@opencode-ai/plugin` and
  `@opencode-ai/sdk`, v2 is `@opencode/plugin` (plus `@opencode/plugin/tui`).
- **v2 unpinned npm plugins auto-refresh on every startup**, so a published
  plugin must pin exact versions. Source: `opencode.ai/v2/docs/build/plugins`.
- **The v1→v2 migration is not mechanical.** The docs are explicit that v1
  plugin implementations do not run in v2 and that the APIs remain separate; the
  dual export "does not translate hooks automatically". So Phase 3 maintains two
  thin adapters over one shared core (§4.2).
- **The v2 session hook kinds are enumerated**, and they are not
  interchangeable: `prompt` (before durable prompt admission), `context` (the
  agent loop, including tool-driven continuations), `compaction` (checkpoint
  summaries), `generate` (transient calls), `title` (title generation),
  `model.request` (per-provider request settings), plus `http.request` and
  `http.response`. `compaction` and `title` accept a `result` that skips the
  model call entirely. Source: `opencode.ai/v2/docs/build/plugins`.
- **v2 events are a typed stream, not a hook key**: `ctx.event.subscribe()`
  returns an `AsyncIterable` of events carrying `event.type` and a typed
  `event.data`. The session events include `session.created`, `session.idle`,
  `session.status`, `session.deleted`, `session.compaction.started` /
  `.ended` / `.failed`, `session.execution.*`, and `permission.asked` /
  `permission.replied`. Sources: `opencode.ai/v2/docs/build/plugins`;
  `devcxl/opencode-session-peers` research notes on the v2 event union.
- **OpenCode CLI headless forms exist**: `opencode run [message..] --format json`
  (raw JSON event stream), `opencode serve` for a long-lived server, and
  `--attach` to reuse one. Source: `opencode.ai/docs/cli/`.

### 3.3 The hook model

- **Claude Code's hooks are the de-facto standard for the shape**, and the
  repo already carries a full reference scan at `.mavis-scratch/scan/cc-hooks.md`:
  a hook event, a matcher group, and one or more handlers; command handlers
  receive JSON on stdin; exit 0 means "no decision", exit 2 is a blocking error,
  and a handler may return a JSON decision. Thirty-three event names are
  documented, with matcher semantics that differ per event.
- **The distinction that matters: a hook is a deterministic control, not
  advice.** The research this phase is derived from (the AI-native SDLC
  analysis) lands on the same conclusion from the other direction — skills
  advise, hooks decide. A hook that can *block* is a control point; a hook that
  only reports is telemetry.
- **Headless agent CLIs have become good hook targets.** `mcode exec` reads a
  prompt from stdin, returns one stable `ExecResult` JSON document, and has
  documented exit codes (0 success, 2 invocation error, 3 configuration error,
  4 runtime or task failure, 6 timeout, 7 limit exceeded, 70 internal error, 130
  cancelled, 141 broken pipe). Source: `agent.minimax.io/docs/cli/automation.md`.
  `opencode run --format json` gives an event stream instead of a fixed shape,
  and `claude -p` gives another shape again — so a hook runner must not assume
  one.

### 3.4 What this means for the shape of Phase 3

1. The plugin, not the config file, is the integration.
2. The hook layer belongs on the server, consuming events the server already
   emits.
3. The plugin must speak the revision the server speaks today and adapt when
   M5 lands — not wait for M5.
4. A human-facing UI in OpenCode must be built with OpenCode's TUI API, because
   MCP Apps will not render there.

### 3.5 The category consolidated, and the host became a control plane

This is the finding that changes Phase 3's positioning, and it was not in the
one-line brief.

**The host market consolidated around four names.** OpenCode (v1.19.4,
2026-10-22), Claude Code (v2.1.310, 2026-10-23), Codex and Copilot CLI. Roo
Code was shut down and archived on 2026-05-15; Aider's last release was
2025-08-09; Continue.dev was acquired by Cursor. Every survivor exposes the
same three extension points in the same order: **hooks, then plugins, then
MCP.** That order is the design lesson — the integration that matters binds to
the hook surface first, because that is the only place where a host lets an
outsider *decide* something rather than merely observe it.

**OpenCode V2 is no longer only a host.** It ships a hosted **Console** whose
feature list reads like Agent Shaker's own roadmap: workspaces and shared
access, usage tracking and budget controls, team-wide policies, and
"Harnesses for using Console models from Claude Code, Codex, and other
agents". Three specifics matter:

- *Console policies have final authority.* They are appended after every
  authored statement and apply after agent rules and saved approvals, so an
  organisation-level `allow` lifts a lower-authority `deny`. The matching
  model is explicitly last-one-wins with no specificity rule.
- *Budgets are policy rules with an enforcement point.* A request must fit
  under every applicable budget; a breach produces HTTP 429, checked when the
  request starts so in-flight work can overshoot slightly. The blocking
  policy is recoverable from an activity log.
- *Console Harnesses host other agents' runtimes* — Claude Code, Codex,
  Hermes, Pi, Raycast — behind one API key and one workspace.

**The competitor on the other side is Paperclip** (MIT, 106,790 stars), which
takes the opposite approach: it *executes* agents through a DB-backed wakeup
queue with coalescing, budget checks, secret injection and adapter
invocation, and it ships its own plugin SDK. Its positioning line is the
sharpest summary of the category: *"Bring Your Own Agent — any agent, any
runtime, one org chart. If it can receive a heartbeat, it's hired."*

**Where the two products actually meet is the heartbeat.** Agent Shaker's
heartbeat is a *lease on a worker slot*; Paperclip's is a *wakeup trigger that
drives an agent to do work*. Both assume the agent will report back. The 2026
pattern across both OpenCode Console and Paperclip is the same sentence: **a
console that hosts other agents' runtimes.**

**What this means for Phase 3.** The honest reading is that Agent Shaker's
defensible position in late 2026 is *"the vendor-neutral, self-hosted
shared-truth layer for agents"* — cross-vendor agent coverage, a
protocol-neutral contract (A2A + MCP) instead of harness-specific adapters,
agent-identity-scoped state, an append-only activity model, and self-hosting
as a first-class property. OpenCode's Harnesses are a curated list, not an
open protocol. Phase 3 should therefore be built as a **layer that hosts
adopt**, not as a console that competes: the plugin is the multi-agent truth
that Console's single-workspace model lacks, and the hook layer is the paper
trail that server-side policy needs but cannot have, because Console policies
are per-workspace and Agent Shaker's would be per-team across vendors.

The comparison that best shows the remaining edge is OpenCode's own
**References** primitive — named, attachable aliases to a local directory or a
Git checkout, with a description injected into agent instructions:

| | OpenCode References | Agent Shaker shared contexts |
|---|---|---|
| Storage | local directory or Git checkout | server-side, Postgres |
| Scope | per project, per client | cross-agent, cross-project, cross-vendor |
| Retrieval | attach an alias | full-text search over markdown |
| Freshness | background refresh, ≥24 h eligibility | server-side, immediate |
| Identity | none — a path alias | agent + task + project scoping |
| Multi-agent | no — one agent's filesystem | yes |

Both hosts now believe "named external context with machine-readable guidance"
is a first-class primitive. That validates Agent Shaker's feature shape and
narrows the defensible edge to the multi-agent, identity-scoped, searchable,
server-side properties.

Full evidence, with URLs, in
[MARKET_SCAN_2026-10.md](./MARKET_SCAN_2026-10.md).

---

## 4. Part A — the OpenCode plugin

### 4.1 Package shape

**Location:** `plugins/opencode/` — a standalone npm package with its own
`package.json`, `tsconfig.json` and test suite. Not a workspace of the root
`package.json` (that package is `agent-shaker-mcp-bridge`, a hand-rolled stdio
bridge for Copilot — a different distribution with a different consumer), and
not under `web/`.

**Name:** `@techbuzzz/opencode-agent-shaker`. The scope is a decision, not a
fact — see §9 D1. An unscoped `opencode-agent-shaker` is a squatting risk on a
name an outside party will also want.

**License:** MIT, matching the repository (`LICENSE`).

**Pinning:** exact versions for `@opencode/plugin` and `@opencode-ai/plugin`.
Unpinned npm plugins are re-fetched on every startup under v2, which turns a
broken upstream release into an outage with no rollback.

**Entry point:** one file that exports both shapes.

```ts
// plugins/opencode/src/index.ts
import { Plugin } from "@opencode/plugin"

export default {
  ...Plugin.define({
    id: "agent-shaker",
    async setup(ctx) { /* v2 path */ },
  }),
  async server() { /* v1 path */ },
}
```

### 4.2 One core, two adapters

The v1 and v2 APIs are genuinely different and the OpenCode migration guide is
explicit that the dual export does not translate hooks between them. So the
package is three layers, not one:

| Layer | Contents | Depends on the host API? |
|---|---|---|
| `core/` | the Agent Shaker client, the event→call mapping table, the policy rules, the payload shapes | no |
| `adapters/v1.ts` | maps the core's intent onto v1 hook keys (`tool.execute.before`, `experimental.session.compacting`, `event`) | yes |
| `adapters/v2.ts` | maps the same intent onto v2 domain hooks (`ctx.tool.hook`, `ctx.session.hook`, `ctx.event.subscribe`) | yes |

`core/` is pure TypeScript over `fetch` and holds every test worth writing.

The two APIs are not renames of each other. This is the mapping OpenCode's own
migration material gives, and it is what the adapters implement:

| V1 extension point | V2 equivalent |
|---|---|
| `event` (a returned hook key) | `ctx.event.subscribe()` — an `AsyncIterable` of typed events |
| `chat.message` | `ctx.session.hook("prompt")` |
| `chat.params` | `ctx.session.hook("context")` |
| `chat.headers` | `ctx.session.hook("model.request")` |
| `experimental.chat.system.transform` | `ctx.session.hook("context")`, editing `event.system` |
| `experimental.chat.messages.transform` | `ctx.session.hook("context")`, editing `event.messages` |
| `experimental.session.compacting` | `ctx.session.hook("compaction")` |
| `tool.execute.before` / `tool.execute.after` | `ctx.tool.hook("execute.before" / "execute.after")` |
| `permission.asked` | the `ctx.permission` hook |
| — | `ctx.shell.hook("create.before")` — new in v2 |

Three traps in that table:

- **The v2 `prompt` hook runs before durable prompt admission, while `context`
  runs immediately before an agent model request.** They are not
  interchangeable.
- **`context` runs for the agent loop, including tool-driven continuations.**
  `compaction`, `generate` and `title` are separate hooks for the auxiliary
  request kinds. A transformation that must apply to every model request has
  to be registered on all of them.
- **The v1 event names survive into v2 but the shape changes.** v1 delivers
  `event.type`; v2 delivers `event.type` plus a typed `event.data` carrying
  `sessionID`. The compaction event was renamed: v1 has `session.compacted`,
  v2 has `session.compaction.started` / `.ended` / `.failed`.

### 4.3 Capability probe, not assumption

The host's capabilities differ by version and by build. The plugin probes at
load and degrades, rather than throwing and taking the host's plugin loader
down with it:

- If `ctx.session.hook` is absent → v1 adapter, or no adapter at all.
- If the compaction hook is absent → the compaction bridge is disabled and the
  plugin logs one warning naming the capability it lost.
- If `ctx.permission` is absent → the permission gate is skipped.
- **One exception:** if the plugin cannot register the *session lifecycle*
  hooks, that is a contract violation and the plugin fails loudly. A plugin
  that silently stops heartbeating is worse than one that refuses to load.

### 4.4 The event → call mapping

This is the core of Part A. Every row uses tools that **already exist** in the
35-tool surface.

| OpenCode event | v2 hook | Agent Shaker call | Why |
|---|---|---|---|
| `session.created` | `ctx.event.subscribe()` | `send_heartbeat` | The agent becomes live and takes a lease. |
| `session.idle` | event stream | `update_my_status` (`idle`) + `send_heartbeat` | Releases the "working" state; the lease is still held. |
| `session.error` | event stream | `update_my_status` (`blocked`) | The agent is alive but stuck; a human can see it. |
| `session.deleted` | event stream | `update_my_status` (`offline`) | Stops the lease without waiting for the reaper. |
| compaction (pre) | `ctx.session.hook("compaction")`; v1 `experimental.session.compacting` | `add_context` | Pushes the pre-compaction digest into shared truth **before** the host discards it. |
| compaction (post) | `ctx.session.hook("context")` | `search_contexts` / `list_contexts` → `event.system.push(...)` | Re-injects shared project truth into the new, smaller context. |
| `tool.execute.before` | `ctx.tool.hook("execute.before")` | `get_my_tasks` (read) | Policy gate — see §4.6. |
| `tool.execute.after` | `ctx.tool.hook("execute.after")` | `add_context` | Captures verification output as evidence — see §4.7. |
| `permission.asked` | `ctx.permission` | — | **Deferred.** Needs M4 approvals, which do not exist. |

**Heartbeat timing is server-driven, not guessed.** `send_heartbeat` already
returns `lease_ttl_seconds` and `next_heartbeat_before`
(`internal/mcp/coordination_tools.go:215-222`). The plugin schedules its next
beat from that value, so the client follows the server's lease rather than
assuming 60 seconds. The server's default is a 60 s lease and a 30 s reaper
(`internal/coordination/reaper.go:36-38`), overridable through
`AGENT_LEASE_TTL` and `REAPER_INTERVAL` (`cmd/server/main.go:520-547`).

### 4.5 The compaction bridge — the plugin authors the summary

Compaction is where a session's working memory dies. Agent Shaker's premise is
that shared truth survives the session, so the bridge is the feature that makes
the premise true.

OpenCode V2 changed what is possible here, and this is the strongest finding
for Phase 3.

**The compaction hook can supply the summary, not just observe it.** The v2
`compaction` hook receives the transcript and may set
`event.result = { summary, native?, metadata?, tokens? }`. Setting it records
the compaction and **skips the model call entirely**. No other host in the
ecosystem exposes a write path into its own compaction summary — Claude Code
has `PreCompact` and `PostCompact` but no result, Codex has a `SessionStart`
matcher, and the managed-agent platforms own the harness entirely.

That turns Agent Shaker from a store that *receives* a summary into the
component that *produces* one. The content a host's own summary is supposed to
preserve — pending work, blockers, next moves, the relevant file set — is
precisely what Agent Shaker already stores: the current task, the open
blockers, the dependency state, the shared contexts.

**The contract the plugin implements:**

| | |
|---|---|
| Input | `event.messages` — the transcript being summarised |
| Output | `event.result.summary` — the replacement summary |
| `event.result.metadata` | the coordination identifiers: project, task, agent, session |
| `event.result.tokens` | token accounting, when the host reports it |

The `metadata` slot is the one that matters beyond the text: it carries the
coordination identifiers across a compaction boundary, so a resumed session
knows which task it belongs to without re-deriving it from prose.

**The v1 path is weaker, and the plan says so.** v1's
`experimental.session.compacting` fires before the host generates a summary and
can shape that generation, but it cannot replace the result. So the two
adapters are not equivalent: v2 *authors*, v1 *influences*. The plugin
implements the best available behaviour on each and reports which one is
active through `announce_capability`.

**Compaction in v2 is checkpoint-based, and that is a contract rather than a
threshold.** `compaction.auto` defaults to true; `compaction.keep.tokens`
defaults to 15,000 recent tokens kept *beside* the summary; `compaction.buffer`
defaults to 15 % of the model limit, at least 16k. The plugin can size its
injection against `keep.tokens` and reason about the cost deterministically.

**The summary is written back to shared truth as well.** Whatever the plugin
authors is stored with `add_context` too, so the coordination plane holds the
same artifact the model just saw. That is the difference between a
coordination store and a scratchpad.

**Re-injection after compaction** (v2 `ctx.session.hook("context")`): the
plugin fetches the project's shared context through `search_contexts` /
`list_contexts` and pushes it into the session with
`event.system.push({ type: "text", text })`. The payload is bounded — a hard
character cap and a relevance ordering, not the whole context store.

**Register the injection on three hooks, not one.** OpenCode's own built-in
plugins register the same handler on `context`, `compaction` and `generate`.
A plugin that registers only on `context` produces a compaction request whose
first user message lacks everything the normal turns carried — and on any
backend with prefix caching, that session is re-prefilled cold at the single
most expensive request of its life.

**Cache per turn, not per request.** The `context` hook fires on *every* model
call, including the tool-driven continuations inside one prompt. Fetching
shared context on each of those is a request storm, and it changes nothing
within a turn, because the shared truth does not move while a prompt is being
answered. Fetch once per prompt, reuse for the rest of the turn, clear the
cache on the next prompt.

The one thing the plugin must not do is push a large block of shared truth into
every context build. That is how an integration makes a host slower and gets
switched off.

### 4.6 The policy gate

Two hooks can stop an agent, and Phase 3 uses both.

- `ctx.tool.hook("execute.before")` can block a tool call by throwing.
- `ctx.shell.hook("create.before")` can rewrite the command, its working
  directory, its timeout, its shell and its environment before the command
  runs.

The shell hook is the stronger of the two, because it sees the actual command
line — `git push`, `rm`, a curl to an unknown host — where the tool hook sees
a tool name. A policy that is about *commands* belongs in the shell hook; a
policy that is about *tools* belongs in the tool hook.

The default rule is deliberately small: **a mutating action requires a claimed
task.** The plugin reads the agent's tasks with `get_my_tasks`; if none is in
progress, it throws and the model is told why.

- Opt-in per project. It is a policy, not a default.
- **Fail-open on timeout by default, and the setting must be explicit.** A
  gate that stops work because the server was slow is a gate that gets
  removed. But the opposite failure is worse in a governed project, so the
  choice is made per project rather than inherited by accident:
  `fail_closed: true` makes a timeout block. The lesson comes from Claude
  Code, where a timed-out `PreToolUse` hook silently renders no decision and
  the action proceeds — a gate that must never fail open has to say so, or it
  fails open by accident.
- The read is cached briefly so a burst of tool calls does not become a burst
  of server reads.
- The permission hook is deliberately **not** used in Phase 3. v2 exposes
  `ctx.permission.hook("evaluate")`, where the callback may set `effect` to
  `allow`, `ask` or `deny`, with a message that becomes the denial reason.
  That is exactly the hook an approval workflow needs — and exactly why it
  waits: the server has no approval queue to consult yet (M4).

### 4.7 Evidence capture

`ctx.tool.hook("execute.after")` sees the tool's output. When the tool is a
test or build command, the plugin writes the result into the task's context
through `add_context` — the verification output, recorded where the task can
find it.

This is the concrete answer to the verification gap: an agent that says "done"
without evidence now has its evidence attached to the task by the host, not by
the agent's own good manners.

### 4.8 The human-facing surface

MCP Apps will not render in OpenCode (§3.1). So the plugin's UI uses the
OpenCode TUI plugin API:

- `context.ui.toast.show` for transient state ("claim a task first",
  "context restored after compaction").
- `context.ui.router.register` for a small panel showing the agent's current
  task, its lease countdown, and pending handoffs.

Scope it to a toast and one panel in Phase 3. A full TUI is Phase 5.

### 4.9 Server-side changes Part A requires

Only three, and none of them is on the critical path for the core loop:

1. **`announce_capability` (new MCP tool).** `internal/models/tooltype.go:18-19`
   already names it: *"what an agent is capable of is a separate concern
   (Phase 3's announce_capability)"*. The plugin announces which capabilities it
   can serve — `compaction`, `context_injection`, `policy_gate`,
   `evidence_capture` — plus the host name and version and the negotiated MCP
   revision. Storage: a `capabilities JSONB` column on `agents`
   (migration 021).
   *Why it earns its place:* without it the server cannot know whether an agent
   can accept an injected context or a compaction snapshot, and Phase 5's
   `get_team_status` has nothing to report. The negotiated revision it carries
   is how the plugin learns whether the server is on `2024-11-05` or
   `2026-07-28`.
2. **Tool-registry extraction.** `internal/mcp/handler.go` is 3,285 lines and
   holds **two independent hand-maintained lists** — the tool list literal at
   `:615` and the dispatch switch at `:1397-1481` — that must be edited in
   lockstep. Adding `announce_capability` doubles the edit surface. Split the
   registry into `internal/mcp/tools.go` before adding the tool. (This is a
   refactor of a file the plan already has to touch, and it is the kind that
   pays for itself the second time.)
3. **A generated `opencode.json` that installs the plugin — and that a v2 host
   actually reads.** Three corrections, all from the v2 configuration docs:

   - *The MCP block moved.* v1 nests servers under `mcp`; v2 nests them under
     `mcp.servers`. The generator emits the v1 shape
     (`internal/configgen/configgen.go:181-192`), so a v2 host reading the
     file it expects would find no server in it. The OpenCode format needs a
     version marker to choose between the two shapes.
   - *The plugin key changed.* v1 reads `plugin`, v2 reads `plugins`, and v2
     entries may be objects carrying `options` — which is how the plugin
     receives the server URL, project id and agent id (`ctx.options`).
   - *The protocol revision is selectable.* The v2 MCP configuration exposes
     `protocol: "legacy" | "auto" | "2026-07-28"`. Because the server today
     serves `2024-11-05`, the generated file must not ask for `2026-07-28` —
     a host that did would get HTTP 400 and `-32022`
     (`internal/mcp/protocol.go:48`). `auto` is the correct value until M5
     lands, at which point the generated file names `2026-07-28` and the
     client needs no change.

   See §9 D5.

---

## 5. Part B — the hook layer

### 5.1 The insertion point

The server already broadcasts every meaningful state change to the WebSocket
hub — 27 event names across roughly twenty call sites, including
`task_handoff` (`internal/mcp/coordination_tools.go:370`), `task_ready`
(`internal/coordination/dependency_resolver.go:111`), `blocker_raised`
(`internal/mcp/coordination_tools.go:622`), `agent_gone_stale`
(`internal/coordination/reaper.go:159`), `agent_heartbeat`
(`internal/mcp/coordination_tools.go:204`).

Today those broadcasts are the end of the line. Part B turns them into
commands.

**Design:** introduce `internal/events` — a broker with one method,
`Publish(ctx, Event)`. The WebSocket hub becomes one subscriber; the hook
runner becomes another; the audit writer becomes a third. The twenty-odd direct
`hub.BroadcastToProject(...)` calls become `broker.Publish(...)`.

This is a refactor with a payoff: the hook layer does not need its own
instrumentation of twenty code paths, and the audit journal gets every event
for free.

**Note the naming defect this surfaces:** both `task_update`
(`internal/mcp/handler.go:3450`) and `task_updated`
(`internal/mcp/coordination_tools.go:457`) exist, and `agent_update` coexists
with an `agent_updated` shape. The broker is the right place to normalise the
vocabulary — one canonical event name per domain event, with the old names
kept as aliases for the WebSocket so the existing frontend does not break.

### 5.2 Event catalogue

| Event | Fired when | Source today |
|---|---|---|
| `task.claimed` | a task is assigned to an agent | `executeClaimTask` |
| `task.completed` | a task reaches `done` | `executeCompleteTask` |
| `task.handoff` | a task completes and a successor is created | `coordination_tools.go:370` |
| `task.reassigned` | ownership changes | `executeReassignTask` |
| `task.blocked` | a blocker is raised against a task | `coordination_tools.go:614` |
| `task.ready` | dependencies satisfied | `dependency_resolver.go:111` |
| `blocker.resolved` | a blocker is answered and the task released | `ReleaseAfterBlockerResolution` |
| `agent.stale` | a lease expires | `reaper.go:159` |
| `context.published` | a context is added | `executeAddContext` |

Hook matching uses a **matcher** on the event, in the Claude Code sense: a
regular expression over a named field of the entity. `"matcher": "^\\[API\\]"`
over a task title, or `{"field": "priority", "pattern": "high"}`.

### 5.3 Hook configuration

```json
{
  "id": "3f2c…",
  "project_id": "b91e…",
  "event": "task.completed",
  "matcher": { "field": "title", "pattern": "^\\[API\\]" },
  "command": ["mcode", "exec", "--input", "-", "--input-format", "json",
              "--output-format", "json", "--cwd", "{{project.repo_path}}",
              "--permission", "off", "--timeout", "5m", "--max-steps", "20"],
  "stdin": {
    "mode": "prompt",
    "template": "You are the reviewer for team {{project.name}}. Task {{entity.id}}: {{entity.title}}. Review the completed work and report defects."
  },
  "timeout_seconds": 300,
  "enabled": true,
  "post": { "kind": "context", "title": "Review of {{entity.title}}" }
}
```

Field by field:

- **`command` is an argv array, executed without a shell.** No `sh -c`, no
  string interpolation into a shell command line. This removes the entire
  command-injection class by construction rather than by escaping.
- **`{{...}}` placeholders** are substituted per argv element. A placeholder
  that resolves to a path with a space is still one argv element.
- **`stdin.mode`** is `event` (default: the whole event JSON) or `prompt` (a
  template). `prompt` is what `mcode exec --input-format json` wants; `event`
  is what a plain script wants.
- **`post`** is the feedback channel. The runner writes the hook's JSON stdout
  back into the domain as a context on the task. This is what turns a hook
  from a notification into a **reviewer**: the verdict lands on the task, where
  the agent that did the work will read it.
- **`timeout_seconds`** is enforced by the server, which kills the process
  group. Default 120, capped by configuration.

### 5.4 The payload contract

The runner writes one JSON document to the hook's stdin:

```json
{
  "hook_event_name": "task.completed",
  "event": "task.completed",
  "delivery_id": "9c71…",
  "attempt": 1,
  "occurred_at": "2026-10-24T17:35:00Z",
  "project": { "id": "b91e…", "name": "platform", "repo_path": "/srv/platform" },
  "entity": {
    "type": "task", "id": "2ad4…", "title": "[API] add pagination",
    "status": "done", "priority": "high",
    "assigned_to": "5e10…", "output": "…"
  },
  "actor": {
    "type": "agent", "id": "5e10…", "name": "backend-1",
    "role": "developer", "tool_type": "opencode"
  }
}
```

`hook_event_name` is Claude Code's field name, included so an existing
Claude Code hook script works unchanged against Agent Shaker. `event` is the
alias for consumers that prefer it.

### 5.5 The MiniMax reference executor

This is the case the user asked for, and it is the one that justifies the
whole subsystem.

**Hook A — review the completed work.**

```json
{
  "event": "task.completed",
  "command": ["mcode", "exec", "--input", "-", "--input-format", "json",
              "--output-format", "json", "--cwd", "{{project.repo_path}}",
              "--permission", "off", "--timeout", "5m", "--max-steps", "20"],
  "stdin": {
    "mode": "prompt",
    "template": "Review the completed task {{entity.id}} ({{entity.title}}) in {{project.repo_path}}. Report defects, missing tests and unverified claims. Answer as JSON: {\"verdict\":\"pass|changes\",\"findings\":[...],\"summary\":\"...\"}."
  },
  "post": { "kind": "context", "title": "Review of {{entity.title}}" },
  "timeout_seconds": 300
}
```

`mcode exec` reads the prompt from stdin, writes one `ExecResult` document to
stdout, and exits with a documented code. The `post` step writes that verdict
back as a context on the task, so the workflow is:

> worker finishes a task → the server notices → a reviewer agent runs on the
> server → the verdict lands on the task → the worker reads it on its next
> turn.

**Hook B — a code-review variant**, using the dedicated review entry point:

```json
{
  "event": "task.completed",
  "command": ["mcode", "exec", "review", "--cwd", "{{project.repo_path}}",
              "--output-format", "json", "--effort", "high"],
  "stdin": { "mode": "event" }
}
```

**Exit codes, normalised.** The runner records the exit code verbatim *and* a
normalised status, because the two ecosystems disagree:

| `mcode exec` exit | Meaning | Normalised hook status |
|---|---|---|
| 0 | success | `succeeded` |
| 2 | invocation or input error | `failed` |
| 3 | configuration error | `failed` |
| 4 | runtime or task failure | `failed` |
| 6 | timeout | `timeout` |
| 7 | runtime limit exceeded | `failed` |
| 70 | internal error | `failed` |
| 130 | cancelled | `cancelled` |
| 141 | broken pipe | `failed` |

and, for Claude-Code-style handlers, exit 2 additionally means "blocking
decision". The runner resolves the ambiguity by configuration: a hook declares
`exit_two_is_blocking: true` or it does not. Default `false`, because
`mcode exec` uses 2 for a malformed invocation.

**A prerequisite, stated plainly:** `mcode` is MiniMax Code's public terminal
command, and on the machine this plan was written on it is **not on `PATH`** —
the MiniMax Code desktop application is installed under
`C:\Users\buzin\AppData\Local\Programs\MiniMax Code`, but the CLI shim is not
present. A hook host that runs `mcode exec` must have the CLI installed. This
is a deployment fact, not a design constraint, and the hook layer does not care
which CLI it runs: `opencode run --format json` and `claude -p` are
interchangeable executors behind the same contract.

### 5.6 Execution model

- **Asynchronous by default.** The write that caused the event returns
  immediately; the hook runs on a bounded worker pool. A slow hook must never
  be able to slow an agent down.
- **Bounded concurrency per project** and a global cap, so one project's hook
  cannot starve the server.
- **No retry storm.** Failures are recorded; a retry policy (attempts, backoff)
  is per hook and defaults to no retry.
- **Output capture** is truncated (64 KB stdout, 16 KB stderr) and stored in
  `hook_runs`.
- **Every run is an audit row.** The hook layer and the audit journal are
  built together, not sequentially.

### 5.7 Security

Hooks execute arbitrary commands. This is the highest-risk thing in Phase 3 and
it is RCE by design, in the same class as GitHub Actions or a Claude Code hook.
The controls are not optional:

- **Disabled by default.** A global flag (`AGENT_SHAKER_HOOKS_ENABLED`,
  default off) plus a per-project opt-in.
- **Only humans with the right scope may author hooks.** The `people` table
  already distinguishes `human` from `service` (migration 009) and `api_keys`
  carries scopes (migration 010).
- **Executable allowlist.** A hook names an executable from a per-project
  allowlist. Default deny.
- **No shell, ever.** `exec` with an argv array.
- **Working directory constrained** to a registered `project_repos` path.
- **Secrets by reference.** A hook refers to a secret by name; the server
  injects it at run time from its own store, it is never written into the hook
  row, never logged, and is redacted from captured output.
- **Every run is audited.**

### 5.8 Server-side changes Part B requires

- **Migration 019 — `hooks` and `hook_runs`.**
  `hooks(id, project_id, event, matcher JSONB, command TEXT[], stdin JSONB,
  post JSONB, timeout_seconds, enabled, created_by, created_at)`;
  `hook_runs(id, hook_id, delivery_id, event, entity_type, entity_id,
  status, exit_code, stdout, stderr, duration_ms, started_at, finished_at)`.
- **Migration 022 — `audit_events`.** The append-only journal M3 promises, and
  the first thing both halves of Phase 3 write to. Columns: `id BIGSERIAL`,
  `occurred_at`, `actor_type`, `actor_id`, `project_id`, `entity_type`,
  `entity_id`, `action`, `before JSONB`, `after JSONB`, `trace_id`,
  `request_id`, `prev_hash`, `hash`.

  The number was `011` in the first plan, and it is `022` now because the
  tree moved. `011` and `012` were reserved for M3 and M4 when `011` was the
  next free number; Phase 3 took `019` and `020`, and a migration numbered
  below the tree's maximum would never be applied at all — `golang-migrate`
  walks forward only, so a late `011` would sit in the directory forever
  while every deployment reported itself as migrated. The forward-only rule
  wins over the placeholder.

  The `prev_hash`/`hash` pair makes the chain tamper-**evident**. A hash chain
  is only as strong as its anchor, so the head is published where the operator
  controls it: a file the operator names, and the API that reads it back
  (§9 D6, §11.15).
- **`internal/events/`** — the broker (§5.1).
- **`internal/hooks/`** — config store, matcher, renderer, runner, result
  classifier.
- **REST surface:** `GET`/`POST /api/projects/{id}/hooks`,
  `PATCH`/`DELETE /api/hooks/{id}`, `GET /api/hooks/{id}/runs`,
  `GET /api/projects/{id}/hook-runs`,
  and `POST /api/hooks/{id}/test` — a dry run against a synthetic payload.
  The test endpoint is not a convenience: a hook that can only be validated by
  firing it at real work is a hook nobody will enable.
- **UI:** a Hooks page and a run log in the Nuxt app, following the existing
  page and form patterns.

---

## 6. Task breakdown

Task IDs are stable and meant to be copied into issues without rewriting, in
the roadmap's convention. `P3.<track>.<n>` — track `A` foundation, `B` hook
layer, `C` plugin, `D` server support, `E` distribution and surfaces. Where
the roadmap already owns a task, it is referenced rather than duplicated.

Every task names the files it owns and a verifiable acceptance criterion. The
criterion is what a reviewer checks; it is not a description of the work.

### 6.0 Status

**Landed and verified in Docker:** `P3.A1`, `P3.A3`, `P3.A4`, `P3.A6`,
`P3.B1`–`P3.B12`, `P3.C1`, `P3.C4`, `P3.C5`, `P3.C7`, `P3.E1`, and — in a
second pass — `P3.D1`–`P3.D5`, `P3.E4` and `P3.E5`. The two critical paths
— the hook path `A1 → A3 → B1 → B9 → B11 → E1` and the plugin path
`C1 → C4 → C5 → C7` — are complete, and both are proven end to end against a
throwaway stack: the hook layer runs a real command against a real event,
and the MiniMax Code bridge renews a lease and pushes a compacted transcript
(§11.14).

**Outstanding:** nothing in the decomposition. The audit journal (`P3.A5`,
migration `022_audit_log`) landed with M3's own tasks, and the phase's
evidence layer is the journal rather than a promise of one.

**Deliberately not done:** the row-level change log. T3.3 asks for "every
create, update and delete" to record an event with `before` and `after`; the
journal records the **domain** vocabulary — the changes that affect
coordination. A row-level history of every column of every table is a
different feature, and it is not what a coordination trail is for.

### 6.1 Track A — Foundation: the journal, the broker, idempotency

Milestone M3 in the roadmap already owns the journal (T3.1–T3.6). Phase 3
executes it — the journal is not optional groundwork for the hook layer, it
*is* the hook layer's audit trail — and adds five tasks of its own.

| ID | Task | Files | Acceptance |
|---|---|---|---|
| `P3.A1` | Introduce `internal/events`: an `Event` type, a `Broker` with `Publish`, subscriber registration | `internal/events/broker.go` (new) | A broker with two fake subscribers delivers one event to each; publishing under a cancelled context does not block |
| `P3.A2` | Reconcile M3's T3.3 ("wrap the mutating paths") with the broker: publication goes through `internal/events`, and the journal becomes one of its subscribers rather than a second capture point | `internal/audit/`, `internal/events/` | Every mutating path still records an event with `before` and `after`, and the WebSocket still receives the same payloads — the frontend test suite is unchanged |
| `P3.A3` | Convert the direct hub broadcasts, and normalise the event vocabulary (`task_update`/`task_updated`, `agent_update`/`agent_updated`) | `internal/websocket/hub.go`, `internal/mcp/coordination_tools.go`, `internal/handlers/*.go` | One canonical name per domain event; the old names survive as WebSocket aliases so the existing client keeps working |
| `P3.A4` | Idempotency keys on the mutating MCP tools — `claim_task`, `complete_task` | `migrations/020_idempotency.up.sql`, `internal/mcp/handler.go` | A repeated request carrying the same key returns the first result instead of claiming twice or completing twice; a unique index enforces it |
| `P3.A5` | The journal's hash chain and its anchor | `migrations/022_audit_log.up.sql`, `internal/audit/` | The `prev_hash`/`hash` chain is verified by a test, and the anchor is published where the operator controls it — not stored only in the same database it protects |
| `P3.A6` | Fix the stale comment in `tooltype.go` and confirm the tree is clean before anything else starts | `internal/models/tooltype.go` | The comment matches the switch at `:103`; `make check-all` passes |

### 6.2 Track B — The hook layer

| ID | Task | Files | Acceptance |
|---|---|---|---|
| `P3.B1` | Migration `019`: `hooks` and `hook_runs` | `migrations/019_hooks.up.sql` | Applies to a virgin database and to one that already holds data; indexes on `(project_id, event)` and `(hook_id, started_at DESC)` |
| `P3.B2` | The hook configuration: model, store and CRUD | `internal/models/hook.go` (new), `internal/database/queries/hooks.go` (new) | A hook round-trips through the store; `command` is stored as an array and never as a string |
| `P3.B3` | The matcher: an event plus a regular expression over a named field of the entity | `internal/hooks/matcher.go` (new) | A hook matching `{"field": "title", "pattern": "^\\[API\\]"}` fires only for a matching title; a malformed pattern is rejected at creation, not at fire time |
| `P3.B4` | The renderer: `{{...}}` placeholders substituted per argv element, plus the stdin modes `event` and `prompt` | `internal/hooks/render.go` (new) | A placeholder resolving to a path containing a space stays one argv element; the prompt template receives the entity fields |
| `P3.B5` | The runner: `exec` with an argv array, no shell, process-group kill on timeout, output capture truncated to 64 KB stdout and 16 KB stderr | `internal/hooks/runner.go` (new) | A timeout kills the whole process group, not just the direct child; a hook writing 1 MB stores at most 64 KB |
| `P3.B6` | Result classification: the exit code verbatim plus a normalised `succeeded`/`failed`/`timeout`/`cancelled`, and the `exit_two_is_blocking` rule | `internal/hooks/classify.go` (new) | Exit 0 → `succeeded`; exit 2 → `failed` by default and `blocked` when the hook declares the rule; exit 130 → `cancelled` |
| `P3.B7` | Security: the executable allowlist, the working directory constrained to a registered `project_repos` path, secrets by reference, redaction of captured output | `internal/hooks/security.go` (new) | A hook naming an executable outside the allowlist is rejected at creation; a secret appears in neither the hook row, nor the logs, nor the captured output |
| `P3.B8` | The global off switch and the per-project opt-in | `cmd/server/main.go`, `internal/hooks/` | With `AGENT_SHAKER_HOOKS_ENABLED` unset, no hook runs; with it set, only projects that opted in run hooks |
| `P3.B9` | Wire the broker to the runner: a subscriber that enqueues the matching hooks onto a bounded worker pool, with per-project and global concurrency caps | `internal/events/`, `internal/hooks/dispatcher.go` (new) | The write that caused the event returns before the hook runs; one project's slow hook cannot starve another's |
| `P3.B10` | The `post` step: write the hook's stdout back into the domain as a context on the entity's project | `internal/hooks/post.go` (new) | A hook that prints a JSON verdict produces a context row attached to the task |
| `P3.B11` | The REST surface: CRUD, the run log, and `POST /api/hooks/{id}/test` | `internal/handlers/hooks.go` (new), `cmd/server/routes.go` | The test endpoint runs a hook against a synthetic payload without touching real work |
| `P3.B12` | Tests | `internal/hooks/*_test.go` (new) | Event → argv → exit code → classification is covered end to end; a database failure does not fail the business operation |

### 6.3 Track C — The OpenCode plugin package

| ID | Task | Files | Acceptance |
|---|---|---|---|
| `P3.C1` | The package skeleton: `plugins/opencode/` with `package.json`, `tsconfig.json`, the dual export (`Plugin.define` plus `server()`), and exact version pins for `@opencode/plugin` and `@opencode-ai/plugin` | `plugins/opencode/package.json`, `plugins/opencode/tsconfig.json`, `plugins/opencode/src/index.ts` | The package loads on OpenCode 1.18.29+ and on OpenCode 2.x without an error |
| `P3.C2` | `core/client.ts`: a typed client over the tools that exist today | `plugins/opencode/src/core/client.ts` (new) | Every call the core makes maps to a tool that already exists — the core loop requires no new server surface |
| `P3.C3` | The capability probe | `plugins/opencode/src/core/capabilities.ts` (new) | A missing optional capability degrades to one warning; a failure to register the session-lifecycle hooks fails setup loudly |
| `P3.C4` | The event mapping: one core, two adapters | `plugins/opencode/src/adapters/v1.ts`, `plugins/opencode/src/adapters/v2.ts` | The v1 adapter uses the v1 hook keys and the v2 adapter the v2 domains; both drive the same core intent, and neither translates the other's shape |
| `P3.C5` | The session lifecycle bridge: `session.created` → `send_heartbeat`, `session.idle` → `update_my_status`, `session.error` → blocked, `session.deleted` → offline | `plugins/opencode/src/core/lifecycle.ts` (new) | The server lists the agent as live; the agent goes offline within lease plus reaper after the plugin stops |
| `P3.C6` | The heartbeat schedule, taken from the server's `lease_ttl_seconds` rather than a guess | `plugins/opencode/src/core/lifecycle.ts` | The plugin never beats the lease it was given, and a shortened lease takes effect on the next beat |
| `P3.C7` | The compaction bridge: author the summary on v2, influence it on v1, and write the result back with `add_context` | `plugins/opencode/src/core/compaction.ts` (new) | After a compaction, a context row holds the summary the model actually saw, and the coordination identifiers survive in `result.metadata` |
| `P3.C8` | The context re-injection: registered on `context`, `compaction` and `generate` together, cached per turn, bounded by a character cap | `plugins/opencode/src/core/context.ts` (new) | One fetch per prompt, not one per model call; the injected payload never exceeds the cap |
| `P3.C9` | The policy gate: the tool hook and the shell hook, with `fail_closed` explicit per project | `plugins/opencode/src/core/policy.ts` (new) | The gate throws for an unclaimed agent and passes for a claimed one; a server timeout blocks only when `fail_closed` is set |
| `P3.C10` | The evidence capture: `tool.execute.after` → `add_context` | `plugins/opencode/src/core/evidence.ts` (new) | A failed test run is attached to the task with its `event.error.message` |
| `P3.C11` | The TUI surface: a toast and one status panel | `plugins/opencode/src/adapters/v2-tui.ts` (new) | A toast appears on a blocked tool call; the panel shows the current task and the lease countdown |
| `P3.C12` | The tests: unit for the core, and a contract suite against a live server in Docker | `plugins/opencode/tests/` (new) | `make check-all` runs both, and no LLM is involved in either suite |

### 6.4 Track D — Server support for the plugin

| ID | Task | Files | Acceptance |
|---|---|---|---|
| `P3.D1` | Extract the tool registry out of `handler.go` — the list at `:615` and the dispatch at `:1397-1481` are two hand-maintained copies of the same thing | `internal/mcp/tools.go` (new), `internal/mcp/handler.go` | A test enumerates both and fails if they drift; `handler.go` shrinks by the size of the list |
| `P3.D2` | The `announce_capability` tool: host name, host version, capabilities, negotiated MCP revision | `internal/mcp/tools.go`, `internal/mcp/handler.go` | An announcing plugin is visible through `get_agent`; the revision it reports is the one the server actually negotiated |
| `P3.D3` | Migration `021`: `agents.capabilities JSONB` | `migrations/021_agent_capabilities.up.sql` | Applies to a virgin database and to one that already holds data |
| `P3.D4` | The OpenCode config shape: `mcp.servers`, the `plugins` array with `options`, `protocol: "auto"`, with the v1 keys kept alongside | `internal/configgen/configgen.go` | The generated file parses as both a v1 and a v2 configuration, and the protocol value never asks for a revision the server refuses |
| `P3.D5` | The config generator's own tests | `internal/configgen/configgen_test.go` | The generated OpenCode file is parsed, not string-compared |

### 6.5 Track E — Distribution and surfaces

| ID | Task | Files | Acceptance |
|---|---|---|---|
| `P3.E1` | The hooks UI: a page per project, the run log and the test button | `web/app/pages/hooks.vue` (new), `web/app/composables/useHooks.ts` (new) | An operator creates, tests, enables, disables and inspects a hook without touching the API |
| `P3.E2` | The journal read API | `internal/handlers/activity.go` (new), `cmd/server/routes.go` | `GET /api/activity` filters by project, entity and time |
| `P3.E3` | The activity UI: the journal reader | `web/app/pages/activity.vue` (new) | Filter by project and entity; `make web-typecheck` and `make web-test` pass |
| `P3.E4` | The npm publish workflow | `.github/workflows/publish-npm.yml` (new) | A tag publishes the package with OIDC provenance; a dry run publishes nothing |
| `P3.E5` | The organisation manifest at `/.well-known/opencode` | `internal/handlers/opencode.go` (new), `cmd/server/routes.go` | The manifest lists the MCP server and the plugin, and is served without authentication |
| `P3.E6` | The build gate: `plugin-typecheck` and `plugin-test`, wired into `check-all` | `Makefile` | `make check-all` fails when the plugin does not typecheck or its tests fail |
| `P3.E7` | The plugin README and the setup guide | `plugins/opencode/README.md` (new), `docs/MCP_SETUP.md` | The guide shows the generated config, the install command, and what the plugin does and does not do |

### 6.6 Tracked separately, not in this phase

- **The Skills-over-MCP channel.** The MCP Skills extension (SEP-2640) is the
  host-neutral way to ship a coordination playbook — `SKILL.md` plus
  supporting files, a manifest with a URI, a SHA-256 digest and a byte size,
  validated against the extension's rules (digests verified, at most 512 files
  and 16 MiB, approval bound to the file set). It needs the server to expose
  `skills/list` and `skills/get`. Worth doing, but it is a distribution
  sibling rather than part of the phase: `P3.E5` covers the host-specific
  channel, and this one covers the protocol-neutral one.
- **The signed journal** (`P3.A5`) and **agent identity** (§9 D10) depend on
  decisions nobody has taken yet.

### 6.7 Sequence

Three waves, and the two products do not wait for each other.

- **Wave 1 — foundation.** `P3.A1` → `P3.A2`/`P3.A3` → `P3.A4`/`P3.A5`/`P3.A6`,
  in parallel with `P3.C1` → `P3.C2` → `P3.C3` → `P3.C4`. The plugin skeleton
  touches no server code, so it moves while the journal lands.
- **Wave 2 — the two products.** `P3.B1` → `P3.B2` → `P3.B3` → `P3.B4` →
  `P3.B5` → `P3.B6` → `P3.B7` → `P3.B8` → `P3.B9` → `P3.B10`, in parallel with
  `P3.C5` → `P3.C6` → `P3.C7`/`P3.C8`/`P3.C9`/`P3.C10` → `P3.C11`.
- **Wave 3 — surfaces and distribution.** `P3.B11`/`P3.B12` → `P3.E1`;
  `P3.D1` → `P3.D2`/`P3.D3` → `P3.D4`/`P3.D5` → `P3.E5`; `P3.C12` → `P3.E4` →
  `P3.E6` → `P3.E7`; `P3.E2`/`P3.E3` land with the journal.

**Critical path to a shipped hook:** `P3.A1` → `P3.A3` → `P3.B1` → `P3.B9` →
`P3.B11` → `P3.E1`. Nine tasks.

**Critical path to a shipped plugin:** `P3.C1` → `P3.C4` → `P3.C5` →
`P3.C7`.

### 6.8 Decisions that block tasks

| Decision | Blocks |
|---|---|
| D1 — package name and scope | `P3.C1`, `P3.E4` |
| D4 — hook default and authoring authority | `P3.B7`, `P3.B8` |
| D5 — the OpenCode config shape | `P3.D4` |
| D6 — the journal's anchor | `P3.A5` |
| D9 — the lease against the host's warming cadence | `P3.C6` |

### 6.9 Definition of done

Every task in this phase is closed only when all of these hold:

- The acceptance criterion is met, and the check is a command a reviewer can
  run. Nothing closes on "it works on my machine".
- `make check-all` passes.
- No secret appears in a source file, a log line, a captured hook output or a
  test fixture.
- Every new table has a migration that applies both to a virgin database and
  to one that already holds data.
- Every new mutating path has a test that asserts the observable outcome, not
  the literal response.
- The task owns the files it names, and touches nothing else. Unrelated
  refactors are separate tasks.

---

## 7. Dependencies on the rest of the roadmap

| Dependency | Blocking? | Reasoning |
|---|---|---|
| **M5 — stateless MCP (2026-07-28)** | **No** | The server negotiates `2024-11-05` and the OpenCode host speaks it. The plugin is built against today's revision and reports the negotiated revision through `announce_capability`, so it adapts when M5 lands. Waiting for M5 would delay Phase 3 by weeks to buy nothing the plugin needs. |
| **M3 — audit journal** | **Yes, and Phase 3 delivers it** | The journal was reserved for M3 as migration `011` and never written; Phase 3 writes it as `022`, because the hook layer and the plugin both need somewhere to put their evidence. |
| **M4 — approvals** | **No** | Approvals are needed for the `permission.asked` gate (§4.4) and for a *blocking* hook mode. Both are deferred. Phase 3 ships asynchronous hooks only. |
| **Idempotency keys on mutating tools** | **Yes, and Phase 3 delivers it** | The 2026 protocol makes retries normal traffic: a broken response stream loses the in-flight request and the client MUST re-issue it with a new request id, and an MRTR retry re-sends the *original* request. A coordination plane whose tools mutate shared state must be idempotent by construction. `claim_task` and `complete_task` should accept a client-supplied idempotency key **now**, not after M5 — the cost is one column and one unique index, and the alternative is a duplicated claim or a double-completed task. |
| **M7 — context retrieval** | **No** | The compaction bridge uses `search_contexts` and `list_contexts`, which exist and work. M7 improves retrieval; Phase 3 does not wait for it. |
| **M9 — distribution** | **No** | MCP registry publication is M9. The npm package and the `.well-known/opencode` channel are Phase 3's own distribution story. |
| **Phase 4 — Copilot bridge** | **No** | The hook layer is host-neutral by design. |
| **A2A delegation** | **No — and do not advertise it** | Outbound A2A delegation does not execute today: `cmd/server/main.go` builds the task manager with a nil executor, so a delegating peer receives a success signal for work that was never performed (`docs/ROADMAP.md:22-25`). The plugin must not present delegation as working. |

---

## 8. Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Hooks are arbitrary command execution | **High** | Disabled by default; per-project opt-in; executable allowlist; no shell; cwd constrained; secrets by reference; every run audited. |
| OpenCode Console turns the integration target into the competitor | **High — strategic** | Build the vendor-neutral layer, not a console (§3.5). The plugin is the multi-agent truth that Console's single-workspace model lacks; the hook layer is the cross-vendor paper trail its per-workspace policies cannot be. |
| The generated config is the v1 shape, so a v2 host finds no server | **High** | Fixed in P3-11: emit `mcp.servers`, the `plugins` key, and `protocol: "auto"`. Covered by a test that parses the generated file. |
| The uncommitted working tree | **Resolved** | The VS Code and Copilot layouts landed in `b080689`. What remains in the tree is this document, the market scan, a `.gitignore` rule for the scratch directory, and the removal of the vendor documentation snapshots that a runtime auto-commit had swept in. |
| OpenCode's plugin API is moving | Medium | Dual export; exact version pins; capability probe; a fail-loud rule for the session lifecycle only. |
| The v2 compaction hook's invocation is unverified in the wild | Medium | Implement both the v1 and v2 paths; treat the `session.compaction.*` events as the fallback signal; report the active mode through `announce_capability`. |
| The plugin makes the host slower | Medium | Bounded context injection; a per-turn cache; a hard character cap; register on `context`, `compaction` and `generate` together. |
| The host's idle cadence fights the lease | Medium | OpenCode warms a session roughly every 4 minutes across a 30-minute window. The server's 60-second lease reaped every 30 seconds (`internal/coordination/reaper.go:36-38`) is fine, but the plugin's heartbeat must survive a warmed-but-idle session — otherwise a healthy agent is declared stale. See §9 D9. |
| `handler.go` is a 3,285-line monolith with two hand-maintained tool lists | Medium | P3-10 extracts the registry before adding a tool. |
| Hook stdout can leak secrets | Medium | Redaction against the secret set before capture and before `post`. |
| A plaintext credential sits in `.opencode/opencode.jsonc` in a working tree | Medium | Rotate that key and move it to `{env:AGENT_SHAKER_API_KEY}`. It is git-ignored, so it was not committed — but it is a live-format key on disk. |
| Event-name duplication (`task_update` / `task_updated`) | Low | Normalised in the broker (P3-2), with aliases for the WebSocket. |
| The server advertises MCP `2024-11-05` on its A2A agent card (`internal/a2a/server/agent_card.go:100`) | Low | Correct as-is — a card has no negotiated session. Stated so nobody "fixes" it. |

---

## 9. Open decisions

These need the lead's answer. The recommendation is the first option in each.

**D1 — npm package name and scope.** `@techbuzzz/opencode-agent-shaker`
(recommended) — scoped, so the name cannot be squatted, and it matches the
repository's GitHub org. Alternative: `opencode-agent-shaker` unscoped, which
carries squatting risk on a name others will also want.

**D2 — where the package lives.** `plugins/opencode/` as a standalone package
(recommended) — published independently, no effect on the Go build or the web
app's dependency tree. Alternative: a second entry point in the root
`agent-shaker-mcp-bridge` package, which couples two unrelated consumers.

**D3 — does Phase 3 wait for M5?** No (recommended). Build against
`2024-11-05`, report the negotiated revision, adapt later. Alternative: hold
the plugin until the stateless transport lands, which delays the whole phase
behind a transport rewrite.

**D4 — hook authoring authority and the default.** Disabled globally by
default, enabled per project, authorable only by a human principal with the
hook scope (recommended). The alternative — enabled by default — turns every
deployment into a remote-code-execution surface the moment the feature ships.

**D5 — the `opencode.json` shape.** Emit the **v2** shape — `mcp.servers`, the
`plugins` array with `options`, and `protocol: "auto"` — and keep the v1
`plugin` key and the v1 `mcp` block alongside it for hosts that still read
them (recommended), with a test that parses the generated file for both. The
alternative — emit one shape and instruct users to edit the file — makes the
generated config wrong for part of the installed base, and a v2 host reading a
v1 `mcp` block finds no server at all.

**D6 — the audit chain's anchor.** Publish the chain's root hash to a
location the operator controls (recommended — the project's own repository, or
a signed release attestation), so the journal is tamper-evident against
someone with database write access. Alternative: store the anchor in the
database, which proves nothing against a database-level attacker.

**D7 — the uncommitted VS Code / Copilot work.** Fold it into Phase 3 as a
first commit (recommended), so the formats the server advertises and the ones
the UI offers are consistent from the start. Alternative: land it separately
before Phase 3 begins.

**D8 — how far the TUI goes.** A toast and one status panel (recommended).
Alternative: a full task board inside the TUI, which is a product in its own
right and belongs to Phase 5.

**D9 — the lease against the host's idle cadence.** Keep the server's
60-second lease and let the plugin heartbeat on the server's schedule
(recommended). OpenCode warms a session roughly every 4 minutes across a
30-minute window, so a longer lease would not be wrong on paper — but the
reaper's job is to notice an agent that has actually stopped, and a lease
longer than the host's own keep-alive only delays that notice. The
alternative — a lease matched to the warming cadence — makes a stopped agent
look alive for half an hour.

**D10 — how far to go on agent identity.** Ship the audit journal unsigned
(recommended for Phase 3), and treat signed action receipts as a Phase 5
project with its own decision about where the signing key lives. The relevant
IETF work — the Agent Passport System and the `aiagent-auth` draft — is not
stable yet, and building against a moving draft is how a protocol becomes a
liability.

---

## 10. Sources

Protocol and platform, checked 2026-10-24:

- MCP specification, revision 2026-07-28 — stateless core, `server/discover`,
  `Mcp-Method`/`Mcp-Name`, `ttlMs`/`cacheScope`, `_meta` client identity.
- MCP SEP-1865 / `modelcontextprotocol/ext-apps`,
  `specification/2026-01-26/apps.mdx` — MCP Apps, stable 2026-01-26.
- MCP Apps overview (mcp-framework.com) — host support list.
- A2A v1.0 / v1.0.1 specification (a2a-protocol.org, `a2aproject/A2A`) —
  agent card, `A2A-Version`, signed cards, `supportedInterfaces[]`.
- `opencode.ai/v2/docs/build/plugins` and `.../migrate-v1/` — the v2 plugin
  API, the dual export, the v1↔v2 API mapping.
- `opencode.ai/docs/config/` (updated 2026-10-23) — `plugin` vs `plugins`,
  `{env:VAR}` substitution, plugin directories.
- `opencode.ai/docs/cli/` — `opencode run`, `--format json`, `opencode serve`.
- `agent.minimax.io/docs/cli/automation.md` — `mcode exec`, stdin/stdout
  contracts, exit codes, `--prompt-mode`, `--output-schema`.
- `agent.minimax.io/docs/llms.txt` — the regional document index used for
  MiniMax Code product facts.
- `.mavis-scratch/scan/cc-hooks.md` — the in-repo Claude Code hooks
  reference used as the hook-contract model.

Repository, as of `feat/coordinate-plan`:

- `internal/mcp/handler.go` — 35-tool list (`:615`) and dispatch (`:1397`);
  `server/discover` (`:509`, `:541`).
- `internal/mcp/protocol.go` — revision negotiation (`:25`, `:30-35`), error
  codes (`:48`, `:59`).
- `internal/mcp/coordination_tools.go` — heartbeat (`:113`), handoff (`:231`),
  blocker (`:546`).
- `internal/coordination/reaper.go` — 60 s lease, 30 s reaper (`:36-38`).
- `internal/configgen/configgen.go` — the OpenCode layout (`:181-192`), the
  credential spellings (`:47`, `:55`), the three-file bundle.
- `internal/models/tooltype.go` — the tool-type vocabulary, the formats, and
  the `announce_capability` note (`:18-19`).
- `migrations/` — 001–022, with 011 and 012 unallocated.
- `docs/ROADMAP.md` — the M3/M4/M5 milestones, task T3.1, and the A2A
  execution defect (`:22-25`).
- `docs/research/agent-shaker-community-ideas-2026-09.md` — the in-repo
  competitive scan.

Market and competitive, in [MARKET_SCAN_2026-10.md](./MARKET_SCAN_2026-10.md)
with per-claim URLs:

- `anomalyco/opencode` — the canonical repository, MIT, v1.19.4 (2026-10-22).
  The `sst/opencode` path redirects here.
- `@opencode/plugin` 2.1.4 (2026-10-23, created 2026-09-02) and
  `@opencode-ai/plugin` 1.19.4 — the two npm scopes, both MIT.
- `opencode.ai/v2/docs/compaction` — the checkpoint model: `auto`, `keep.tokens`
  (15,000), `buffer` (15 %, min 16k), and the `SessionCompactionResult`
  contract with `summary`, `native`, `metadata` and `tokens`.
- `opencode.ai/v2/docs/policies`, `.../console/budgets/`, `.../references/`,
  `.../warming/` — the Console's policy authority, budget enforcement,
  the References primitive, and the 4-minute/30-minute warming cadence.
- `modelcontextprotocol.io/specification/2026-07-28/changelog.md` — SEP-2567
  (no session id), SEP-2575 (stateless core, `server/discover`), SEP-2549
  (`ttlMs`/`cacheScope`, deterministic ordering), SEP-2322 (MRTR
  `input_required`), and the idempotency requirement for re-issued requests.
- `modelcontextprotocol.io/extensions/skills/overview.md` — SEP-2640, the
  Skills extension: digests, 512 files, 16 MiB.
- `modelcontextprotocol.io/extensions/apps/overview.md` — SEP-1865, MCP Apps.
- `codex-docs.com/en/docs/hooks` — Codex CLI's `PreToolUse` interception and
  its `mcp_tool` handler kind.
- `a2a-protocol.org` v1.0 / v1.0.1 — agent card, `A2A-Version`, signed cards.
- `datatracker.ietf.org/doc/html/draft-pidlisnyi-aps-03` and
  `draft-klrc-aiagent-auth-01` — the Agent Passport System and `aiagent-auth`.
- `open-telemetry/semantic-conventions-genai` — the GenAI conventions after
  they moved out of core, all still at Development status.
- Paperclip (MIT, 106,790 stars) — the execution-side competitor.
- Claude Managed Agents, the OpenAI Agents API, Microsoft Entra Agent ID —
  the server-side governance and identity direction.

## 11. Implementation notes

What the build changed against the plan above. These are the places where the
code said something different from the design, and the design was wrong.

### 11.1 The domain event stream is not the WebSocket stream

The plan said the broker replaces the direct hub broadcasts. It does not, and
it should not: the broadcasts are the browser's notification and carry
whatever shape the UI wants — `task_update` and `task_updated` were two names
for one change, and the payloads are flat maps that predate the coordination
plane.

So there are two views of a change, published at the same place:

- the **broadcast** goes to the WebSocket, unchanged, under its existing name;
- the **domain event** goes to the broker, with one canonical name per change.

The hook layer consumes the domain event. A hook that had to parse a UI
payload to work out which task completed would break whenever the frontend
moved a field, and would silently stop firing on the changes nobody
remembered to update it for.

The publishers are the mutation sites themselves: claim, complete, reassign,
handoff, blocker raised, task blocked, task ready, context published, agent
gone stale. `internal/events` holds the vocabulary; `internal/events/broker.go`
holds the fan-out.

### 11.2 The hook's working directory is configured, not derived

The plan said the working directory is constrained to a registered
`project_repos` path. `project_repos` records a repository **URL**, not a path
on disk — the server has no local checkout to resolve, and guessing one would
put a command in a directory nobody chose.

`AGENT_SHAKER_HOOK_WORKDIR` names it instead. The `{{project.repo_path}}`
placeholder therefore means "the directory the hook runs in", and it resolves
to that value.

### 11.3 The environment contract

| Variable | Default | Meaning |
|---|---|---|
| `AGENT_SHAKER_HOOKS_ENABLED` | `false` | The global switch. Off means no hook runs, whatever is configured. |
| `AGENT_SHAKER_HOOK_EXECUTABLES` | empty | The allow-list, comma separated. An empty list allows nothing. |
| `AGENT_SHAKER_HOOK_WORKDIR` | the server's own directory | Where hook commands run. |
| `AGENT_SHAKER_HOOK_MAX_CONCURRENCY` | 4 | How many hooks run at once across the process. |
| `AGENT_SHAKER_HOOK_QUEUE_SIZE` | 64 | How many runs may be waiting. |

The per-project switch is `projects.hooks_enabled` (migration 019). Creating a
hook turns it on for that project, because creating a hook *is* the decision
to run commands for it — and a separate switch nobody knew about is a hook
that silently never fires. `PUT /api/projects/{id}/hook-settings` turns it
back off without deleting anything.

### 11.4 The MiniMax Code bridge needs no plugin

MiniMax Code has hooks of its own, so it does not need the OpenCode plugin
path at all. The bridge is a project hook:

- `.minimax/hooks.json` — `SessionStart`, `SessionEnd` and `PostCompact`.
- `.minimax/agent-shaker-hook.mjs` — the script they call.

`SessionStart` renews the agent's lease, `SessionEnd` releases it, and
`PostCompact` reads the transcript and pushes it into the shared contexts. The
last one is the same compaction bridge the OpenCode plugin implements, and it
is the reason the shared store exists: the session's context survives the
session.

The script is fail-open and side-effect only. It never blocks a turn, never
writes to stdout — an empty stdout is what tells the runtime "nothing to add"
— and always exits 0. A bridge that could fail a turn would be turned off.

The hook file is inert until the operator enables it in Settings → Hooks →
Project. The environment it reads:

```
AGENT_SHAKER_URL         the API base URL
AGENT_SHAKER_API_KEY     the agent's credential
AGENT_SHAKER_PROJECT_ID  the project the session belongs to
AGENT_SHAKER_AGENT_ID    the agent this session is
```

### 11.5 The schema

| Migration | What |
|---|---|
| `022_audit_log` | `audit_events`: the append-only journal, with the `prev_hash`/`hash` chain and the anchor. |
| `019_hooks` | `hooks` and `hook_runs`, plus `projects.hooks_enabled`. |
| `020_idempotency` | `idempotency_keys`: `(tool, agent_id, key)` → the first response. |

The idempotency keys are on `claim_task` and `complete_task`, and they are
optional. A client that sends one gets a repeat of the same request replayed
rather than a second claim or a second completion; a client that sends none
gets the old behaviour, because requiring a key would break every existing
agent on the day the revision landed.

### 11.6 The checks run in Docker

`scripts/verify-docker.sh` and `scripts/verify-docker.ps1` run every check in
a container: gofmt, `go vet`, the build, `golangci-lint`, the tests, the race
detector, the frontend typecheck and the frontend tests.
`docker-compose.test.yml` brings up a throwaway Postgres on a tmpfs under its
own compose project, so the integration stack cannot reach a developer's
running database.

The rule the scripts exist to enforce: nothing is installed on the host and
nothing is run on the host. A checkout is not a toolchain, and a check that
depends on what the developer happens to have installed is a check that passes
on one machine and fails on another.

`scripts/hook-e2e.mjs` is the end-to-end proof: it creates a project, an
agent and a task, binds a hook to `task.completed`, completes the task, and
waits for the run log to show the command's own captured output.

#### What was verified, and where

Every check ran in a container. Nothing was installed on the host and nothing
was run on the host.

| Check | Result |
|---|---|
| `gofmt -l cmd internal tests` | clean |
| `go vet ./cmd/... ./internal/... ./tests/...` | clean |
| `go build ./...` | clean |
| `go test -count=1` | 20 packages, all pass |
| `go test -race -count=1` | 20 packages, all pass — after the race in the runner was fixed (§11.7) |
| `npm run typecheck` (web) | clean |
| `npm test` (web) | 6 files, **83/83** pass, of which 29 are the hook composable's |
| `npm run typecheck` (plugin) | clean |
| `npm test` (plugin) | **54/54** pass |
| `scripts/hook-e2e.sh` | a real command ran against a real event; captured stdout `hook ran for: prove the hook layer runs a command` |
| `scripts/minimax-hook-e2e.sh` | the MiniMax Code bridge renewed a lease and pushed a compacted transcript (§11.14) |
| `scripts/migration-e2e.sh` | the migrations applied over a database that already held rows, and the rows survived (§11.17) |

Two operational notes the verification produced:

- **The Go package list is explicit, not `./...`.** `./...` walks
  `web/node_modules` when it exists, which is slow and races any npm run in
  the same tree — a race that actually broke a verification run here.
- **The frontend is copied into its container, not bind-mounted.** `npm ci`
  deletes and recreates `node_modules`, so a bind mount writes that tree into
  the checkout. The copy keeps the host tree untouched.

### 11.7 What the verification caught

The build was verified in Docker, and the verification found three defects
that reading the code would not have:

- **A non-zero exit was reported as a signal.** The runner set `Signaled` on
  every `exec.ExitError`, so a command that ran and exited 3 was recorded as
  exit 128. The distinction is the one thing a hook author has to work with:
  128 means "killed", 3 means "the command said no". The runner now asks the
  process state whether the process actually exited.
- **The runner raced with itself.** `cmd.Run` publishes `cmd.Process` from
  inside the goroutine it runs in, and the timeout path reads that same field
  to kill the process group — a data race on precisely the field the timeout
  depends on. The race detector finds it in the timeout test every time. The
  process is now started in the calling goroutine, so the field is published
  before anything reads it.
- **`project_repos` records a URL, not a path.** The plan said the hook's
  working directory would be constrained to a registered repository path. It
  cannot be, because there is no local checkout in the schema — see §11.2.
- **The per-project opt-in had no API.** The column existed and nothing could
  set it, so every hook would have sat inert. Creating a hook now turns the
  project on, and `PUT /api/projects/{id}/hook-settings` turns it back off.

Two more came out of the delegation, and both are worth keeping: the v1
plugin API's `server` export is a **function**, not an object, and the v1 tool
hooks carry `{tool, sessionID, callID}` rather than the `{tool, input}` shape
the plan assumed.

### 11.8 The tool registry

`P3.D1` moved the advertised schemas and the dispatch that serves them out of
`internal/mcp/handler.go` and into `internal/mcp/tools.go`. The file shrank
from 3,285 lines to 2,827, and the two halves of the registry now sit in one
place instead of two ends of a very long file.

The dispatch became a **map** rather than a switch:

```go
var toolExecutors = map[string]toolExecutor{...}
```

A switch's case labels are not enumerable at runtime, so the test that holds
the advertised tools and the callable tools in step would have had to parse Go
source to do it. With a map it is a map lookup. The test is symmetric — an
executor without a schema is a tool nobody can find, and a schema without an
executor is a tool that fails when it is called.

The move itself was scripted (`scripts/extract-tool-registry.mjs`) rather than
retyped, because a 700-line schema block retyped by hand is a 700-line chance
to introduce a difference. The script derives the executors from the switch it
replaces, and it refuses to run twice.

**The script found a defect in itself.** Its first pass matched tool names
with `[a-z_]+`, which silently skipped the three A2A tools — their names
contain a digit. The registry's own test named them, which is the argument for
having the test.

### 11.9 The OpenCode 2 document

`P3.D4` and `P3.D5` added the OpenCode 2 layout. It is a **separate format**,
not a variation of the OpenCode one, and the reason is structural: `mcp` is one
key with two shapes, so one file cannot serve both majors. OpenCode 1 reads
the servers directly under `mcp`; OpenCode 2 reads them under `mcp.servers` and
says so in its own words.

Four things differ, and all four fail silently when wrong:

| | OpenCode 1 | OpenCode 2 |
|---|---|---|
| Servers | `mcp.agent-shaker` | `mcp.servers.agent-shaker` |
| Disabled | `enabled` | `disabled` |
| OAuth | not mentioned | **on by default for remote servers** |
| Plugins | `plugin: ["name"]` | `plugins: [{ package, options }]` |

The OAuth one is the sharpest edge: OpenCode 2 turns OAuth on for every remote
server unless the file says `oauth: false`, and this server authenticates with
a header. Without the field the client tries to authenticate against a server
that does not speak OAuth, and the agent connects to nothing — with no error to
read. The generated file says `"oauth": false`.

The protocol revision is pinned to `legacy`, because that is what the server
speaks today. `auto` would spend a probe on every connection to discover what
`legacy` already says, and `2026-07-28` would fail against a server that has
not made the move. Both become correct when it does.

The format is chosen by the caller — `GET /api/agents/{id}/config?format=opencode-v2`
— and the default for `tool_type: opencode` stays the v1 layout, because that is
the one that is known to work for the installations that exist.

The tests parse the rendered document rather than matching its text. A document
can be valid JSON, carry every key anyone expects, and still be a document the
host ignores.

### 11.10 What a client says about itself

`P3.D2` and `P3.D3` added `announce_capability` and migration `021`.

The server has to know what a client can do for it, and it cannot guess: an
agent whose client cannot accept a pushed context, or author a compaction
summary, is an agent the server must not ask to. The tool is called once, at
setup, and the answer lands in `agents.capabilities`.

The vocabulary is closed (`compaction`, `context_injection`, `policy_gate`,
`evidence_capture`), so a typo is refused rather than stored: a capability
nobody can act on is worse than an announcement that failed.

The plugin announces only what it can honour — the intersection of the host
surface it probed and the features it is configured for. Announcing a
capability the client cannot deliver would send the server looking for a
feature that is not there.

### 11.11 The organisation manifest

`P3.E5` publishes `/.well-known/opencode`: the one place an organisation can
hand a whole fleet of agents the same coordination server and the same plugin
without touching a machine. OpenCode fetches it itself, before any credential
exists, and merges what it finds underneath every other configuration source.

It is rendered by the same generator that renders a per-agent bundle, so there
is one implementation of the OpenCode document rather than two that drift. It
carries no secret — every credential in it is the `{env:AGENT_SHAKER_API_KEY}`
placeholder — which is what makes it safe to serve from an unauthenticated
path.

### 11.12 The publish path

`P3.E4` adds `.github/workflows/publish-npm.yml`: a tag, a check that the tag
agrees with the package's own version, the checks run again at publish time,
and npm trusted publishing with OIDC provenance — no long-lived token in the
secrets and nothing to rotate.

The tag prefix is the package's own name rather than a bare `v*`, because the
Docker images and the npm package are released independently and must not move
together.

### 11.13 The page, split

The hook page was the largest route view in the app — 34,870 bytes, with the
form, the list, the run log and three dialogs in one file. It is now 13,673
bytes, below `projects/[id].vue`, and the pieces live where the rest of the
app keeps them:

| File | Responsibility |
|---|---|
| `components/domain/HookCard.vue` | one configured hook: what it runs, what it matches, and the four actions |
| `components/domain/HookFormModal.vue` | the form, its validation, and the conversion to the request body |
| `components/domain/HookRunRow.vue` | one run, with its captured output collapsed |
| `pages/hooks.vue` | the composition surface: the project picker, the switch, the list, the log, the dialogs |

The rule is the repo's own convention — every other feature has a `*Card.vue`
and a `*FormModal.vue` under `components/domain` — and it is also the Vue
composition rule: a route view that owns orchestration *and* the markup for
three sections has two responsibilities.

The form moved with its state, including the guard against a double submit:
`FormModal` emits `submit` twice for one click, so the in-flight request is
latched inside the modal rather than in the page that happens to use it today.

### 11.14 The MiniMax bridge, proven

The bridge was written and left unproven, which is the state in which a hook
looks like a feature and behaves like a guess. `scripts/minimax-hook-e2e.mjs`
drives the hook script the way MiniMax Code would — the payload on stdin, the
environment in place — against the throwaway stack:

```
✓ an agent-bound credential was minted
✓ SessionStart renewed the lease (last_seen …895782Z → …007331Z)
✓ PostCompact pushed the transcript as context …
```

What it proves is the part that is ours: the script reads the payload,
resolves the agent, renews the lease and pushes the compacted transcript. What
it does **not** prove is that MiniMax Code invokes the hook — that needs the
host, and the hook stays inert until an operator enables it in
Settings → Hooks → Project.

The credential matters: the hook runs with a key bound to one agent, so it can
renew its own lease and write its own contexts, and nothing else. A bridge
that ran with a flat key would be a bridge with no owner.

### 11.15 The journal

`internal/audit` is the evidence layer. The broker publishes a domain event;
the journal subscriber writes it; the chain makes the record checkable.

**The chain is per project, and it is written under a lock.** Two writers
reading the same head produce two rows claiming the same predecessor — a fork,
which is exactly what the chain exists to prevent. The writer takes a
Postgres advisory transaction lock on the project before it reads the head, so
there is one writer per chain at a time.

**The anchor is the point.** A hash chain is only as strong as its anchor: a
database-level attacker who rewrites the trail can recompute every hash in it.
So the head is published *outside* the database — a file the operator names
(`AGENT_SHAKER_AUDIT_ANCHOR_DIR`), and the API that reads it back
(`GET /api/activity/anchor`). An operator who pins the head has a value that
a rewrite of the trail cannot match.

**Nothing rewrites the journal.** There is no update and no delete against
`audit_events` anywhere in the codebase, and a test pins that by reading the
package's own source — there is no behavioural way to prove the absence of a
statement, so the check is a source scan.

**A journal write never fails the operation that caused it.** The subscriber
swallows the error; the recorder returns it. The decision to ignore it belongs
to the caller, and the caller knows that a trail is worth less than the work.

**What it records is the domain vocabulary**, not every column of every row.
A task claimed, a context published, an agent gone stale. A row-level change
log for the whole schema is a different feature and is not what a coordination
trail is for; the distinction is stated rather than blurred.

`web/app/pages/activity.vue` reads it: filter by project and entity, and the
chain head beside the list.

### 11.16 A PowerShell trap worth writing down

The verification scripts are `.ps1`, and Windows PowerShell 5.1 reads a
BOM-less file as **ANSI**, not UTF-8. A `✓` written in UTF-8 is three bytes,
and read as CP1252 they are `â`, `œ` and a **right double quotation mark** —
which PowerShell treats as a string delimiter. The file parses as an
unterminated string and fails with an error pointing at a line that looks
perfectly fine.

The repository's own scripts get away with it because their non-ASCII
characters are in comments. The rule that falls out: **a non-ASCII character
inside a double-quoted string in a `.ps1` file is a parse error waiting to
happen.** Comments are safe; strings are not.

The same trap applies to the em dash, which is also three bytes ending in a
smart quote.

### 11.17 The migrations over data that already exists

A migration that only ever runs against an empty database proves nothing
about the case it will actually meet. `scripts/migration-e2e.sh` applies every
migration but the journal, writes a project, an agent, a task and a context
through the API, applies the journal migration **over those rows**, and then
checks that the rows are still readable and that the journal now records what
happens.

It found a defect that only exists with rows in the table: the journal's read
path scanned its nullable string columns into plain `string`, and a NULL in
`trace_id` failed the whole read with *"converting NULL to string is
unsupported"*. An empty database never produced a NULL there, so no unit test
would have found it.

**The migration number was wrong, and the proof caught it.** The journal was
planned as `011`, reserved for M3 since the first plan. `011` is now *below*
the tree's maximum, and `golang-migrate` walks forward only — so a migration
numbered `011` would never have been applied at all. Every deployment would
have reported itself as migrated, and the table would not have existed. The
journal is `022`.
