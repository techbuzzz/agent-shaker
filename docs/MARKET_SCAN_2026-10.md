# Agent Shaker — Market & Technology Trend Scan (late October 2026)

**Date of scan:** 2026-10-24
**Scope:** Phase 3 design input (OpenCode deep integration: npm package, compaction hook, heartbeat on session lifecycle).
**Purpose:** find *trends* among new similar applications and the *newest* approaches in the agent tooling ecosystem, so Phase 3 is not built on a stale plan.
**Method:** primary sources preferred — official docs, spec revisions, SEP documents, release notes, GitHub API, npm registry API. Secondary/aggregator sources are labelled as such.

> **This is evidence, not a plan.** Nothing here prescribes an implementation. Section 3 ranks techniques; Section 4 lists category table stakes; Section 5 lists what could not be verified and must be checked by hand.

### Reading the confidence column

| Confidence | Meaning |
| --- | --- |
| **High** | Verified directly against a primary source (official doc, spec, release API, registry API) on 2026-10-24. |
| **Medium** | Primary source exists but is partial, or the fact is corroborated by multiple secondary sources only. |
| **Low** | Single secondary source; treat as a lead, not a fact. |

### Notable corrections to assumptions in the brief

1. **MCP revision `2026-07-28` exists and is the current revision.** Confirmed on the canonical spec URL and via the changelog page.
2. **OpenCode is no longer `sst/opencode`.** The canonical repository is now `anomalyco/opencode` (the old path redirects). It is MIT, 220,944 stars.
3. **OpenCode has a second-generation plugin API.** V2 ships a *different* npm package (`@opencode/plugin`, first published 2026-09-02) from V1 (`@opencode-ai/plugin`), and V2 replaces the V1 `tool.execute.*` / `event` hook model with a named hook-domain API. A Phase 3 design written against the V1 plugin API would be stale on arrival.
4. **A2A is at spec v1.0.0** (repo release v1.0.1). The claim "A2A v1.2" appearing in aggregator posts is not supported by the official docs and is listed in Section 5.
5. **OpenCode V2 is not just a host — it is a coordination and governance plane.** The V2 docs include a hosted **Console** with workspaces, members, SSO, SCIM, **budgets as policy rules (HTTP 429 on breach)**, **team-wide policies that have "final authority over the member's own configuration"**, **Harnesses that run other agents' runtimes (Claude Code, Codex, Hermes, Pi, Raycast)**, and a **References** primitive for named external context — plus a built-in **warming** cadence (4-minute idle interval, 30-minute window). This is the strongest competitive signal in the scan and it changes how Phase 3 should be positioned. See §A1.7.

---

## Section 1 — Findings table

The load-bearing claims only. Every row carries a real URL and the date the evidence was observed or published.

| # | Claim | Evidence URL | Date | Confidence |
| --- | --- | --- | --- | --- |
| 1 | MCP's current protocol revision is `2026-07-28`; it is the canonical "latest" revision on the spec site. | https://modelcontextprotocol.io/specification/latest | 2026-07-28 (published) | High |
| 2 | MCP `2026-07-28` is **stateless**: the `initialize` / `notifications/initialized` handshake is removed, every request carries protocol version + client capabilities in `_meta`, and `Mcp-Session-Id` is gone from Streamable HTTP. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | 2026-07-28 | High |
| 3 | MCP `2026-07-28` adds a mandatory `server/discover` RPC; servers MUST implement it to advertise supported protocol versions, capabilities and identity. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | 2026-07-28 | High |
| 4 | MCP `2026-07-28` introduces **Multi Round-Trip Requests (MRTR)**: servers return `InputRequiredResult` (`resultType: "input_required"`) instead of server-initiated requests; `resultType` is now required on all results (`"complete"` / `"input_required"`). | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | 2026-07-28 | High |
| 5 | MCP `2026-07-28` requires standard headers `Mcp-Method` and `Mcp-Name` on Streamable HTTP POST requests, and adds `x-mcp-header` for custom headers from tool parameters. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | 2026-07-28 | High |
| 6 | MCP `2026-07-28` requires `ttlMs` and `cacheScope` on `tools/list`, `prompts/list`, `resources/list`, `resources/read`, `resources/templates/list` results (new `CacheableResult` interface), and requires servers to return tools in a deterministic order for prompt-cache hit rates. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | 2026-07-28 | High |
| 7 | MCP **deprecates Roots, Sampling and Logging**; new implementations should not add them. HTTP+SSE transport is also Deprecated. OAuth 2.0 Dynamic Client Registration is deprecated in favour of Client ID Metadata Documents. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | 2026-07-28 | High |
| 8 | MCP moved server-side **Tasks** out of the core protocol into an official extension `io.modelcontextprotocol/tasks`; `tasks/result` is replaced by polling `tasks/get` plus client-to-server `tasks/update`. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md · https://modelcontextprotocol.io/extensions/tasks/overview.md | 2026-07-28 | High |
| 9 | MCP has an **extensions framework** (SEP-2133, Final); `ClientCapabilities`/`ServerCapabilities` now carry an `extensions` field, and extensions are how optional features ship without touching core. | https://modelcontextprotocol.io/extensions/overview.md · https://modelcontextprotocol.io/seps.md | 2026-04-17 (SEP-2596) / ongoing | High |
| 10 | **MCP Apps** (SEP-1865, Final, Extensions Track) exists: servers can ship interactive UI resources to hosts. | https://modelcontextprotocol.io/seps.md · https://modelcontextprotocol.io/extensions/apps/overview.md | 2025-11-21 | High |
| 11 | **MCP Skills extension** (SEP-2640, Final): servers expose `skills/list` + `skills/get`, files are served through `resources/read`; a skill is a `SKILL.md` + supporting files; hosts MUST verify SHA-256 digest and byte size before use; servers SHOULD NOT exceed 512 files or 16 MiB per skill. | https://modelcontextprotocol.io/extensions/skills/overview.md | 2026-04-23 (SEP created) | High |
| 12 | The **official MCP Registry** exists (in preview) with a `server.json` manifest format; publishing uses `mcp-publisher init` / `publish`; namespace authentication is required (`io.github.<user>/*`), only trusted public package registries are allowed, and only `io.modelcontextprotocol.registry/publisher-provided` survives in `_meta` (4 KB limit). The registry is **not designed for self-hosting**. | https://modelcontextprotocol.io/registry/about.md · https://modelcontextprotocol.io/registry/quickstart · https://github.com/modelcontextprotocol/registry/blob/main/docs/reference/server-json/official-registry-requirements.md | 2026-10-23 (registry release v1.8.4) | High |
| 13 | The MCP spec now has a formal **feature lifecycle and deprecation policy**: Active / Deprecated / Removed states, a minimum 12-month deprecation window, and a public registry of deprecated features. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md · https://modelcontextprotocol.io/community/feature-lifecycle.md | 2026-07-28 | High |
| 14 | 43 MCP SEPs are **Final**; the workflow is PR-based with SEP numbering derived from PRs. | https://modelcontextprotocol.io/seps.md | 2026-10-24 (fetched) | High |
| 15 | **A2A spec is v1.0.0**; normative protobuf is the source of truth; bindings are JSON-RPC over HTTP (+SSE), gRPC and HTTP/REST; core objects are Agent Card, Task, Message, Part, Artifact. | https://a2a-protocol.org/llms.txt · https://a2a-protocol.org/latest/specification/ | 2026-10-24 (fetched) | High |
| 16 | A2A provides signed Agent Cards; `GetExtendedAgentCard` fetches richer metadata *after* authentication; discovery is at `/.well-known/agent-card.json`. | https://a2a-protocol.org/llms.txt | 2026-10-24 | High |
| 17 | A2A has official SDKs for Python, JS/TS, Java, Go, .NET and Rust, plus an official CLI (`a2a-cli`) that provides an Agent Skill and Agent Plugin. | https://a2a-protocol.org/llms.txt · https://github.com/a2aproject/a2a-cli | 2026-10-01 (CLI, per protocol-matrix chronology) | High |
| 18 | A2A and MCP are complementary and officially described as such: MCP = agent↔tool, A2A = agent↔agent. | https://a2a-protocol.org/llms.txt | 2026-10-24 | High |
| 19 | `@a2a-js/sdk` latest is **1.3.0** (Apache-2.0, published 2026-09-29). | https://registry.npmjs.org/@a2a-js/sdk | 2026-09-29 | High |
| 20 | `a2aproject/A2A` has 26,280 stars; latest release **v1.0.1** published 2026-05-28. | https://api.github.com/repos/a2aproject/A2A | 2026-05-28 | High |
| 21 | **OpenCode's canonical repo is `anomalyco/opencode`** (the `sst/opencode` path redirects). MIT, 220,944 stars, 29,537 forks, latest release **v1.19.4** published 2026-10-22. | https://api.github.com/repos/anomalyco/opencode · https://api.github.com/repos/anomalyco/opencode/releases/latest | 2026-10-22 | High |
| 22 | OpenCode ships **two plugin SDK packages**: V1 `@opencode-ai/plugin` (1.19.4, 2026-10-22) and V2 `@opencode/plugin` (**2.1.4, 2026-10-23**; package created 2026-09-02; 1,258 versions). Both MIT. | https://registry.npmjs.org/@opencode/plugin · https://registry.npmjs.org/@opencode-ai/plugin | 2026-10-23 | High |
| 23 | OpenCode V2 plugin entry point is `export default Plugin.define({...})` imported from `@opencode/plugin`; local plugins are auto-discovered from `.opencode/plugins/` (project) and `~/.config/opencode/plugins/` (global). | https://opencode.ai/v2/docs/build/plugins | 2026-10-24 | High |
| 24 | OpenCode V2 config key is `plugins` (array) in `opencode.json(c)` — V1 used the singular `plugin`. Entries accept npm names, `name@version`, scoped names, relative/absolute paths, `file://` URLs, or `{ package, options }` objects; `opencode plugin add\|list\|check\|update\|remove` manages them from the CLI. | https://opencode.ai/v2/docs/plugins | 2026-10-24 | High |
| 25 | OpenCode V2 exposes a **named hook-domain API** via `ctx.session.hook(name, cb)`. Hook names: `prompt`, `context`, `compaction`, `generate`, `title`, `model.request`, `http.request`, `http.response`, `experimental.ws.handshake`, `experimental.ws.send`, `experimental.ws.receive`, `retry`. | https://opencode.ai/v2/docs/build/plugins | 2026-10-24 | High |
| 26 | In OpenCode V2 the `compaction` hook may supply `event.result` of type `SessionCompactionResult { summary: string; native?; metadata?; tokens? }` — **a plugin can author the compaction summary itself.** | https://opencode.ai/v2/docs/build/plugins | 2026-10-24 | High |
| 27 | OpenCode V2 exposes a **permission gate** as a hook: `ctx.permission.hook("evaluate", cb)` where the callback may change `effect` to `allow` / `ask` / `deny`; an explicitly configured deny is final and never invokes the hook. | https://opencode.ai/v2/docs/build/plugins | 2026-10-24 | High |
| 28 | OpenCode V2 exposes a **shell gate**: `ctx.shell.hook("create.before", cb)` can rewrite command, cwd, timeout, shell and environment before execution. | https://opencode.ai/v2/docs/build/plugins | 2026-10-24 | High |
| 29 | OpenCode V2 compaction is checkpoint-based: `compaction.auto` (default `true`), `compaction.keep.tokens` (default **15,000** recent tokens kept beside the summary), `compaction.buffer` (default 15 % of the model limit, minimum 16k), manual compaction on request, and an experimental background-compaction mode. V1's `compaction.tail_turns` / `compaction.prune` are gone. | https://opencode.ai/v2/docs/compaction · https://opencode.ai/v2/docs/migrate-v1 | 2026-10-24 | High |
| 30 | OpenCode V2 speaks MCP **`2026-07-28`**: its MCP config exposes `protocol: "legacy" \| "auto" \| "2026-07-28"`, plus `type: local\|remote` and `codemode` (default `true`). OpenCode can host MCP servers server-side. | https://opencode.ai/v2/docs/mcp-servers | 2026-10-24 | High |
| 31 | A V1 OpenCode plugin can ship a **V2 `setup()` and a V1 `server()` in the same module**, so V1 plugins keep working during migration. | https://opencode.ai/v2/docs/migrate-v1 | 2026-10-24 | High |
| 32 | OpenCode has **no official plugin marketplace**. Distribution is npm + git specs via the CLI; the "OpenCode Plugin Marketplace" is an explicitly unaffiliated community project. | https://opencode.ai/docs/plugins/ · https://github.com/Tommertom/opencode-plugin-marketplace | 2026-10-24 | High |
| 33 | **Claude Code v2.1.310** (2026-10-23), 153,250 stars, 26,655 forks. | https://api.github.com/repos/anthropics/claude-code | 2026-10-23 | High |
| 34 | Claude Code exposes **33 hook events**, including `PreToolUse`, `PermissionRequest`, `PostToolUse`, `PostToolUseFailure`, `PostToolBatch`, `SessionStart`, `SessionEnd`, `SubagentStart`, `SubagentStop`, `PreCompact`, **`PostCompact`**, `PreModelSwitch`, `PostModelSwitch`, `TaskCreated`, `TaskCompleted`, `TeammateIdle`, `Stop`, `StopFailure`, `Elicitation`, `ElicitationResult`, `MessageDisplay`, `InstructionsLoaded`, `FileChanged`, `CwdChanged`, `ConfigChange`, `WorktreeCreate`, `WorktreeRemove`, `UserPromptSubmit`, `UserPromptExpansion`, `Notification`, `Setup`, `PermissionDenied`. | https://docs.claude.com/en/docs/claude-code/hooks | 2026-10-24 | High |
| 35 | Claude Code hook semantics: exit code **2 blocks** the action; `onFailure: "block"` (requires v2.1.295+) makes a crashed/timed-out hook block too; `async` / `asyncRewake` run in background; `once` removes a successful hook; `shell: "powershell"` is supported; hook types are `command`, `http`, `mcp_tool` and Agent-SDK callback. | https://docs.claude.com/en/docs/claude-code/hooks | 2026-10-24 | High |
| 36 | Claude Code plugins use a `.claude-plugin/plugin.json` manifest and can bundle skills, agents, hooks and MCP servers; marketplaces are git repos containing `.claude-plugin/marketplace.json`; Anthropic's official marketplace is auto-added on first interactive session; `/plugin` exposes a Discover tab with a **Context cost** estimate per plugin. | https://docs.claude.com/en/docs/claude-code/plugins | 2026-10-24 | High |
| 37 | **Codex CLI has a hooks framework** with `hooks.json` or inline `[[hooks.Event]]` tables in `config.toml`; handler types `command` and `mcp_tool` are supported (`prompt`/`agent` are parsed but skipped); `PreToolUse` can intercept Bash, `apply_patch` edits and MCP tool calls; plugins bundle hooks via `.codex-plugin/plugin.json` or a default `hooks/hooks.json`, with `PLUGIN_ROOT` / `PLUGIN_DATA` env vars. | https://www.codex-docs.com/en/docs/hooks | 2026-10-24 | High |
| 38 | Codex hooks require **persisted trust by hash** — changed hooks are skipped until re-reviewed — and can be bypassed for one run with `--dangerously-bypass-hook-trust`. Codex MCP servers live in `~/.codex/config.toml` under `[mcp_servers.<name>]`. | https://learn.chatgpt.com/docs/config-file/config-advanced.md · https://learn.chatgpt.com/docs/extend/mcp | 2026-10-24 | High |
| 39 | **The 2026 CLI-agent market consolidated around OpenCode, Claude Code, Codex CLI and Copilot CLI.** Roo Code was shut down and archived on **2026-05-15** (final release v3.54.0, 24,265 stars, Apache-2.0); Aider is dormant (last release v0.86.0 on 2025-08-09, last push 2026-05-22, 49,734 stars); Continue.dev was acquired by Cursor. Active: Cline (70,808 stars, desktop-v0.0.48 on 2026-10-17), Goose v1.56.0 (2026-10-23, 55,629 stars, Apache-2.0), Crush v0.98.2 (2026-10-23, 28,787 stars), Zed v1.25.3 (2026-10-23, 92,136 stars), Warp (AGPL-3.0), Copilot CLI v1.0.103 (2026-10-22, 11,293 stars), Codex (130,766 stars, Apache-2.0). | https://api.github.com/repos/RooCodeInc/Roo-Code · https://api.github.com/repos/Aider-AI/aider · https://api.github.com/repos/cline/cline · https://api.github.com/repos/block/goose · https://api.github.com/repos/charmbracelet/crush · https://api.github.com/repos/zed-industries/zed · https://api.github.com/repos/warpdotdev/Warp · https://api.github.com/repos/github/copilot-cli · https://api.github.com/repos/openai/codex | 2026-10-24 (GitHub API) | High (GitHub metadata) / Medium (shutdown rationale — secondary) |
| 40 | **Paperclip** (`paperclipai/paperclip`) is MIT, 106,790 stars, self-hosted, and positions itself as the app to "manage AI agents for work". Its four pillars: Agentic Task Manager, Org Chart for Agents, Agent Employee Training, Agentic OS. It **executes** agents: "Bring Your Own Agent — any agent, any runtime, one org chart. If it can receive a heartbeat, it's hired", with adapters for Claude Code, Codex, CLI agents and HTTP/webhook bots. | https://github.com/paperclipai/paperclip | 2026-10-24 | High |
| 41 | Paperclip's heartbeat execution is a **DB-backed wakeup queue** with coalescing, budget checks, workspace resolution, secret injection, skill loading and adapter invocation; mutating actions are recorded as durable activity; budgets have warning thresholds and hard stops; approval gates and tool-gateway action approvals exist. | https://github.com/paperclipai/paperclip (README) · https://docs.paperclip.ing/guides/day-to-day/costs/ · https://docs.paperclip.ing/guides/day-to-day/approvals/ | 2026-10-24 | High |
| 42 | Paperclip ships its **own plugin SDK** (issue orchestration APIs, UI slots, scheduled jobs, real-time streams, agent sessions) — i.e. it competes for the same "integration surface" slot that OpenCode plugins and Claude Code plugins occupy. | https://github.com/paperclipai/paperclip/blob/master/packages/plugins/sdk/README.md | 2026-10-24 | High |
| 43 | Other coordination-category applications with real traction in late 2026: **Multica** 53,707 stars; **Vibe Kanban** 28,445 stars (Apache-2.0, last push 2026-09-19); **Superset** 15,514 stars. | https://api.github.com/repos/multica-ai/multica · https://api.github.com/repos/BloopAI/vibe-kanban · https://api.github.com/repos/superset-sh/superset | 2026-10-24 | High (stars) / Medium (product scope — not individually deep-read) |
| 44 | **Claude Managed Agents** launched 2026-04-08 in beta. It virtualises an agent into **session (append-only log of everything that happened)**, **harness** (the loop) and **sandbox** (execution environment), so each can be swapped independently. Priced at $0.08 per session-hour plus tokens. | https://www.anthropic.com/engineering/managed-agents · https://claude.com/resources/articles/claude-managed-agents | 2026-04-08 | High |
| 45 | Claude Managed Agents permission policies gained an **`auto`** mode on 2026-09-10: the server evaluates each agent or MCP tool call and runs it, denies it, or pauses for approval; `agent.tool_use` / `agent.mcp_tool_use` events report an `evaluation` field. Self-hosted sandboxes (2026-05-19) and **MCP tunnels** to private MCP servers are available. **OpenTelemetry export for Messages API and Managed Agents sessions entered beta on 2026-10-14.** | https://docs.anthropic.com/en/release-notes/api · https://claude.com/resources/articles/claude-managed-agents-updates | 2026-10-14 | High |
| 46 | **OpenAI Agents API** entered public beta on **2026-09-10**, running the open-source Codex harness: `client.beta.agents.sessions.create(...)`, MCP tools over HTTP, `multi_agent: { enabled, max_concurrent_subagents }`, automatic context compaction, vaults, hosted/self-hosted sandboxes, `required_actions` for approvals, `OpenAI-Beta: agents=v1` header. **US-only data residency; no Zero Data Retention.** | https://openai.com/index/introducing-the-agents-api/ · https://help.openai.com/en/articles/20001551-agents-api-beta-faq | 2026-09-10 | High |
| 47 | The **OpenAI Agents SDK** models human-in-the-loop as **interruptions + resumable state**: the run records an approval interruption instead of executing, returns `interruptions` plus `state`, the app approves/rejects, and the same run resumes from that state. Guardrails come in input / output / tool variants. | https://developers.openai.com/api/docs/guides/agents/guardrails-approvals | 2026-10-24 | High |
| 48 | **Microsoft Agent 365** went GA on **2026-05-01** at $15/user/month as "the control plane for agents" — observe/govern/secure; it ships an **Agent Registry**, **Entra Agent ID** (agents as first-class identities with Conditional Access, RBAC/ABAC, access reviews, audit), Identity Protection for agents, and discovers local agents (OpenClaw first, Copilot CLI and Claude Code named as coming) into Defender/Intune. | https://www.microsoft.com/en-us/security/blog/2026/05/01/microsoft-agent-365-now-generally-available-expands-capabilities-and-integrations/ · https://learn.microsoft.com/en-us/microsoft-agent-365/overview | 2026-05-01 | High |
| 49 | **Vertex AI Agent Builder was rebranded "Gemini Enterprise Agent Platform" on 2026-04-22** (Cloud Next '26). Renames include Agent Engine → **Agent Runtime**, Agent Builder Sessions → **Agent Platform Sessions**, Memory Bank → **Agent Platform Memory Bank**, Vertex AI Studio → Agent Studio. | https://cloud.google.com/blog/products/ai-machine-learning/introducing-gemini-enterprise-agent-platform · https://docs.cloud.google.com/gemini-enterprise-agent-platform/vertex-ai-name-changes | 2026-04-22 | High |
| 50 | **OTel GenAI semantic conventions moved out of the core repo.** All `gen_ai.*` content was deprecated in core semconv v1.42.0 (2026-06-12) and now lives in `open-telemetry/semantic-conventions-genai`; the old `opentelemetry.io/docs/specs/semconv/gen-ai/` page is a "Moved" stub. **Every GenAI span/event/metric is still `Development` status — nothing is Stable, and the new repo has no tagged release.** | https://opentelemetry.io/docs/specs/semconv/gen-ai/ · https://github.com/open-telemetry/semantic-conventions-genai · https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-agent-spans.md | 2026-06-12 (move) | High |
| 51 | OTel GenAI agent-span conventions cover `create_agent`, `invoke_agent` (client + internal), `invoke_workflow`, `plan`, `execute_tool`, `execute_tool` for agent skills, `load_skill`, `read_skill_resource`, and command execution — i.e. a skill/knowledge-plane surface is already specified. | https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-agent-spans.md | 2026-10-24 (fetched) | High |
| 52 | A durable-execution observability proposal is active in the OTel GenAI repo: `gen_ai.agent.execution.id` as a logical run id across suspend/resume, `gen_ai.agent.paused`/`checkpointed`/`resumed` events, and `resumed_from.type`/`.id` to link a resumed segment to its boundary. | https://github.com/open-telemetry/semantic-conventions-genai/issues/462 | 2026-08-19 → 2026-10-22 | High |
| 53 | **IETF has three live agent-identity drafts**: `draft-klrc-aiagent-auth` (WIMSE identifier as the primary agent identifier, attestation-based credential issuance, short-lived credentials), `draft-pidlisnyi-aps` (Agent Passport System — Ed25519 self-signed passports, delegation chains that cannot widen, signed action receipts, four provenance tiers), and `draft-sweeney-wimse-credential-delegation` (DPoP-bound Delegation Tokens). | https://datatracker.ietf.org/doc/html/draft-klrc-aiagent-auth-01 · https://datatracker.ietf.org/doc/html/draft-pidlisnyi-aps-03 · https://datatracker.ietf.org/doc/draft-sweeney-wimse-credential-delegation/ | 2026-03 (aiagent-auth) / 2026 (aps-03) / 2026-10 (credential-delegation) | High |
| 54 | MCP `2026-07-28` documents **OpenTelemetry trace-context propagation** conventions for `_meta` keys (`traceparent`, `tracestate`, `baggage`) — SEP-414, Final. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | 2025-04-25 (SEP) / 2026-07-28 (in revision) | High |
| 55 | The protocol stack has settled into named layers: MCP (agent↔tool), A2A (agent↔agent), **AG-UI** (agent↔user-facing app, MIT, `ag-ui-protocol/ag-ui`, 16,792 stars, release 2026-10-21) and **Agent Client Protocol** (agent↔editor, Zed + JetBrains, `agentclientprotocol/agent-client-protocol`, schema-v1.26.0 on 2026-10-16). AG-UI has gained handshakes that let it "front for" agents behind MCP and A2A. | https://docs.ag-ui.com/agentic-protocols · https://api.github.com/repos/ag-ui-protocol/ag-ui · https://api.github.com/repos/agentclientprotocol/agent-client-protocol | 2026-10-24 | High (repos) / Medium (layer taxonomy — corroborated across official AG-UI docs and secondary matrices) |
| 56 | MCP servers reference implementations are healthy: `modelcontextprotocol/servers` at **v2.0.0** (2026-10-19, 91,541 stars), `modelcontextprotocol/registry` at **v1.8.4** (2026-10-23), `modelcontextprotocol/modelcontextprotocol` releases cut as `2026-07-28`. SDK: `@modelcontextprotocol/sdk` **1.32.2** (2026-10-23, MIT). | https://api.github.com/repos/modelcontextprotocol/servers · https://api.github.com/repos/modelcontextprotocol/registry · https://registry.npmjs.org/@modelcontextprotocol/sdk | 2026-10-23 | High |
| 57 | The IETF "AI-Agent Auth" draft also records that a **best-practices framework (AIAGENT-AUTH)** maps OAuth 2.0 + WIMSE onto agent interactions and **defines no new protocols**. | https://datatracker.ietf.org/doc/html/draft-pidlisnyi-aps-03 | 2026-10-24 (fetched) | High |
| 58 | Agent Shaker's own baseline, for gap analysis: MCP server with 28 tools (`get_my_identity` → `get_my_tasks` → `claim_task` → `complete_task`), A2A delegation with streaming + artifact sharing, shared markdown contexts, milestones, WebSocket live updates, Nuxt 4 frontend, Docker Compose deployment, per-agent API-key credentials and server-side MCP config generation. Latest release v0.5.6 (2026-10-11). | https://github.com/techbuzzz/agent-shaker · repo `README.md` | 2026-10-11 | High (local repo) |

> **Rows 59–70 were added in a second pass and are the most consequential cluster in this scan.** They establish that **OpenCode V2 is not only a host — it is becoming a coordination and governance plane** (Console, workspaces, team-wide policies, budgets, harnesses for other agents' runtimes, and its own external-context primitive). Any Phase 3 plan that treats OpenCode purely as "a tool to integrate with" is under-reading the threat.

| # | Claim | Evidence URL | Date | Confidence |
| --- | --- | --- | --- | --- |
| 59 | **The OpenCode Console is an optional hosted service** adding workspaces for team access, inference, provider gateway, **usage tracking and budget controls**, **team-wide policies that control OpenCode behaviour**, web search, and "Harnesses for using Console models from Claude Code, Codex, and other agents". Every account starts with a Personal workspace; `/connect` in OpenCode binds a workspace. | https://opencode.ai/v2/docs/console/ | 2026-10-24 | High |
| 60 | **OpenCode Console ships Harnesses for other agents' runtimes**: Claude Code (Anthropic Messages API), Codex (OpenAI Responses API), Hermes, Pi and a Raycast extension. A harness points at the Console gateway with an API key and usage bills to the workspace. Base URLs: `https://opencode.ai/inference` (pay-as-you-go) and `https://opencode.ai/inference/go` (Go subscription). Console also recognises the **native session headers** sent by Claude Code, Codex, Hermes and Pi, so no extra configuration is needed. | https://opencode.ai/v2/docs/console/harnesses/ | 2026-10-24 | High |
| 61 | **Console exposes its own MCP server at `https://opencode.ai/console/mcp`, authenticating with OAuth using dynamic client registration** — i.e. OpenCode still relies on DCR, which MCP `2026-07-28` deprecated in favour of Client ID Metadata Documents. | https://opencode.ai/v2/docs/console/harnesses/ · https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | 2026-10-24 | High (Console MCP) / High (DCR deprecation) — the divergence itself is the finding |
| 62 | **OpenCode Console has workspace budgets implemented as policy rules.** Levels: entire workspace, all members, tagged members, specific members, all service accounts, specific service accounts. A request must fit under *every* applicable budget (a larger allowance never overrides a smaller one). Amounts in USD, resets on the 1st of each month at 00:00 UTC, a $0 budget blocks all counted spend. On breach: **HTTP 429** with `Upstream request failed: Policy budget exceeded`; in-flight requests finish so spend can exceed the limit slightly. Spend covers paid Console models and own-provider models with a price set; free models do not count. Usage export as CSV for 7/30/90 days; the blocking policy is visible in Activity › Logs. | https://opencode.ai/v2/docs/console/budgets/ | 2026-10-24 | High |
| 63 | **Console policies are delivered to every OpenCode V2 client connected to the workspace and have "final authority over the member's own configuration", taking effect within about a minute.** Policy statements can `allow`, `deny` or `ask`; **"When several statements match, the last one wins. There is no specificity rule"**; Console-managed rules are appended after every authored statement, so "an organization allow lifts a lower-authority deny". Console policy applies *after* agent rules and saved approvals, overriding both an `ask` and an `allow`. Denials name the workspace ("Blocked by Acme's policy"). The Console plugin and the policy plugin **cannot be disabled** from the plugin list. On disconnect, statements are cleared and the last fetched statements stay in effect. | https://opencode.ai/v2/docs/policies/ | 2026-10-24 | High |
| 64 | **OpenCode V2 permissions are declarative rules** in `opencode.json(c)`: `{ "action": ..., "resource": ..., "effect": "allow" \| "deny" \| "ask" }`, with whole-value wildcards; **if no rule matches, the default is `ask`**. V1 used different field and action names (`permission`/`bash`/`task` → V2 `permissions`/`shell`/`subagent`). Access outside the active Location follows the `external_directory` permission. | https://opencode.ai/v2/docs/permissions/ | 2026-10-24 | High |
| 65 | **OpenCode V2 has its own external-context primitive: "References"** — named, attachable aliases pointing at a local directory or a Git repository (with optional branch), each with a `description` that is injected into agent instructions as guidance, an optional `hidden` flag, and background refresh (a checkout is eligible for refresh if the last attempt was ≥24 h ago; clone/refresh never blocks a prompt). References **do not grant extra tool permissions** — external reads still require the applicable permission. | https://opencode.ai/v2/docs/references/ | 2026-10-24 | High |
| 66 | **OpenCode V2 has a built-in session "warming" (keep-warm) mechanism**: `"warming": true` warms a recently active session after **four minutes** without a model request, continuing every four minutes until **30 minutes** have passed since the latest non-warming request; the object form configures the warming prompt, idle interval and active duration. | https://opencode.ai/v2/docs/warming/ | 2026-10-24 | High |
| 67 | OpenCode V2 plugin context exposes an **event stream** via `ctx.event.subscribe({ signal })` returning an async iterable of events, but the concrete event-type union was not enumerated in this scan. | https://opencode.ai/v2/docs/build/plugins | 2026-10-24 | High (API exists) / Unverified (event names) |
| 68 | OpenCode V2's plugin API is documented in two dialects — a **plain TypeScript** dialect and an **Effect** dialect (`@opencode/plugin/effect`), plus an RPC contract (`./rpc` export) and a CLI/TUI dialect (`@opencode/plugin/tui`). | https://opencode.ai/v2/docs/build/plugins/effect/ · https://opencode.ai/v2/docs/build/plugins/rpc/ · https://opencode.ai/v2/docs/build/plugins/cli/ | 2026-10-24 | High |
| 69 | OpenCode V2 also documents **ACP** support (Agent Client Protocol) as a first-class CLI integration alongside plugins, skills and references. | https://opencode.ai/v2/docs/cli/acp/ | 2026-10-24 | High |
| 70 | The consolidation trend has a **direct competitor analogue in OpenCode itself**: Console Harnesses (Claude Code, Codex, Hermes, Pi, Raycast) mirror Paperclip's adapter model (Claude Code, Codex, CLI agents, HTTP/webhook bots). Two independent projects converged on "bring your own agent runtime into a console". | https://opencode.ai/v2/docs/console/harnesses/ · https://github.com/paperclipai/paperclip | 2026-10-24 | High |

---

## Section 2 — Detailed notes

### A1. OpenCode — the Phase 3 integration target, re-grounded

#### A1.1 Identity, versions, licensing

OpenCode is the largest-starred agent host in the comparison set (220,944 stars / 29,537 forks, MIT). The repository moved from `sst/opencode` to **`anomalyco/opencode`**; the old path redirects and both report identical metadata, so treat `anomalyco/opencode` as canonical and record the redirect so the packaging decision is not built on a stale URL.

The latest stable line is **v1.19.4, published 2026-10-22** — npm `opencode-ai` 1.19.4, `@opencode-ai/plugin` 1.19.4, `@opencode-ai/sdk` 1.19.4.

#### A1.2 There are now two plugin generations, and the brief's "npm package" question has a trap

| | V1 | V2 |
| --- | --- | --- |
| npm package | `@opencode-ai/plugin` (1.19.4) | **`@opencode/plugin`** (2.1.4, created 2026-09-02) |
| Entry point | `export const MyPlugin: Plugin = async ({...}) => ({...})` | `export default Plugin.define({...})` |
| Config key | `plugin: [...]` | `plugins: [...]` |
| Hook model | `tool.execute.before` / `tool.execute.after` / `event` / `chat.message` / `chat.params` / `permission.ask` | **named hook domains** via `ctx.session.hook(name, cb)` |
| Local discovery | `.opencode/plugin/`, `~/.config/opencode/plugin/` | `.opencode/plugins/`, `~/.config/opencode/plugins/` |

A Phase 3 plan that says "publish an OpenCode plugin as `@opencode-ai/plugin`" is targeting the previous generation. V2 is 1258 versions old and moving fast (2.1.4 published the day before this scan).

V2 also has a **CLI-plugin split**: plugins that expose a TUI component are loaded from `cli.json` and run locally in the terminal, staying active when the CLI connects to a *remote* OpenCode server; the CLI obtains the active plugin list from the connected server. The MCP-facing plugin is a *server* plugin (`index.ts` + optional `tui.ts` in the same package). This matters for Agent Shaker: a plugin that must work against a remote/server-hosted OpenCode has to be authored as a server plugin with the RPC export (`./rpc`) optional but recommended for shared contracts.

#### A1.3 The exact hook surface a Phase 3 integration can bind to

From the V2 plugin reference, the `SessionHooks` interface is:

```ts
interface SessionHooks {
  prompt: SessionPrompt
  context: SessionContextHook
  compaction: SessionContextHook & { result?: SessionCompactionResult }
  generate: SessionContextHook
  title: SessionRequestHook & { result?: string }
  "model.request": SessionModelRequestHook
  "http.request": SessionHttpRequestHook
  "http.response": SessionHttpResponseHook
  "experimental.ws.handshake": SessionWebSocketHandshakeHook
  "experimental.ws.send": SessionWebSocketSendHook
  "experimental.ws.receive": SessionWebSocketReceiveHook
  retry: SessionRetryHook
}
```

and the registration API:

```ts
ctx.session.hook<Name extends keyof SessionHooks>(
  name: Name,
  callback: (event: SessionHooks[Name]) => Promise<void> | void,
  options?: { providerID?: string },
): Promise<Registration>
```

Three observations that are directly load-bearing:

1. **`compaction` is a first-class hook and the plugin may own the summary.** `SessionCompactionResult { summary: string; native?: Record<string, unknown>; metadata?: Record<string, unknown>; tokens?: TokenUsage }`. This is the single most important finding for Phase 3: the compaction summary is not read-only, it is an output a plugin can populate. Agent Shaker's shared markdown contexts are a natural source for that summary, and `metadata`/`tokens` give a place to carry the coordination-plane identifiers (project, task, agent) across a compaction boundary.
2. **`context` and `compaction` share the same hook shape** (`SessionContextHook`), so a plugin that observes the outgoing context can observe the compaction request too. `context` is the natural place to *inject* shared truth into the prompt; `compaction` is the natural place to *capture* it into the summary.
3. **Permission and shell are hooks too, not just observers.** `ctx.permission.hook("evaluate", cb)` can flip `effect` between `allow`/`ask`/`deny`, with the rule that an explicitly configured deny is final and does not invoke the hook. `ctx.shell.hook("create.before", cb)` can rewrite `command`, `cwd`, `timeout`, `shell` and env. A coordination plane that wants to *enforce* a lease or a dependency ("you cannot edit files outside the assigned worktree") has a native gate here, not a wrapper hack.

There is also a `retry` hook with `RetryDecision = { retry: false } | { retry: true; delay: number }`, letting a plugin make a terminal failure retryable or veto a proposed retry — with the documented caveat that context-overflow recovery is *separate* because it compacts rather than retrying.

#### A1.4 Compaction behaviour (the part Phase 3 most likely gets wrong)

OpenCode V2 compaction is **checkpoint-based**:

- `compaction.auto` — default `true`. Compacts automatically near the context limit, and recovers once when a provider rejects a request as too long. Manual compaction always works.
- `compaction.keep.tokens` — default **15,000**: approximate recent conversation kept *beside* the summary so exact recent details survive.
- `compaction.buffer` — default 15 % of the model limit, at least 16k tokens to keep free below the limit. A buffer at least as large as the model limit is ignored.
- `experimental.background_compaction` — compaction normally pauses the session while the summary is written; background mode avoids that.

The document shape is `[ system prompt ][ older conversation ] [ recent 15k ] [ summary instructions ]` → after `[ system prompt ][ summary of older + recent ] [ recent 15k ] [ pending work ]`.

This is a very specific contract, and it is the strongest technical argument in this whole scan for why the Phase 3 brief's "compaction hook" is the right integration point: OpenCode explicitly separates *what is retained verbatim* (`keep.tokens`) from *what is summarised*, and the summary is a structured artifact with a stated job — preserve pending work, blockers, next moves, and the relevant file set. That is precisely the content Agent Shaker already stores as shared markdown contexts and task state.

#### A1.5 MCP and config

OpenCode V2's MCP configuration nests servers under `mcp.servers`, with `type: local|remote`, `codemode` (default `true`) and a `protocol` selector of `legacy | auto | 2026-07-28`. The last value means **OpenCode already speaks the current MCP revision**, including the stateless model — i.e. a Phase 3 integration cannot assume the 2024-11-05 session semantics when talking to an OpenCode-hosted MCP server.

Config file is `opencode.json(c)` with schema at `https://opencode.ai/config.json`. Global plugin discovery mirrors project discovery under `~/.config/opencode/plugins/`. A `plugins/` directory beside a project-root `opencode.json(c)` is **not** auto-discovered — it must be configured explicitly or moved under `.opencode/`.

#### A1.6 Marketplace and popularity

There is **no official OpenCode plugin marketplace**. Distribution is npm (regular and scoped) and git specs (`github:`, `git+ssh://`, `git+https://`), installed via `opencode plugin add`. The commonly-referenced "OpenCode Plugin Marketplace" (`Tommertom/opencode-plugin-marketplace`) explicitly states it is *independent, community-driven and not affiliated with, endorsed by, or officially connected to OpenCode (opencode.ai) or Anomaly Co*.

Popularity indicators: 220,944 stars (2026-10-24), 29,537 forks, MIT, `@opencode/plugin` at 1,258 published versions since 2026-09-02, and `opencode-ai` npm at 12,256 total versions.

#### A1.7 OpenCode V2 is not just a host — it is becoming a coordination and governance plane

This is the single most consequential finding of the scan, and it was not in the brief's framing. OpenCode V2 ships a **Console** — an optional hosted service — whose feature list reads like Agent Shaker's roadmap:

> "Workspaces for sharing access with your team · Inference for both proprietary and open source models · Providers for connecting your own provider accounts through the Console gateway · **Usage tracking and budget controls** · **Deploy team wide policies to control OpenCode behavior** · Web search · **Harnesses for using Console models from Claude Code, Codex, and other agents**"

Five sub-systems matter to Agent Shaker:

**1. Console Harnesses — the host-as-hub pattern.** Console works with *other* agent runtimes, not just OpenCode: Claude Code, Codex, Hermes, Pi and a Raycast extension. Harnesses authenticate with a Console API key (personal keys `oc_sk_...` or service-account keys for shared machines/CI) and call the Console gateway at `https://opencode.ai/inference` (pay-as-you-go) or `https://opencode.ai/inference/go` (Go subscription, $10/$40 monthly) with the path for the API the harness speaks. Console "recognizes the native session headers sent by Claude Code, Codex, Hermes, and Pi, so they need no extra configuration" — and OpenCode's own stable session header is `x-opencode-session: ses_...`, which "Go requires" so the Console can route a conversation and reuse its prompt cache.

This is the same convergence as Paperclip's adapter model (Claude Code, Codex, CLI agents, HTTP/webhook bots), arrived at independently: **the 2026 pattern is a console that hosts other agents' runtimes.** Agent Shaker's A2A delegation surface and Paperclip's adapter plugins are two answers to the same problem; OpenCode's Console is a third, and it is bundled with the largest agent host in the world.

**2. Team-wide policies with final authority.** Console policy statements are compiled from the workspace's Providers and Tools policies and returned with the workspace configuration; OpenCode appends them **after every authored statement**, so they "have the final say: an organization allow lifts a lower-authority deny". They apply *after* agent rules and saved approvals, overriding both an `ask` and an `allow`. The matching model is explicitly **last-one-wins with no specificity rule** — a deliberate contrast with Claude Code's specificity-ranked matcher rules. Changes are delivered on the next fetch, roughly once a minute; on disconnect the last fetched statements stay in effect; the Console and policy plugins cannot be disabled.

**3. Budgets as first-class policy.** Workspace / member / member-tag / specific-member / service-account levels; a request must fit under every applicable budget, and a larger allowance never overrides a smaller one. Breach produces **HTTP 429** with `Upstream request failed: Policy budget exceeded`, checked when a request *starts*, so in-flight requests finish and spend can land slightly above the limit. The blocking policy is recoverable from Activity › Logs; spend exports as CSV.

**4. References — OpenCode's own answer to external shared context.** A `references` map in `opencode.json(c)` gives named, attachable aliases to a local directory or a Git repository (with optional branch), each with a `description` that is injected into agent instructions as *guidance* ("References with a `description` appear in agent instructions with their alias and resolved path"), an optional `hidden` flag that hides it from interactive selectors while keeping it in the API and in instructions, background refresh with a 24-hour eligibility window that never blocks a prompt, and explicit permission semantics — "References do not grant extra tool permissions."

This is the feature that most directly overlaps Agent Shaker's shared markdown contexts, and the comparison is instructive:

| | OpenCode References | Agent Shaker shared contexts |
| --- | --- | --- |
| Storage | Local directory or Git checkout on the agent's machine | Server-side, Postgres |
| Scope | Per-project, per-client | Cross-agent, cross-project, cross-vendor |
| Retrieval | Attach an alias; non-recursive listing of immediate files | Full-text search over markdown |
| Freshness | Background clone/refresh, ≥24 h eligibility | Server-side, immediate |
| Identity | None — just a path alias | Agent identity + task + project scoping |
| Multi-agent | No — one agent's filesystem | Yes — the whole point |

The differentiation is **server-side, searchable, multi-agent, identity-scoped** context. The similarity is that both hosts now believe "named external context with machine-readable guidance" is a first-class primitive — which validates the shape of Agent Shaker's feature and narrows its defensible edge to the multi-agent/identity/audit properties.

**5. Warming — a built-in liveness cadence.** `"warming": true` warms a recently active session after **four minutes** without a model request, repeating every four minutes until **30 minutes** have passed since the latest non-warming request; the object form configures the warming prompt, idle interval and active duration.

This is directly relevant to the brief's "heartbeat on session lifecycle" goal: **OpenCode already defines its own idle cadence (4-minute warm interval, 30-minute window).** Any Agent Shaker lease TTL has to be chosen against that cadence, and any heartbeat derived from OpenCode has to distinguish "the session is idle because the user is thinking" from "the session is gone". It also reframes the question: the open item is not "does OpenCode emit a session-lifecycle event" (see Section 5, item 27) but "what is the authoritative liveness signal in a host that keeps its own sessions warm?"

**Net assessment for Phase 3.** OpenCode's Console occupies a lot of the space Agent Shaker is claiming — budgets, policies, workspaces, multi-agent runtime hosting. What OpenCode's Console does **not** have, and what Agent Shaker's design already has:

- Cross-*vendor* agent coverage (OpenCode's harnesses are Claude Code / Codex / Hermes / Pi / Raycast — a curated list, not an open protocol).
- A protocol-neutral contract (A2A + MCP) versus harness-specific adapters.
- Agent-identity-scoped shared state, an append-only activity model, and task-level dependency/claim semantics.
- Self-hosting as a first-class property.

The honest reading: **Agent Shaker's defensible position in late 2026 is "the vendor-neutral, self-hosted shared-truth layer for agents"** — and the Console's existence is evidence that the category is real, not evidence that the need has been met.

---

### A2. Other CLI / terminal agent hosts

#### A2.1 Claude Code — the reference implementation for hooks-as-gates

`anthropics/claude-code` is at **v2.1.310** (2026-10-23), 153,250 stars. Its hooks reference is the most complete public contract in the ecosystem, and it is the benchmark any other host's hook surface should be read against.

**Event names (33):** `SessionStart`, `Setup`, `UserPromptSubmit`, `UserPromptExpansion`, `PreToolUse`, `PermissionRequest`, `PermissionDenied`, `PostToolUse`, `PostToolUseFailure`, `PostToolBatch`, `Notification`, `MessageDisplay`, `SubagentStart`, `SubagentStop`, `TaskCreated`, `TaskCompleted`, `Stop`, `StopFailure`, `TeammateIdle`, `InstructionsLoaded`, `ConfigChange`, `CwdChanged`, `DirectoryAdded`, `FileChanged`, `WorktreeCreate`, `WorktreeRemove`, `PreCompact`, `PostCompact`, `PreModelSwitch`, `PostModelSwitch`, `Elicitation`, `ElicitationResult`, `SessionEnd`.

**Blocking semantics:** exit code **2** stops the action on events that can block; other non-zero codes are *non-blocking errors* and the action proceeds. `onFailure: "block"` (Claude Code ≥ v2.1.295) inverts this so a crashing or timing-out hook blocks the action. `async: true` runs the hook in the background without blocking; `asyncRewake: true` runs in the background and wakes Claude on exit 2, delivering stderr as a system reminder. `once: true` removes the hook after its first *successful* run. `shell` accepts `bash` or `powershell` (defaults to `powershell` on Windows without Git Bash). Hook types: `command`, `http`, `mcp_tool`, and an Agent-SDK callback form.

**Per-event decision control** is explicit: `PreToolUse` uses `hookSpecificOutput.permissionDecision` (`allow`/`deny`/`ask`/`defer`) + `permissionDecisionReason`; `PermissionRequest` uses `decision.behavior` (`allow`/`deny`); `Elicitation`/`ElicitationResult` use `action` (`accept`/`decline`/`cancel`) + `content`; `TaskCreated` can cancel the task; `Stop`/`SubagentStop` accept `decision: "block"` with a reason.

Three details matter for a coordination plane:

- **`PostCompact` exists.** Claude Code fires it *after* compaction completes, alongside `PreCompact` before. That is the same integration point as OpenCode's `compaction` hook, in a second host.
- **`Elicitation` / `ElicitationResult` hooks exist.** A host can observe and rewrite an MCP server's request for user input — and, per MCP `2026-07-28`, elicitation is now the MRTR-shaped "ask the user" path.
- **Timeout on `PreToolUse` does *not* block the tool call.** A stalled hook renders no decision; only `onFailure: "block"` changes that. A coordination-plane gate that must never fail open has to set it explicitly.

#### A2.2 Codex CLI

Codex has a first-class hooks framework. Hooks are discovered from `hooks.json` or inline `[[hooks.Event]]` tables in `config.toml`, next to each active config layer (`~/.codex/hooks.json`, `~/.codex/config.toml`, `<repo>/.codex/hooks.json`, `<repo>/.codex/config.toml`); if both exist in one layer, both load and Codex warns. Events include `SessionStart` (matcher `^compact$` is the compaction hook), `UserPromptSubmit`, `PreToolUse`, `PostToolUse`. Handler types `command` and `mcp_tool` are supported; `prompt` and `agent` are parsed but skipped. An `mcp_tool` hook calls a tool on an *already-connected* MCP server with structured arguments — "let the `scanner` MCP server scan each patch after Codex writes or edits files".

Plugins bundle hooks through `.codex-plugin/plugin.json` (a `hooks` entry pointing at paths or inline hook objects) or a default `hooks/hooks.json` inside the plugin root; plugin hook commands receive `PLUGIN_ROOT` and `PLUGIN_DATA`. Codex also ships **enterprise hooks**: admin-defined MCP hooks that run on the *cloud coordinator* for supported lifecycle and tool events — with the explicit warning that "MCP hooks do not provide a complete Compliance API audit trail."

Two differentiators for a coordination plane: (a) the `mcp_tool` hook type lets the coordination plane be invoked *by name* as an MCP tool rather than via a shell command, which avoids the shell-out trust problem entirely; (b) Codex requires **persisted hook trust by hash** — a changed hook is skipped until reviewed, with `--dangerously-bypass-hook-trust` for one-off runs.

#### A2.3 GitHub Copilot CLI and the rest

`github/copilot-cli` is at **v1.0.103** (2026-10-22), 11,293 stars — a *young* project next to OpenCode's 220k, but it is a Microsoft-hosted distribution channel and GitHub is where the MCP registry, `awesome-copilot` (40,369 stars) and the plugin ecosystem live.

The rest of the field, measured via the GitHub API on 2026-10-24:

| Host | Latest release | Date | Stars | License | Note |
| --- | --- | --- | --- | --- | --- |
| OpenCode (`anomalyco/opencode`) | v1.19.4 | 2026-10-22 | 220,944 | MIT | V2 plugin API + MCP 2026-07-28 |
| Claude Code | v2.1.310 | 2026-10-23 | 153,250 | — | 33 hook events, plugin marketplace |
| Codex | — (no GH releases) | — | 130,766 | Apache-2.0 | hooks + `config.toml` MCP |
| Zed | v1.25.3 | 2026-10-23 | 92,136 | — | steers Agent Client Protocol |
| Warp | v0.2026.06.03 | 2026-06-03 | 65,671 | AGPL-3.0 | last release ~4 months old |
| Goose | v1.56.0 | 2026-10-23 | 55,629 | Apache-2.0 | |
| Aider | v0.86.0 | 2025-08-09 | 49,734 | Apache-2.0 | dormant; last push 2026-05-22 |
| Cline | desktop-v0.0.48 | 2026-10-17 | 70,808 | Apache-2.0 | active |
| Crush | v0.98.2 | 2026-10-23 | 28,787 | — | |
| Copilot CLI | v1.0.103 | 2026-10-22 | 11,293 | — | |
| Roo Code | v3.54.0 | 2026-05-15 | 24,265 | Apache-2.0 | **shut down, repo archived** |

**The trend in the host market is consolidation, not expansion.** Roo Code shut down on 2026-05-15 with v3.54.0 as the final release; its own docs point to ZooCode (community fork) and Cline, and the team pivoted to a cloud agent ("Roomote"). Continue.dev was acquired by Cursor. Aider's release cadence effectively stopped in August 2025. The survivors — OpenCode, Claude Code, Codex, Copilot CLI, Cline, Goose, Crush, Zed — are the hosts worth integrating, and the integration surface that matters is **hooks + plugins + MCP**, in that order.

#### A2.4 Where the interop protocols sit

- **AG-UI** (`ag-ui-protocol/ag-ui`, MIT, 16,792 stars, release 2026-10-21) — agent↔user-facing-app event protocol, ~16 standard event types over SSE/WebSockets/webhooks. Recently added handshakes so it can "front for" agents behind MCP and A2A. First-party integrations reported in Microsoft Agent Framework, Google ADK, AWS Strands and Bedrock AgentCore. **This is the transport Agent Shaker's Nuxt UI should speak** if it wants to embed a *foreign* agent, rather than only hosting MCP tools.
- **Agent Client Protocol** (`agentclientprotocol/agent-client-protocol`, Apache-2.0, schema-v1.26.0 on 2026-10-16) — agent↔editor, "LSP for agents", steered by Zed and JetBrains. Runs Claude Code, Codex CLI and Gemini CLI inside Zed and JetBrains.

---

### A3. Coordination / control-plane competitors

#### A3.1 Paperclip — the direct competitor, and the one that changed the framing

`paperclipai/paperclip`: MIT, **106,790 stars**, self-hosted, docs at `docs.paperclip.ing`. It describes itself as "the app people use to manage AI agents for work" and organises itself around four pillars:

| Pillar | What it covers |
| --- | --- |
| Agentic Task Manager | Tasks, approvals & review gates, proactive agent coworkers, auditable routines, verification from diffs/screenshots/tests |
| Org Chart for Agents | Mixed human + agent org chart, responsibilities, delegation, governance, scoped secrets, connection permissions, responsible-user identities |
| Agent Employee Training | Skill Studio & shared org-wide skills, evals & saved test runs, active learning loops, performance reviews, skill version history |
| Agentic OS | Cross-provider runtime, sandboxing, integrations & MCP servers, SSO/GRC/RBAC/cost controls, internal trace collection, run history & opt-in tracing |

**The architectural difference that matters for Agent Shaker:** Paperclip *executes* agents. "Bring Your Own Agent — any agent, any runtime, one org chart. **If it can receive a heartbeat, it's hired.**" Its execution path is a DB-backed wakeup queue with coalescing, budget checks, workspace resolution, secret injection, skill loading and adapter invocation; there are adapters for Claude Code, Codex, CLI agents (Cursor/Gemini/bash) and HTTP/webhook bots (OpenClaw, Grok), plus external adapter plugins.

So the two products meet at exactly one point: **the heartbeat**. Agent Shaker's "worker heartbeat with lease" is a *lease on a worker slot*; Paperclip's heartbeat is a *wakeup trigger that drives an agent to do work*. Both assume the agent will be woken and will then report back. Paperclip's implementation is the reference for what "heartbeat" has to survive: coalescing, budget checks, secret injection, bounded recovery, and surfacing cases that need human action.

Paperclip's other table-stakes surfaces, all of which a competitor-scan should treat as baseline: durable **activity log** of mutating actions, heartbeat state changes, cost events, approvals, comments and work products; **budgets** by company/agent/project/goal/issue/provider/model with warning thresholds and hard stops; **approval gates** (board approval workflows, execution policies with review/approval stages, decision tracking, agent pause/resume/terminate); **tool-gateway action approvals**; and a **plugin SDK** with issue-orchestration APIs, UI slots, scheduled jobs, real-time streams and agent sessions.

Its stated non-goals are explicit and useful for positioning: "Not just a chatbot", "Not an agent framework", "Not just a workflow builder", "Not a prompt manager", "Not limited to one agent", "Not only for code review".

#### A3.2 The rest of the category

| Product | Signal (2026-10-24) | Posture |
| --- | --- | --- |
| **Multica** (`multica-ai/multica`) | 53,707 stars | agent workspace / CLI companion (`multica-cli`) |
| **Vibe Kanban** (`BloopAI/vibe-kanban`) | 28,445 stars, Apache-2.0, last push 2026-09-19, last release 2026-04-24 | kanban-style agent orchestration; community-maintained direction |
| **Superset** (`superset-sh/superset`) | 15,514 stars | agent IDE/workbench |
| **AgentsRoom** | closed-source desktop app; `AgentsRoomDev/electron-mcp-for-agentsroom` (13 stars, 2026-10-24) | MCP-facing desktop |
| **OpenHands** (`All-Hands-AI/OpenHands`) | 91,728 stars, MIT, v1.33.0 (2026-10-23) | executes agents, adjacent to Paperclip's side |
| **OpenAI Agents API / Agents SDK** | public beta 2026-09-10 | managed execution, sessions, approvals |
| **Claude Managed Agents** | beta 2026-04-08 | managed execution, self-hosted sandboxes, MCP tunnels |
| **Microsoft Agent 365** | GA 2026-05-01 | *control plane for* agents — observe/govern/secure, registry, Entra Agent ID |
| **Gemini Enterprise Agent Platform** | launched 2026-04-22 (Vertex AI Agent Builder rebrand) | build/scale/govern/optimise agents |

The category split that emerges is worth stating plainly, because it determines Agent Shaker's defensible position:

- **Execution products** (Paperclip, OpenAI Agents API, Claude Managed Agents, OpenHands) *run* agents.
- **Governance products** (Microsoft Agent 365, Entra Agent ID) *observe and constrain* agents across a fleet, tied to enterprise IAM.
- **Context/coordination products** (Agent Shaker, Multica, Vibe Kanban, Superset, AgentsRoom) *store the shared truth* between agents.

Agent Shaker's "does not execute agents" positioning is therefore not a weakness to apologise for — it is the gap between the first two, and the second group (governance) is now actively buying the same shared-truth layer. The risk is the *third* group being out-built by the first: Paperclip's category has already grown org charts, budgets, approvals and a plugin SDK.

---

### B1. MCP — the current revision in detail

Everything in this subsection is from the `2026-07-28` changelog and the extensions overview, unless marked.

#### B1.1 The protocol became stateless

- Protocol-level sessions and the `Mcp-Session-Id` header are removed. List endpoints no longer vary per connection.
- The `initialize` / `notifications/initialized` handshake is gone. Every request carries protocol version and client capabilities in `_meta` (`io.modelcontextprotocol/protocolVersion`, `io.modelcontextprotocol/clientCapabilities`); clients SHOULD identify themselves per request (`io.modelcontextprotocol/clientInfo`); servers SHOULD identify themselves in each result's `_meta` (`io.modelcontextprotocol/serverInfo`). Version mismatch returns `UnsupportedProtocolVersionError`.
- Cross-call state must use **explicit, server-minted handles passed as ordinary tool arguments** (SEP-2567). This is the direct consequence for Agent Shaker: a long-running claim/lease can no longer be keyed on an MCP session id; it must be keyed on an Agent Shaker-issued handle that travels as a tool argument.

#### B1.2 `server/discover` and version negotiation

Servers MUST implement `server/discover` to advertise supported protocol versions, capabilities and identity. Clients MAY call it before any other request for up-front version selection, or use it as a backward-compatibility probe on STDIO. **This is the single most useful new primitive for a coordination plane**: an Agent Shaker MCP server can advertise "I speak 2026-07-28 and these extensions" and a client can branch on it before issuing anything.

#### B1.3 MRTR replaces server-initiated requests

Server-initiated requests (`roots/list`, `sampling/createMessage`, `elicitation/create`) are replaced by the **Multi Round-Trip Requests** pattern: the server returns `InputRequiredResult` (`resultType: "input_required"`) whose `inputRequests` carries what it needs; the client responds with `inputResponses` on a **retry of the original request**. `resultType` is now required on all results (`"complete"` or `"input_required"`); clients MUST treat results from earlier-protocol servers that omit it as `"complete"`.

Consequences for Agent Shaker: an approval gate that used to be "elicitation/create" must now be "return `input_required` with an `inputRequests` entry, then accept the retried request". The elicitation completion notification and `elicitationId` were removed; servers needing to correlate across retries encode their own identifier in `requestState`. SSE stream resumability and message redelivery (`Last-Event-ID`) are also removed — a broken response stream loses the in-flight request and the client MUST re-issue it with a new request id. That last point has a direct operational implication for any coordination plane with long-running work: **idempotency keys are now mandatory, not optional.**

#### B1.4 Headers, caching, deterministic ordering

- `Mcp-Method` and `Mcp-Name` are required on Streamable HTTP POST requests; `x-mcp-header` carries custom headers from tool parameters.
- `tools/list`, `prompts/list`, `resources/list`, `resources/read`, `resources/templates/list` results MUST carry `ttlMs` (freshness hint) and `cacheScope` (`"public"` or `"private"`) via the new `CacheableResult` interface, complementing `listChanged` notifications.
- Servers SHOULD return tools from `tools/list` in a **deterministic order** to enable client-side caching and improve LLM prompt-cache hit rates. A coordination plane exposing 28 tools (Agent Shaker's current count) has a direct prompt-token cost, and the spec now tells servers how to make that cost stable and cacheable.

#### B1.5 Tool results and schemas

`inputSchema` and `outputSchema` are loosened to allow any JSON Schema 2020-12 keywords; `structuredContent` may be any JSON value; `$ref` resolution requirements and composition-keyword resource bounds were added (SEP-2106). This makes `outputSchema` + `structuredContent` a viable way to publish *typed* coordination state (claim results, lease objects, standup payloads) rather than prose blobs.

Tool annotations (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`) remain part of the tool schema and are the mechanism a client uses to decide whether to prompt. For a coordination plane whose tools mutate shared state, declaring `destructiveHint`/`idempotentHint` correctly is what lets a host auto-approve safe reads.

#### B1.6 Extensions: Tasks, Apps, Skills

The **extensions framework** (SEP-2133) is now the mechanism for optional capability, and `extensions` is a field on both client and server capabilities. Live extensions:

| Extension | SEP | Status | What it gives |
| --- | --- | --- | --- |
| `io.modelcontextprotocol/tasks` | SEP-2663 | Final | Server-side long-running work: `tasks/get` polling, `tasks/update` client→server input, handles may be returned unsolicited |
| `io.modelcontextprotocol/skills` | SEP-2640 | Final | `skills/list` + `skills/get`; skill files served via `resources/read`; digest+size verification by hosts |
| MCP Apps | SEP-1865 | Final | Interactive UI resources shipped by servers to hosts |

The **Skills extension** is directly relevant to Agent Shaker's "shared markdown contexts" and to a Phase 3 skills story: a skill is a `SKILL.md` with YAML frontmatter (`name`, `description`) plus supporting files; the manifest MUST include `SKILL.md` and every supporting file with URI, SHA-256 digest and byte size; hosts MUST verify digest/size/frontmatter before use and MUST NOT retrieve files ahead of need; persisted approval binds to the complete set of file URIs and digests and is revoked if any file changes; servers SHOULD NOT exceed 512 files or 16 MiB per skill. **Skills-over-MCP is the standard way to ship "how to work here" to an agent that already talks MCP** — which is a plausible future distribution channel for Agent Shaker's coordination playbooks.

The **Tasks extension** is the natural home for Agent Shaker's long-running coordination primitives (a claim that may block on a dependency, a lease renewal, a standup that must wait on other agents) — with the caveat that tasks are now an *extension*, not core, so a client must advertise support.

#### B1.7 Authorization and the registry

- Authorization servers SHOULD include `iss` in authorization responses per RFC 9207, and MCP clients MUST validate a present `iss` against the recorded issuer before redeeming the code (SEP-2468).
- Clients MUST key persisted credentials by the issuer identifier, MUST NOT reuse them with a different authorization server, and MUST re-register when the authorization server changes (SEP-2352).
- Clients MUST specify an `application_type` during Dynamic Client Registration (SEP-837).
- **OAuth 2.0 DCR (RFC 7591) is deprecated** in favour of **Client ID Metadata Documents**. OpenCode V2 already ships CIMD support with its own client metadata document at `https://opencode.ai/oauth/opencode/client.json`.
- Remote server auth is OAuth-based; the registry's own auth modes include GitHub OIDC token exchange, OIDC token exchange and signature token exchange.

**Official MCP Registry** (`registry.modelcontextprotocol.io`, in preview, repo release v1.8.4 on 2026-10-23): `server.json` with `name` (reverse-DNS, exactly one slash, 3–200 chars, case-sensitive), `version` (unique per publish, semver, no ranges), `description` (1–100 chars), and at least one of `packages` (registryType/identifier/version/transport, with `environmentVariables` marked `isSecret`/`isRequired`) or `remotes` (`type: streamable-http|sse`, public `url`). Optional: `title`, `repository`, `websiteUrl`, `icons`, `_meta`. Publishing requires namespace authentication (`io.github.<user>/*`) and package-ownership proof; only trusted public registries are allowed (npm, PyPI, NuGet, crates.io, Docker Hub, GHCR, Quay, Google Artifact Registry, ACR, MCR, and MCPB via GitHub/GitLab releases). Only `io.modelcontextprotocol.registry/publisher-provided` survives in `_meta`, capped at 4 KB. **The registry is not designed for self-hosting** — which is consistent with Agent Shaker being a server rather than a registry, but means "publish Agent Shaker's MCP server to the registry" is a distribution decision, not a self-hosting one.

---

### B2. A2A — current status and relationship to MCP

- **Spec version v1.0.0.** The official `llms.txt` describes the protocol specification as "(v1.0.0)"; the repo's latest release is **v1.0.1** (2026-05-28). An aggregator post claiming "A2A v1.2" is not corroborated by the official docs (see Section 5).
- **Normative artifact is protobuf** (`specification/a2a.proto`); the JSON Schema is non-normative and generated at build time. This is a meaningful change for a Go implementation: a Go implementation should target the proto semantics, not a hand-written JSON shape.
- **Bindings:** JSON-RPC 2.0 over HTTP with SSE, gRPC, and HTTP/REST.
- **Core objects:** Agent Card (discovery), Task (stateful unit of work), Message, Part, Artifact.
- **RPC surface:** `SendMessage`, `SendStreamingMessage`, `GetTask`, `ListTasks`, `CancelTask`, `SubscribeToTask`, `GetExtendedAgentCard`, and the four push-notification-config methods.
- **Streaming events:** `task`, `message`, `status_update` (taskId, contextId, status), `artifact_update` (taskId, artifact, append, last_chunk).
- **Security:** HTTPS/TLS required in production; authentication declared in the Agent Card via OpenAPI-style security schemes; push-notification webhooks secured by per-task authentication tokens.
- **Discovery:** `/.well-known/agent-card.json`. `GetExtendedAgentCard` returns richer metadata *after* authentication — the two-tier card model is the current answer to "how do you publish capability without publishing secrets".
- **Signed agent cards** are part of the v1.0 story.
- **Enterprise features** are a first-class doc section (authentication, authorization, transport security, tracing, monitoring), as are **multi-tenancy** and **extensions** with a tiered promotion process.
- **SDKs:** Python (`a2a-sdk`), JS/TS (`@a2a-js/sdk` 1.3.0), Java, Go, .NET (`A2A`), Rust (`a2a-rs`).
- **CLI:** `a2a-cli` provides client interfaces to connect with A2A agents and ships an **Agent Skill and Agent Plugin** — a directly reusable precedent for how a coordination plane can be distributed as a host plugin rather than as a server.

**Relationship to MCP, as officially stated:** "A2A covers agent-to-agent communication; MCP covers agent-to-tool communication. They are complementary." The practical 2026 reading is a three-layer stack — MCP (agent↔tool), A2A (agent↔agent), AG-UI (agent↔user-facing app) — with ACP (agent↔editor) and AGNTCY/registry-style infrastructure underneath.

**For Agent Shaker specifically:** A2A `Task` is a *stateful unit of work* with a lifecycle, and Agent Shaker already has a task lifecycle with atomic claim. The two models are near-isomorphic, which is why `A2A_SCHEMA_UPDATE_v1.0.md` exists in the repo already. The v1.0 `contextId`/`taskId` pair, `status_update` events and `artifact_update` (with `append` and `last_chunk`) map cleanly onto Agent Shaker's task/dependency/artifact model. The gap worth naming: A2A v1.0 has no first-class *lease* concept, and no concept of a **shared, project-scoped context store** — which is exactly Agent Shaker's differentiator.

---

### B3. Interop approaches worth adopting in 2026

#### B3.1 Agent identity and attestation — three live IETF drafts

| Draft | What it specifies | URL |
| --- | --- | --- |
| `draft-klrc-aiagent-auth-01` | WIMSE identifier as the **primary identifier for agents**; attestation-based credential issuance (TEE evidence, software integrity measurements, supply-chain provenance, platform/orchestrator attestation); short-lived credentials with explicit expiry; cryptographic proof-of-possession (Workload Identity Tokens + HTTP Message Signatures / WPTs) | https://datatracker.ietf.org/doc/html/draft-klrc-aiagent-auth-01 |
| `draft-pidlisnyi-aps-03` | **Agent Passport System**: Ed25519 self-signed passports binding key ↔ agent id/name/owner/expiry; delegation chains that **cannot widen** across scope, spend, depth, time, reputation, values or reversibility; **signed action receipts**; four provenance tiers (observed / infrastructure-attested / provider-attested / self-declared) | https://datatracker.ietf.org/doc/html/draft-pidlisnyi-aps-03 |
| `draft-sweeney-wimse-credential-delegation-01` | DPoP-bound Delegation Tokens; agents authenticate to a delegation service with a `did:key` JWT bearer assertion; DPoP binding to `htm`/`htu`; revocation SLA on agent termination | https://datatracker.ietf.org/doc/draft-sweeney-wimse-credential-delegation/ |

Two observations. First, the APS draft explicitly notes that the **AIAGENT-AUTH best-practices framework "defines no new protocols"** — the direction of travel is OAuth 2.0 + WIMSE, not a new identity stack. Second, the signed-action-receipt concept in APS maps directly onto Agent Shaker's activity/event model: a coordination plane is exactly the place where "an agent exercised authority X over resource Y" can be signed and later replayed.

There is also a reference implementation ecosystem (`agent-passport-system/agent-passport-system`: passport grades 0–3, `did:aps` identifiers, `did:key`/`did:web`/SPIFFE SVID/OAuth interop) and an academic proposal (**Open Agent Passport**, arXiv 2603.20953) that frames the same problem as "pre-action authorization": intercept tool calls synchronously *before* execution, evaluate against declarative policy, and emit a cryptographically signed audit record.

#### B3.2 What vendors are actually shipping for identity

- **Microsoft Entra Agent ID** — agents become first-class directory identities with Conditional Access, RBAC/ABAC, access reviews, lifecycle policies and audit; Identity Governance can scope an agent's access to a subset of a user's permissions (GA 2026-05-01).
- **Claude Managed Agents** — scoped permissions, identity management and execution tracing built in; permission policies with an `auto` evaluation mode.
- **MCP** — CIMD replacing DCR, `iss` validation, issuer-keyed credential storage.
- **Google** — Agent Platform governance plane as part of the 2026-04-22 rebrand.

#### B3.3 Session handoff and context/artifact passing

- A2A `Task` + `contextId` + `artifact_update` (with `append`/`last_chunk`) is the standard for agent→agent artifact transfer.
- MCP `2026-07-28` adds **server-minted state handles** as ordinary tool arguments — the standard for agent→server continuation.
- The MCP **Skills extension** is the standard for server→agent instruction packages with digest verification.
- The MCP **Tasks extension** is the standard for server-side long-running work with client→server input.

The open gap, which is Agent Shaker's opportunity: **there is no standard for "the shared, project-scoped, cross-agent truth store"**. A2A has no shared context; MCP has no cross-server shared state. Agent Shaker's markdown contexts + full-text search + milestones occupy exactly that gap, and the 2026 protocol stack does not fill it.

---

### B4. Hooks and gates as a pattern in agent hosts

| Host | Mechanism | Can block? | Gate granularity | Plugin/distribution model |
| --- | --- | --- | --- | --- |
| **OpenCode V2** | `ctx.session.hook(name, cb)`; `ctx.permission.hook("evaluate")`; `ctx.shell.hook("create.before")` | Yes — permission effect `allow`/`ask`/`deny`; shell rewrite | Per action/resource; shell command rewrite | npm/git packages, `opencode.json(c)`, CLI `opencode plugin`; no official marketplace |
| **Claude Code** | 33 events; `command` / `http` / `mcp_tool` / SDK-callback hooks | Yes — exit 2, `decision:"block"`, `onFailure:"block"` | Per tool call, per permission request, per task create, per compaction | `.claude-plugin/plugin.json` + marketplaces (`marketplace.json`); Anthropic official marketplace auto-added |
| **Codex CLI** | `hooks.json` / `[[hooks.*]]` in `config.toml` | Yes — `PreToolUse` intercepts Bash, `apply_patch`, MCP tools | Per tool, matcher-based | Plugin manifest `.codex-plugin/plugin.json`, `hooks/hooks.json`, `PLUGIN_ROOT`/`PLUGIN_DATA` |
| **OpenAI Agents SDK** | Not hooks — guardrails + approvals | Yes — approvals *pause* the run | Per tool call, per workflow step | Library-level |
| **Claude Managed Agents** | Server-side permission policies (`auto` mode) | Yes — server runs / denies / pauses for approval | Per agent or MCP tool call, evaluated server-side | Hosted API |

The pattern that generalises: **every serious host in 2026 exposes a pre-execution decision point, and each one distinguishes "observe" from "decide"**. Claude Code encodes this in exit codes and JSON fields; OpenCode encodes it in the permission hook's `effect`; Codex encodes it in `PreToolUse`; the managed-agent platforms encode it server-side.

Three design consequences for Agent Shaker's Phase 3:

1. **A hook that must fail closed must say so.** Claude Code's `onFailure: "block"` and the note that a timed-out `PreToolUse` hook does *not* block, are the concrete form of this. A lease check implemented as a hook that silently times out is a correctness bug, not a resilience win.
2. **`mcp_tool`-style hooks are the right shape for a coordination plane.** Codex's `mcp_tool` handler invokes a tool on an already-connected MCP server with structured arguments — no shell, no quoting, no trust prompt. Agent Shaker's MCP tools (`claim_task`, `complete_task`) are already exactly this API; the integration should call them as MCP tools, not as shell scripts.
3. **Hosts differ in *when* they trust a hook.** Codex hashes hook definitions and requires persisted re-approval when they change; Claude Code's marketplaces add a "Context cost" estimate and a security tiering (official/community/third-party). A shipped integration will be evaluated on its trust posture, not only its function.

---

### B5. Compaction and context-window management as an integration surface

| Host | Compaction model | Integration surface |
| --- | --- | --- |
| **OpenCode V2** | Checkpoint-based; `compaction.auto` (default true), `compaction.keep.tokens` (default 15,000), `compaction.buffer` (default 15 % of limit, min 16k); manual request; `experimental.background_compaction`; summary covers blockers + next moves + relevant files | **`ctx.session.hook("compaction", cb)` — the plugin may supply `result.summary`, `result.metadata`, `result.tokens`**; plus `ctx.session.hook("context", cb)` to observe outgoing context |
| **Claude Code** | Automatic compaction with `PreCompact` / **`PostCompact`** hooks | `PreCompact`/`PostCompact` hooks; `SessionStart` matcher includes `compact` (a compaction-triggered session start) |
| **Codex CLI** | `hooks.SessionStart` with matcher `^compact$` | Compact-scoped `SessionStart` hook |
| **OpenAI Agents API** | Automatic compaction of earlier context as a session approaches its limit, preserving what the agent needs to continue | Automatic; agents span multiple context windows without implementing compaction logic |
| **Claude Managed Agents** | Harness owns context management | Automatic; MCP tool outputs >100 K chars are spilled to a file in the sandbox (2026-05-19) |
| **MCP** | Not a host — but `resources/read` + Skills extension + `skills/get` are the standards for *serving* context | Server-side `resources`, `skills`, `tasks` |

How external context stores are integrated today, in practice:

1. **Via MCP resources/skills**, not via a host-specific API. A memory server exposes `resources`; a workflow server exposes `skills`. The Skills extension's rules are the strictest in the ecosystem and worth reading as a spec: digest verification, size limits (512 files / 16 MiB), approval bound to the manifest, no retrieval ahead of need.
2. **Via compaction hooks that inject or capture summary content** — the OpenCode `compaction` hook and the Claude Code `PreCompact`/`PostCompact` pair. This is the only place where a host lets an external system influence what survives a compaction.
3. **Via summary-authoring plugins** — OpenCode V2's `compaction` hook can *produce* the summary. No other host exposes an equivalent write path.
4. **Via file spilling** for oversized tool output — Claude Managed Agents writes >100 K-char tool results to a file in the sandbox.

**The pattern:** compaction is now a *negotiated* boundary, not a private implementation detail. OpenCode has a `summary` contract and a `metadata`/`tokens` slot; Claude Managed Agents treats a session as "the append-only log of everything that happened" and treats the harness (which owns context management) as a separately swappable component. A coordination plane that wants to own cross-agent context continuity should target exactly this boundary.

---

### B6. Observability for agents

#### B6.1 OTel GenAI semantic conventions — moved, and still unstable

The `opentelemetry.io/docs/specs/semconv/gen-ai/` pages are now **"Moved" stubs**: "GenAI semantic conventions have moved to the OpenTelemetry GenAI semantic conventions repository. This page has moved and is no longer maintained in this repository."

Timeline as reported by the semconv work and the OTel repo:

| Release | Change | Consequence |
| --- | --- | --- |
| v1.40.0 (Feb 2026) | Retrieval spans, cache-token attributes, `gen_ai.agent.version` | Richer agent/RAG telemetry |
| v1.41.0 (Apr 2026) | `invoke_agent` split into client and internal spans; reasoning-token fields; streaming-latency metrics; stricter `execute_tool` naming | In-process agent frameworks get their own span shape |
| **v1.42.0 (2026-06-12)** | **All `gen_ai.*` content deprecated in core and moved to `open-telemetry/semantic-conventions-genai`** | The authoritative source changed without a versioned release at the new home |

Current state of the new home: **no tagged release**, README schema URL still a TODO, ships from `main` against schema `gen-ai/1.42.0`. Every GenAI span, event, metric and attribute is **Development** status — nothing GenAI-specific is Stable. The only Stable attributes referenced from GenAI tables are core ones (`error.type`, `server.address`, `server.port`).

Agent-span conventions that exist: `create_agent`, `invoke_agent` (client + internal), `invoke_workflow`, `plan`, `execute_tool`, `execute_tool` for agent skills, `load_skill`, `read_skill_resource`, command execution.

Practical rule: **do not pin a "GenAI semconv version"** — there isn't one. Pin the instrumentation library and record which schema it targets, keep raw attribute strings behind a thin mapping layer, and budget for a dashboard revision when names shift.

#### B6.2 Durable-execution telemetry is being designed right now

Issue #462 in the GenAI repo ("Proposal: observability semantics for durable agentic runtime execution", opened 2026-08-19, active through 2026-10-22) proposes, and partially implements:

- `gen_ai.agent.execution.id` — a **logical run id distinct from the trace id**, surviving suspend/resume and process boundaries.
- `gen_ai.agent.paused` / `checkpointed` / `resumed` events.
- `resumed_from.type` / `resumed_from.id` — a flat pair linking a resumed segment to the boundary it continues from.
- Span links for resume, delegation, retries, fan-out, aggregation and compensation.
- A semantic note (PR #588) clarifying that `execute_tool` describes execution *as observed at the instrumentation boundary*, and that a successful tool result does **not** prove a durable external effect.

That last clause is the one that matters most for a coordination plane: it is a normative statement that "the agent called the tool and it returned OK" ≠ "the work was committed". A coordination plane whose value proposition is durable shared truth is exactly the layer that can supply the missing evidence.

#### B6.3 Vendor-side observability

- **Anthropic** — OpenTelemetry export in beta as of **2026-10-14**, covering Messages API requests *and Claude Managed Agents sessions*. `agent.tool_use` / `agent.mcp_tool_use` events carry an `evaluation` field alongside `evaluated_permission` (2026-09-10).
- **MCP** — SEP-414 documents trace-context propagation conventions for `_meta` keys `traceparent`, `tracestate`, `baggage`; this is in the `2026-07-28` revision.
- **Codex** — exports metrics and log events for API requests, tool calls and sessions; Claude Code exports metrics and logs with trace support in beta.
- **Paperclip** — internal trace collection, run history, opt-in tracing, telemetry data contract, and a "Connectivity" family of records.

#### B6.4 Eval-driven quality

- Paperclip treats evals as a *pillar*: Skill Studio, shared org-wide skills, evals & saved test runs, active learning loops and quality metrics, performance reviews for agents, skill version history and restore. Its `evals/` directory is runnable with promptfoo.
- Anthropic positions Managed Agents self-evaluation ("define outcomes and success criteria, Claude self-evaluates and iterates") as a research preview.
- OpenAI's Agents SDK docs push a traces-first-then-eval-loops workflow.
- Google ADK's own docs in the Agent Platform launch post: "AI-driven development: a programmatic interface for coding agents to access Google's entire suite of agentic capabilities, allowing them to build, evaluate, and deploy production-ready agents on your behalf."

The consistent 2026 shape: **evals and saved test runs are part of the coordination product, not a separate QA artifact.**

---

## Section 3 — "Newest approaches": a 2026-phase integration should adopt these

Ranked by (a) how load-bearing it is for a coordination-plane product and (b) how recent it is. Items 15–17 are the Console-governance additions; the note in each row gives the rank they would occupy if OpenCode Console is in scope.

| # | Technique / pattern | Source | Why it matters for Agent Shaker |
| --- | --- | --- | --- |
| 1 | **Plugin-as-summary-author**: implement the host's compaction hook so the *external coordination store* produces the compaction summary, rather than only observing it. | https://opencode.ai/v2/docs/build/plugins (`compaction` hook → `SessionCompactionResult.summary`) | OpenCode V2 lets a plugin own `summary`, `metadata` and `tokens`. A shared-truth store that can write the summary is categorically different from one that only reads it — and it is the single strongest differentiator available in Phase 3. |
| 2 | **Checkpoint-based, explicitly-parameterised compaction** (`keep.tokens` + `buffer` + `auto`) rather than an opaque "context full" event. | https://opencode.ai/v2/docs/compaction | Turns compaction from a threshold into a *contract*: what is kept verbatim, what is summarised, when it fires. A coordination plane can size its injection against `keep.tokens` and reason about cost deterministically. |
| 3 | **Hook domains with a named, enumerated event surface** instead of a single generic callback. | https://opencode.ai/v2/docs/build/plugins (`SessionHooks`) | A coordination plane can bind precisely — heartbeat on session lifecycle, state capture on `compaction`, enforcement on `permission` — instead of pattern-matching payloads. |
| 4 | **Pre-execution permission gate exposed as a hook**, where the integrator may flip `allow`/`ask`/`deny`. | https://opencode.ai/v2/docs/build/plugins (`ctx.permission.hook("evaluate")`) | Turns a coordination plane from a *reporting* system into an *enforcing* one: lease expiry or dependency violation can become a denial, not a log line. |
| 5 | **Stateless MCP with server-minted state handles passed as ordinary tool arguments.** | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md (SEP-2567, SEP-2575) | The claim/lease model must survive the removal of `Mcp-Session-Id`. Handles-as-arguments is the sanctioned pattern — and it also makes claims replayable and auditable. |
| 6 | **Idempotency as a protocol requirement**: broken streams lose in-flight requests and clients MUST re-issue with a new request id; MRTR retries the *original* request. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | Retries and re-issues are now normal traffic. A coordination plane whose tools mutate shared state must be idempotent by construction, and should accept a client-supplied idempotency key. |
| 7 | **`server/discover` capability negotiation as the entry handshake.** | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md | Lets a server advertise `2026-07-28` + extension support (tasks, skills, apps) so a client can branch before issuing anything — one place to detect what a given client can do. |
| 8 | **MRTR `input_required` as the approval primitive**, replacing server-initiated elicitation. | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md (SEP-2322) | Approval gates become ordinary results the client already handles, and `requestState` is the place to correlate a human decision across retries. |
| 9 | **Deterministic tool ordering + `ttlMs`/`cacheScope` on list results.** | https://modelcontextprotocol.io/specification/2026-07-28/changelog.md (SEP-2549) | With 28+ tools, stable ordering is a measurable prompt-token saving, and cache-control fields let a coordination plane publish hot lists (open tasks) cheaply. |
| 10 | **Skills-over-MCP for shipping coordination playbooks.** | https://modelcontextprotocol.io/extensions/skills/overview.md (SEP-2640) | The standard way to deliver "how this project works" to any MCP-capable agent, with digest-verified manifests and approval bound to the file set. A distribution channel that requires no host-specific plugin. |
| 11 | **`mcp_tool`-style hooks (call a tool on an already-connected MCP server) instead of shell-out hooks.** | https://www.codex-docs.com/en/docs/hooks | Removes shell quoting, PATH and trust-prompt problems from the integration, and reuses the coordination plane's own typed API. |
| 12 | **Signed agent identity + signed action receipts (APS-style), layered on OAuth 2.0 + WIMSE rather than a new identity stack.** | https://datatracker.ietf.org/doc/html/draft-pidlisnyi-aps-03 · https://datatracker.ietf.org/doc/html/draft-klrc-aiagent-auth-01 | The activity log is the natural place to sign "agent X exercised authority Y over resource Z", and delegation chains that cannot widen are exactly the semantics a coordination plane's task-scoping needs. |
| 13 | **MCP Apps (SEP-1865) for the coordination UI shipped as an MCP server resource.** | https://modelcontextprotocol.io/seps.md · https://modelcontextprotocol.io/extensions/apps/overview.md | Lets the Agent Shaker UI reach into hosts (Claude, ChatGPT) instead of requiring the user to open a separate browser tab — the same "bring the plane to the agent" move as MCP-native UI. |
| 14 | **Durable-execution telemetry: logical execution id + paused/checkpointed/resumed events + resume links.** | https://github.com/open-telemetry/semantic-conventions-genai/issues/462 | The conventions for observing exactly the thing a coordination plane owns (multi-session, resumable, delegated work) are being written *now* — a chance to be an early implementer of the semantics, not a late adopter. |
| 15 | **Server-delivered policy with final authority over local configuration** (statements appended last, applied after agent rules and saved approvals, delivered over ~1 min, server can *lift* a local deny). | https://opencode.ai/v2/docs/policies/ | The 2026 control-plane pattern: the governance decision comes from the server and outranks local state. Entering at rank ~3 if Agent Shaker targets team deployments — it is the mechanism by which a coordination plane becomes authoritative rather than advisory. |
| 16 | **Budgets as policy rules with a 429 on breach, checked at request start, with the blocking policy recoverable from activity logs.** | https://opencode.ai/v2/docs/console/budgets/ | Cost control is now a *policy* concern with an enforcement point and an audit trail, not a reporting feature. Enters at rank ~4 — and it is the concrete shape a spend-tracking feature in a coordination plane should take (levels, precedence, non-retroactive reset, in-flight overshoot). |
| 17 | **Named external-context references with machine-readable guidance injected into agent instructions, and a warm/idle cadence owned by the host.** | https://opencode.ai/v2/docs/references/ · https://opencode.ai/v2/docs/warming/ | Validates the "named context + description" shape for shared truth, and means Agent Shaker's context model must beat it on *multi-agent, server-side, searchable, identity-scoped* — and that any lease TTL must be chosen against the host's own 4-minute/30-minute idle cadence. Enters at rank ~6. |

---

## Section 4 — Table stakes for the coordination-plane category (late 2026)

Derived from the competitor scan. Each row states whether Agent Shaker currently has it, based on the repo's own README and docs.

| # | Capability | Evidence that it is table stakes | Agent Shaker status (2026-10-24) |
| --- | --- | --- | --- |
| 1 | **Atomic task claim with explicit conflict semantics** | Paperclip (task manager + heartbeat), MCP stateless handles | **Have** — `claim_task` in the scoped loop |
| 2 | **Durable activity / event log of mutating actions** | Paperclip "Activity & Events"; Claude Managed Agents "session = append-only log"; OTel issue #462 | **Partial** — events exist; no append-only guarantee, no signing, no OTel export |
| 3 | **Approval / human-in-the-loop gates** | Paperclip approval stages + tool-gateway approvals; OpenAI Agents SDK interruptions; Claude Managed Agents `auto` permission mode; MRTR `input_required` | **Missing** — milestones are "PM-gated" but there is no general approval primitive |
| 4 | **Cost / token / spend tracking with thresholds** | Paperclip budgets by company/agent/project/goal/issue/provider/model with hard stops; **OpenCode Console budgets as policy rules with HTTP 429 on breach**; Claude Managed Agents per-session-hour pricing; OTel `gen_ai.usage.*` | **Missing** — and now the most exposed gap, because the *host* (OpenCode Console) ships this before the coordination plane does |
| 5 | **Heartbeat with liveness semantics** | Paperclip DB-backed wakeup queue with coalescing and bounded recovery; Paperclip "if it can receive a heartbeat, it's hired" | **Have (lease)** — but no coalescing, no bounded-recovery model, no wakeup-side trigger |
| 6 | **Cross-agent shared context with retrieval** | Paperclip shared org-wide skills; MCP Skills extension; OTel `load_skill` / `read_skill_resource` | **Have** — shared markdown contexts + FTS. Strongest existing asset |
| 7 | **Dependency / handoff between agents** | A2A `Task` lifecycle; Paperclip delegation up and down the org chart | **Have** — dependencies/handoffs |
| 8 | **A2A interop** | A2A v1.0.0 GA; signed agent cards; six official SDKs; a2a-cli with Agent Skill + Agent Plugin | **Have** — but verify against the v1.0.0 normative proto (see Section 5) |
| 9 | **MCP interop at the current revision** | MCP `2026-07-28`; OpenCode already selectable to `2026-07-28`; `@modelcontextprotocol/sdk` 1.32.2 | **Partial / verify** — confirm the server is stateless-compatible and implements `server/discover` |
| 10 | **Per-agent identity and scoped credentials** | Entra Agent ID; APS passports; WIMSE; MCP per-issuer credential keying | **Have** — per-agent API keys (Phase 2). **Gap**: keys are static, not short-lived/attested |
| 11 | **Agent identity in the protocol surface** | A2A Agent Card + `GetExtendedAgentCard`; APS `did:aps` | **Partial** — publishes an agent card; no signed card, no extended card |
| 12 | **Streaming / live updates** | A2A `SendStreamingMessage` + `status_update` + `artifact_update`; Paperclip real-time streams | **Have** — WebSocket feed; A2A streaming + artifacts |
| 13 | **Full-text search over shared state** | Paperclip internal trace collection; MCP Resources + `skills/get`; OTel retrieval spans | **Have** — FTS on markdown contexts |
| 14 | **Self-hostable deployment** | Paperclip "Open source. Self-hosted."; Claude Managed Agents self-hosted sandboxes + MCP tunnels; Agent 365 Windows 365 for Agents | **Have** — Docker Compose, Caddy/TLS edge |
| 15 | **Exportability / no lock-in of the shared truth** | Paperclip company export/import packages; Anthropic memory stores that sync into/out of sandboxes | **Partial** — Postgres is portable, but no documented export/import contract |
| 16 | **Evals and saved test runs as a first-class feature** | Paperclip pillar "Agent Employee Training"; OpenAI traces→eval loop; Google ADK eval APIs | **Missing** |
| 17 | **Auditability with an explicit trust posture** | Codex persisted hook trust by hash; Claude Code marketplace security tiers + "Context cost"; Agent 365 audit + compliance retention | **Missing** — no plugin trust story, no context-cost accounting for a plugin that adds tools |
| 18 | **Observability export** | OTel export beta in Anthropic (2026-10-14); Paperclip internal trace collection; MCP SEP-414 traceparent in `_meta` | **Missing** |
| 19 | **Multi-tenancy / org isolation** | A2A multi-tenancy topic; Paperclip multi-organization (separate tasks, agents, permissions, activity histories) | **Partial** — projects exist; no per-org isolation of permissions/activity |
| 20 | **Scheduled / recurring coordination** | Paperclip scheduled (recurring) jobs in the plugin SDK; Claude Managed Agents scheduled deployments (2026-06-09) | **Partial** — daily standups exist; no general scheduler |
| 21 | **Tool-level plugin/context-cost transparency** | Claude Code marketplace shows a "Context cost" estimate per plugin | **Missing** — a Phase 3 OpenCode/Claude plugin that injects context must state its cost |
| 22 | **Reversible/auditable configuration changes** | Paperclip "Governance with rollback": approval gates enforced, config changes revisioned, rollback | **Partial** |
| 23 | **Server-authoritative policy that outranks local client config** | OpenCode Console policies: "final authority over the member's own configuration", applied after agent rules and saved approvals, ~1 min delivery | **Missing** — Agent Shaker's config is currently client-side by definition; a server policy layer is the missing half of "shared truth" |
| 24 | **A named-context primitive with machine-readable guidance** | OpenCode References (`description` injected into agent instructions); MCP Skills extension (`SKILL.md` + digest manifest) | **Partial** — Agent Shaker has shared markdown contexts but no "description" field that tells an agent *when* the context is relevant |
| 25 | **Host-aware liveness/idle semantics** | OpenCode warming: 4-minute idle interval, 30-minute window; Paperclip heartbeat coalescing and bounded recovery | **Partial** — Agent Shaker has a lease but no documented relationship to host idle cadence |

**Top three gaps that are cheap to close and currently visible in the market:** (2) append-only activity log with OTel export, (3) an approval primitive, (4) cost/spend tracking. All three are present in at least two independent competitor categories, and all three are prerequisites for Agent Shaker being taken seriously as a *control plane* rather than a shared notes server.

**The new pressure point:** OpenCode Console now ships budgets and server-authoritative policies *inside the host*. A coordination plane that offers neither spend control nor server-side policy authority is not merely "behind" on features — it is offering less control than the agent host the user already has open. The cheapest counter-position is not to rebuild a Console, but to be the **vendor-neutral** place where budget state and policy decisions live across hosts that each have their own (OpenCode Console, Claude Managed Agents, Paperclip, Agent 365) — and to make the enforcement point open (MCP `input_required`, A2A task status, a denied tool call) rather than proprietary.

---

## Section 5 — Claims I could NOT verify, and what to check manually

These are flagged rather than quietly asserted. Nothing in Sections 1–4 depends on an unverified claim; each is either explicitly labelled in-line or excluded.

1. **"A2A v1.2".** An aggregator post stated A2A is at "v1.2". The official `a2a-protocol.org/llms.txt` describes the specification as **v1.0.0**, and `a2aproject/A2A`'s latest GitHub release is **v1.0.1 (2026-05-28)**. *Check:* `https://github.com/a2aproject/A2A/releases` and the version banner on `https://a2a-protocol.org/latest/specification/`. Possibly the aggregator counted a docs/CLI version separately.
2. **A2A v1.0 GA date.** A Linux Foundation press release referenced in search results announced v1.0 GA around March/April 2026, and a separate source said "April 9, 2026 one-year announcement". The official release tag date (2026-05-28) is verified; the GA announcement date is not. *Check:* the A2A blog (`https://a2a-protocol.org/latest/blog/`) and the LF press release.
3. **Exact MCP Apps (SEP-1865) mechanics.** I confirmed the SEP is Final and that it exists, and I read the Apps extension overview — but I did **not** verify the precise resource-URI scheme, the host-side rendering contract, or current client support. *Check:* `https://modelcontextprotocol.io/extensions/apps/overview.md` end-to-end and the client support matrix.
4. **MCP client-support matrix contents.** The matrix at `https://modelcontextprotocol.io/extensions/client-matrix.md` was referenced by the Skills overview but not read. Coverage of Tasks/Skills/Apps across real hosts is the number that decides whether an Agent Shaker feature can rely on an extension. *Check:* read the matrix and the `ext-skills` implementations list.
5. **OpenCode "popularity indicators" beyond stars.** I verified stars/forks/npm version counts. I did **not** verify npm weekly/monthly download counts for `opencode-ai`, `@opencode/plugin` or `@opencode-ai/plugin`, nor the `opencode.ai` monthly active users. *Check:* `https://api.npmjs.org/downloads/point/last-month/<pkg>` and the OpenCode announcements page.
6. **OpenCode V1 plugin hook names.** The V1 names (`tool.execute.before`, `event`, `chat.message`, `chat.params`, `permission.ask`) come from search-result excerpts and the V2 migration guide, not from a line-by-line read of the V1 plugin reference. The V2 surface was read directly from the V2 reference and is authoritative. *Check:* `https://opencode.ai/docs/plugins/` (V1) for the exact V1 hook names if a V1-targeted artefact is still planned.
7. **The OpenCode V2 "ecosystem marketplace" page.** `opencode.ai/docs/plugins/` says "Browse available plugins in the ecosystem" — I did **not** open that destination. Whether OpenCode ships an official curated listing (as opposed to a community marketplace) is unresolved. *Check:* the link target on the V1 plugins page and `https://opencode.ai/` for a marketplace/console entry.
8. **Claude Code headless mode specifics.** `docs.claude.com/en/docs/claude-code/headless.md` was fetched (37 KB) but not analysed; the specific flags, `--output-format` values and streaming-JSON schema are not verified here. *Check:* that page directly if a Phase 3 step needs headless driving.
9. **Microsoft Agent 365 → Claude Code / Copilot CLI discovery.** The GA blog says Defender and Intune "will be able to discover and manage local AI agents running on Windows devices, starting with OpenClaw agents and expanding soon to other widely used agents like GitHub Copilot CLI and Claude Code" — future tense. *Check:* Microsoft Defender/Intune release notes for when the local-agent inventory actually ships, because it determines whether a local-only coordination plane can be discovered by enterprise policy.
10. **Agent 365 registry sync with AWS Bedrock / Google Cloud.** Seen in a search excerpt, not re-read in a primary page. *Check:* the Agent 365 GA blog and `https://learn.microsoft.com/en-us/microsoft-agent-365/overview` for the exact integration list.
11. **Anthropic "Claude Models on Google Cloud" specifics.** The GA post is dated 2026-10-23 and the platform includes Claude Managed Agents (beta) and skills under the Anthropic-managed option; the precise feature parity between Anthropic-managed and Google-managed was not verified exhaustively. *Check:* `https://cloud.google.com/blog/products/ai-machine-learning/expanding-claude-availability-on-google-cloud` and the Claude Platform release notes.
12. **OpenAI Agents API `required_actions` / `status: "requires_action"` semantics.** The API reference documents the shape and the statuses (`idle`, `in_progress`, `requires_action`, `failed`), but the end-to-end approval handshake was not traced from a worked example. *Check:* `https://developers.openai.com/api/reference/resources/beta/subresources/agents/subresources/sessions/methods/create` and the Agents API observability guide.
13. **OpenAI Agents API data residency.** The FAQ states US-only residency and no Zero Data Retention "including when you use a self-hosted sandbox" — this is unusually strict and materially affects an EU deployment. It is sourced from the OpenAI Help Center but is a policy statement, not an API contract. *Check:* the Agents API data-controls documentation before committing to a deployment region.
14. **Competitor scope details for Multica, Vibe Kanban, Superset, AgentsRoom.** Only GitHub metadata was verified (stars, licences, push dates). Their feature sets, whether they self-host, and whether they have append-only logs / approvals / cost tracking are **not** verified. *Check:* each project README and release notes before making positioning claims.
15. **Vibe Kanban's stated direction.** One search result described a "sunsetting → community-maintained" narrative. The repo shows last push 2026-09-19 and last release 2026-04-24, which is consistent with reduced activity but does **not** confirm any announced sunset. *Check:* the repo's README, `ROADMAP`, and any `maintenance` notice.
16. **Roo Code's shutdown details.** Archive state and final release v3.54.0 (2026-05-15) are verified from the GitHub API; the *rationale* and the "Roomote" pivot come from secondary sources including an AI-wiki page. *Check:* the archived repo's README and the team's own announcement.
17. **Continue.dev / Cursor acquisition.** Reported consistently across secondary sources ("Continue was acquired by Cursor", last non-prerelease tag 2026-06-19) but not verified against a primary announcement. *Check:* continue.dev and the Cursor changelog.
18. **OTel GenAI semconv exact current attribute set.** I confirmed the move out of core (v1.42.0, 2026-06-12), the Development status, the absence of a tagged release, and the existence of the agent-span operations. I did **not** enumerate the current attribute names, and several are in active PRs (e.g. the `handle_turn` consolidation in PR #571, `gen_ai.agent.execution.id` in PR #445). *Check:* pin a specific commit of `open-telemetry/semantic-conventions-genai` before writing any attribute names into code.
19. **OpenTelemetry `handle_turn` vs `handle_conversation_turn` naming.** PR #571 is still open and the discussion explicitly renames the operation mid-flight. *Check:* the merged PR before relying on either name.
20. **Zed's Agent Client Protocol version naming.** I verified the repo and schema tag `schema-v1.26.0` (2026-10-16) and that a secondary matrix calls ACP "Protocol v1 (schema 1.10.2); v2 in alpha" — the two do not agree. *Check:* `https://github.com/agentclientprotocol/agent-client-protocol` releases and the schema directory.
21. **AG-UI adoption list.** "First-party in Microsoft Agent Framework, Google ADK, AWS Strands, Bedrock AgentCore" comes from a secondary report; the AG-UI docs themselves describe the handshakes but I did not verify each named integration. *Check:* each framework's docs for an AG-UI adapter.
22. **AG-UI download volume.** Numbers ranged from ~12.0 M to ~16.9 M combined monthly downloads across the same month in different secondary reports. *Check:* `https://api.npmjs.org/downloads/point/last-month/@ag-ui/client` (one figure quoted: 6,175,767 for the month ending 2026-10-07).
23. **Paperclip's exact GitHub star count is a point-in-time value.** 106,790 stars on 2026-10-24; the repo is pushed daily. *Check:* the API before quoting.
24. **Paperclip's "local clone" provenance.** The repository at `.competitors/paperclip-master/` in this workspace was used to read Paperclip's README and plugin-SDK docs directly. Its exact upstream commit was **not** verified against `github.com/paperclipai/paperclip`, so any line-level quote should be re-checked against upstream. *Check:* `git -C .competitors/paperclip-master log -1` or re-fetch the raw README.
25. **Agent Shaker's current MCP revision compliance.** The repo's MCP surface predates the `2026-07-28` revision; whether the server already implements `server/discover`, stateless semantics, `resultType`, `Mcp-Method`/`Mcp-Name` and `ttlMs`/`cacheScope` was **not** checked in this scan. *Check:* the server implementation and `docs/MCP_SETUP.md`.
26. **Agent Shaker's current A2A revision compliance.** `docs/A2A_SCHEMA_UPDATE_v1.0.md` exists, but conformance to the v1.0.0 *normative protobuf* was not verified. *Check:* the A2A schema against `https://github.com/a2aproject/A2A/blob/main/specification/a2a.proto`.
27. **OpenCode V2 session-lifecycle events for heartbeat purposes.** Partially resolved, still open. Verified: the V2 hook-domain list contains `prompt`, `context`, `compaction`, `generate`, `title`, `model.request`, `http.request`, `http.response`, `experimental.ws.*` and `retry` — **no explicit `session.started` / `session.idle` domain**. `ctx.event.subscribe({ signal })` exists and yields an async iterable, but the event-type union was not extracted. Separately, OpenCode V2 has its own **warm/idle cadence**: warming starts after 4 minutes without a model request and stops 30 minutes after the last non-warming request. *Check:* the event union in `@opencode/plugin` types or the OpenCode SDK/server event stream, and decide whether Agent Shaker's liveness signal is "session event", "model request observed", or "lease renewed by the agent". This remains the highest-value manual check in this list.
28. **`ctx.event.subscribe` event type names.** The subscription API exists (`for await (const event of ctx.event.subscribe({ signal }))`), but the concrete event names (e.g. `session.idle`, `session.updated`, `message.part`) were not extracted. *Check:* the V2 reference's event section, or the `@opencode/plugin` type definitions for the event union.
29. **Console policy and reference freshness mechanics.** Policy statements are fetched "about once a minute" and references refresh at a ≥24 h eligibility window; neither the fetch trigger nor the exact staleness bound was traced to a primary mechanism (background poll vs push). *Check:* the Console API docs at `https://opencode.ai/v2/docs/console/api/` and the policy plugin's fetch behaviour in the `anomalyco/opencode` source.
30. **Console Harness model coverage.** The harness table verified Claude Code, Codex, Hermes, Pi and Raycast. Whether the list is extensible by third parties — and whether a custom harness can be registered — was not verified. *Check:* `https://opencode.ai/v2/docs/console/harnesses/` and the Console API docs.
31. **OpenCode Console pricing and commercial terms.** OpenCode Go ($10) and Go Plus ($40) monthly subscriptions, API-key tiers and pay-as-you-go rates were read from the harness page; the full pricing schedule, overage rules and data-processing terms were not. *Check:* `https://opencode.ai/v2/docs/console/billing/`.
32. **Whether Agent Shaker should register as an MCP server in the official registry.** The registry is real and `server.json` is specified, but the registry "is not designed for self-hosting" and publishing requires namespace authentication, package-ownership proof and `version` uniqueness per publish. Whether Agent Shaker's MCP server belongs in the registry (versus remaining a self-hosted-only endpoint) is a product decision with an operational cost. *Check:* `https://modelcontextprotocol.io/registry/quickstart` and the `server.json` reference.
33. **DCR vs CIMD divergence.** MCP `2026-07-28` deprecated OAuth 2.0 Dynamic Client Registration in favour of Client ID Metadata Documents, yet the OpenCode Console MCP server "uses OAuth with dynamic client registration", and OpenCode V2 separately ships CIMD support with its own client metadata document at `https://opencode.ai/oauth/opencode/client.json`. Whether the Console MCP server will migrate is unknown. *Check:* the Console MCP server's OAuth metadata document and the `modelcontextprotocol/registry` auth-mode list.
34. **Agent Shaker's MCP surface against the `2026-07-28` revision.** Whether the current server already implements `server/discover`, stateless semantics (no `Mcp-Session-Id`), `resultType`, `Mcp-Method`/`Mcp-Name`, and `ttlMs`/`cacheScope` on list results was not checked in this scan. *Check:* the server implementation and `docs/MCP_SETUP.md` against https://modelcontextprotocol.io/specification/2026-07-28/changelog.md.

---

*Scan performed 2026-10-24. Every URL in this document was fetched on that date unless the row states a publication date. GitHub and npm figures are point-in-time and will drift.*
