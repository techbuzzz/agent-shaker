# @techbuzzz/opencode-agent-shaker

An [OpenCode](https://opencode.ai) plugin that binds an OpenCode session to an
[Agent Shaker](https://github.com/techbuzzz/agent-shaker) coordination server.

Agent Shaker does not run agents. It coordinates them: it holds the project,
the tasks, the shared context and the leases. This plugin turns an OpenCode
session into a live Agent Shaker agent — it holds a lease, survives context
compaction, re-injects the shared project truth into every turn and can block
mutating tools that nobody has claimed.

The package serves **both** OpenCode plugin APIs from one install:

| API | Package | Entry point |
| --- | --- | --- |
| v1 | `@opencode-ai/plugin` | `server(input, options)` |
| v2 | `@opencode/plugin` | `setup(context)` |

Both adapters drive the same coordination core, so the behaviour of a session
does not depend on which host loaded the plugin.

## Install

```bash
npm install @techbuzzz/opencode-agent-shaker
```

## Configure

The endpoint and the API key are never hard-coded. The key is bound to exactly
one agent on the server, so the plugin does not have to know which agent it is.

### OpenCode v2 (`plugins` config)

```jsonc
{
  "plugins": [
    {
      "package": "@techbuzzz/opencode-agent-shaker",
      "options": {
        "endpoint": "http://localhost:8080/mcp",
        "api_key": "shk_...",
        "policy": { "enabled": true }
      }
    }
  ]
}
```

### OpenCode v1 (`plugin` config)

```jsonc
{
  "plugin": [["@techbuzzz/opencode-agent-shaker", {
    "endpoint": "http://localhost:8080/mcp",
    "api_key": "shk_..."
  }]]
}
```

`AGENT_SHAKER_ENDPOINT` and `AGENT_SHAKER_API_KEY` are used when the options do
not carry them. A missing endpoint or key is a load-time error: a coordination
plugin that cannot reach the server cannot hold a lease, and one that loads
anyway looks healthy while the lease quietly expires.

### Options

| Option | Default | Meaning |
| --- | --- | --- |
| `endpoint` | — | Absolute URL of the Agent Shaker MCP endpoint. |
| `api_key` | — | Key issued by the coordination server. |
| `project_id` | resolved from the credential | Project the agent belongs to. |
| `agent_id` | resolved from the credential | Agent id; the server rejects it if it disagrees with the key. |
| `request_timeout_ms` | `10000` | Timeout for one MCP call. |
| `heartbeat.lease_fraction` | `0.6` | Fraction of the granted lease after which the next beat is sent. |
| `heartbeat.min_interval_ms` | `5000` | Lower bound of the beat interval. |
| `heartbeat.max_interval_ms` | `300000` | Upper bound of the beat interval. |
| `policy.enabled` | `false` | Turns the local gate on. Opt-in per project. |
| `policy.fail_closed` | `false` | Blocks mutating tools when the server cannot be reached. |
| `policy.read_only_tools` | `read`, `glob`, `grep`, `list`, `ls`, `search`, `webfetch`, `todoread` | Tools allowed without a claimed task. |
| `context.enabled` | `true` | Re-injects the shared project context. |
| `context.max_chars` | `4000` | Hard character cap of the injected payload. |
| `context.max_items` | `8` | Maximum number of shared contexts considered. |
| `context.query` | — | Full-text query for `search_contexts`; without it `list_contexts` is used. |
| `evidence.enabled` | `true` | Records the outcome of test and build runs. |

## What it does

**Session lifecycle.** `session.created` takes a lease with a `working`
heartbeat, `session.idle` moves the agent to `idle` and keeps the lease alive,
`session.error` moves it to `blocked` and `session.deleted` moves it to
`offline` so the lease is dropped without waiting for the server reaper.

**Heartbeat timing is server-driven.** `send_heartbeat` returns
`lease_ttl_seconds`; the next beat is scheduled from that number, never from a
hard-coded guess. The server default is a 60 second lease.

**Compaction.** On v2 the plugin authors the summary itself: it reads the
agent's task, the open blockers, the dependency edges and the recent shared
contexts from the coordination plane, and sets the compaction result, which
records the compaction and skips the model request. The identifiers of the
coordination records travel with the summary in its metadata. On v1 the host
owns the summary, so the plugin contributes the same digest to the compaction
prompt and stores it with `add_context`. Whatever the plugin authors is also
stored in the coordination plane, so the plane holds the same artifact the
model just saw. The v2 `session.compaction.*` events are observed as well, and
the plugin logs which mode is active.

**Context re-injection.** The shared project truth is pushed into the system
prompt of every model request. The same handler is registered on `context`,
`compaction` and `generate`: registering it on only one of them makes the
compaction request differ from every normal turn and busts the prompt-cache
prefix at the most expensive request of the session's life. The shared context
is fetched once per prompt and reused for the rest of the turn; the payload is
relevance-ordered and hard-capped.

**Policy gate.** When `policy.enabled` is set, a mutating tool is blocked while
the agent has no claimed task. A server timeout fails **open** by default; a
project that cannot accept that sets `policy.fail_closed: true`, because a gate
that fails open by accident is a gate that does not gate.

**Evidence.** The outcome of a test or build run is written into the task's
context, including the error message of a failed run.

## Degradation

At load the plugin probes the host and reports what it finds.

- A missing **session event stream** is a host contract violation: the plugin
  refuses to load. A plugin that silently stops heartbeating is worse than one
  that refuses to start.
- A missing **optional capability** (the session hooks, the tool hooks, the
  permission hook, the shell hook) is a single warning naming the capability
  that was lost, and the matching feature is disabled.

## Assumptions

- One API key is one agent. A host that runs several agents at once needs one
  key per agent, and therefore one plugin instance per agent.
- The v1 API cannot replace the host's compaction summary; the plugin can only
  contribute to it.
- `result.tokens` is left unset in the v2 compaction result: the plugin has no
  honest token count to report, and the field is optional.

## Development

```bash
npm install
npm run typecheck
npm test
npm run build
```

The core is plain TypeScript over `fetch` with no dependency on either host
API, so the tests need neither OpenCode nor a live coordination server.

## Licence

MIT
