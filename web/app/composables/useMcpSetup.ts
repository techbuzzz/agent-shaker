/**
 * MCP client setup for Agent Shaker.
 *
 * This is the bridge between the app and the AI agents it coordinates. An
 * agent connects over MCP to the same origin the browser is already using, so
 * the generated config works unchanged in every supported topology — with or
 * without the TLS edge — because `origin` is read from the running page rather
 * than configured separately.
 *
 * ## The API key is referenced, never embedded
 *
 * The generated config points at `${env:AGENT_SHAKER_API_KEY}` rather than
 * inlining a key. That is deliberate: these files are meant to be committed to
 * the agent's own repository, and a literal key in a tracked file is a leaked
 * key. The operator sets the variable in their shell (or a local, git-ignored
 * `.env`) instead.
 *
 * This also means the browser never needs to hold a key. The SPA cannot obtain
 * one — it is server-only runtimeConfig — and it must not ask the user to paste
 * a live secret into a form, because that form would then be one XSS away from
 * exfiltrating it.
 */

/** Name used for the server entry in every generated file. */
const SERVER_NAME = 'agent-shaker'

/** Environment variable the generated configs expect to find. */
export const API_KEY_ENV_VAR = 'AGENT_SHAKER_API_KEY'

export interface McpSetupInput {
  /** The origin the browser is on, e.g. `https://shaker.example.com`. */
  origin: string
  /** Project to bind the connection to. */
  projectId: string
  /** Project name, used in comments and the generated README. */
  projectName?: string
  /** Agent to bind, when one is selected. Scoping is optional. */
  agentId?: string
  agentName?: string
}

export interface SetupFile {
  /** Path relative to the agent's repository root. */
  path: string
  contents: string
}

/** Tools the server advertises, grouped for the generated instructions file. */
const TOOL_GROUPS: Array<{ heading: string; tools: string[]; note: string }> = [
  {
    heading: 'Orientation',
    note: 'Call these first. They resolve the agent and project from the connection URL.',
    tools: ['get_my_identity', 'get_my_project']
  },
  {
    heading: 'Working on tasks',
    note: 'The core loop. Claim a task, do the work, report back.',
    tools: ['get_my_tasks', 'claim_task', 'complete_task', 'reassign_task', 'update_my_status']
  },
  {
    heading: 'Tasks and planning',
    note: 'Broader project views, for a PM role or a wider look.',
    tools: ['list_tasks', 'create_task', 'update_task_status', 'get_dashboard',
            'list_milestones', 'create_milestone', 'assign_task_to_milestone']
  },
  {
    heading: 'Shared knowledge',
    note: 'Markdown contexts and docs shared between agents. Read before acting, write after deciding.',
    tools: ['list_contexts', 'add_context', 'list_global_contexts', 'read_global_context',
            'publish_global_context']
  },
  {
    heading: 'Other agents',
    note: 'Discover and delegate to external agents over A2A.',
    tools: ['discover_a2a_agent', 'delegate_to_a2a_agent', 'get_a2a_task_status']
  },
  {
    heading: 'Onboarding',
    note: 'Only needed the very first time an agent connects to a new project.',
    tools: ['register_self']
  }
]

/** Strip a trailing slash so joins never produce a double slash. */
function trimOrigin(origin: string): string {
  return origin.replace(/\/+$/, '')
}

/**
 * The MCP endpoint URL, with the project bound in the query string.
 *
 * The Go handler reads `project_id` / `agent_id` from the query string (or the
 * `X-Project-ID` / `X-Agent-ID` headers), which is what makes the scoped tools
 * like `get_my_tasks` work without the agent restating its identity on every
 * call.
 */
export function buildMcpUrl(input: McpSetupInput): string {
  const base = `${trimOrigin(input.origin)}/mcp`
  const params = new URLSearchParams()
  params.set('project_id', input.projectId)
  if (input.agentId) params.set('agent_id', input.agentId)
  return `${base}?${params.toString()}`
}

/**
 * `.mcp.json` — the format read by Visual Studio 2026 and by MCP hosts that
 * follow the same convention.
 */
export function buildMcpJson(input: McpSetupInput): string {
  const config = {
    mcpServers: {
      [SERVER_NAME]: {
        type: 'http',
        url: buildMcpUrl(input),
        headers: {
          Authorization: `Bearer \${env:${API_KEY_ENV_VAR}}`
        }
      }
    }
  }
  return JSON.stringify(config, null, 2) + '\n'
}

/**
 * `.vscode/mcp.json` — VS Code's variant. Same server entry under a different
 * top-level key.
 */
export function buildVsCodeMcpJson(input: McpSetupInput): string {
  const config = {
    servers: {
      [SERVER_NAME]: {
        type: 'http',
        url: buildMcpUrl(input),
        headers: {
          Authorization: `Bearer \${env:${API_KEY_ENV_VAR}}`
        }
      }
    }
  }
  return JSON.stringify(config, null, 2) + '\n'
}

/** `.github/copilot-instructions.md` — tells Copilot what the server is for. */
export function buildCopilotInstructions(input: McpSetupInput): string {
  const scope = [
    `- Project: **${input.projectName ?? input.projectId}** (\`${input.projectId}\`)`,
    input.agentId ? `- Agent: **${input.agentName ?? 'selected agent'}** (\`${input.agentId}\`)` : '- Agent: not bound — call `register_self` or scope per call'
  ].join('\n')

  const toolLines = TOOL_GROUPS.map((group) => {
    const bullets = group.tools.map((t) => `  - \`${t}\``).join('\n')
    return `### ${group.heading}\n\n${group.note}\n\n${bullets}\n`
  }).join('\n')

  return `# Agent Shaker — MCP integration

This repository is coordinated through the **Agent Shaker** MCP server. Use it
rather than tracking work in local notes: the server is the shared source of
truth, so other agents see what you do and you see what they do.

${scope}

Endpoint: \`${buildMcpUrl(input)}\`
Authentication: \`Authorization: Bearer \${env:${API_KEY_ENV_VAR}}\`

## How to work

1. Call \`get_my_identity\` to confirm which agent and project this connection
   is bound to. If it is not bound, call \`register_self\`.
2. \`get_my_tasks\` for the work assigned to you.
3. \`claim_task\` before starting, so other agents can see it is in progress.
4. \`complete_task\` when the work is genuinely done.
5. \`update_my_status\` when you are blocked or waiting on someone.
6. Read shared \`list_contexts\` before making a decision, and record the
   decision with \`add_context\` so the next agent inherits it.

## Available tools

${toolLines}
## Notes

- Writes to tasks and milestones are PM-gated; a non-PM agent gets a clear
  refusal rather than a silent no-op.
- Live updates also arrive over the app's WebSocket; the MCP connection is for
  acting, the UI is for watching.
`
}

/** A short README so the generated files explain themselves. */
export function buildSetupReadme(input: McpSetupInput): string {
  return `# Agent Shaker setup

Generated for project **${input.projectName ?? input.projectId}**.

## 1. Set the API key

The configs reference \`${API_KEY_ENV_VAR}\`; they never contain the key itself,
so they are safe to commit.

\`\`\`bash
export ${API_KEY_ENV_VAR}=<your key>
\`\`\`

The key is one of the values in the server's \`API_KEYS\` list. Keep it out of
tracked files.

## 2. Add the config

- \`.mcp.json\` — Visual Studio 2026 and MCP hosts using this convention
- \`.vscode/mcp.json\` — Visual Studio Code
- \`.github/copilot-instructions.md\` — tells Copilot what the server is for

## 3. Verify

\`\`\`bash
curl -sS -X POST "${buildMcpUrl(input)}" \\
  -H "Authorization: Bearer $${API_KEY_ENV_VAR}" \\
  -H 'Content-Type: application/json' \\
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
\`\`\`

An \`initialize\` result means the connection is good. A \`401\` means the key is
wrong; a \`404\` means the edge is not routing \`/mcp\` (that path is only
published under the \`tls\` profile — see docs/DEPLOYMENT.md).
`
}

/** Every file in the bundle, ready to download. */
export function buildSetupFiles(input: McpSetupInput): SetupFile[] {
  return [
    { path: '.mcp.json', contents: buildMcpJson(input) },
    { path: '.vscode/mcp.json', contents: buildVsCodeMcpJson(input) },
    { path: '.github/copilot-instructions.md', contents: buildCopilotInstructions(input) },
    { path: 'AGENT_SHAKER_SETUP.md', contents: buildSetupReadme(input) }
  ]
}

export function useMcpSetup() {
  /**
   * Download a single file.
   *
   * `type: 'application/json'` for JSON so editors do not treat it as a
   * download prompt, and the blob URL is revoked on the next tick so a
   * long-lived tab does not accumulate them.
   */
  function downloadFile(file: SetupFile): void {
    const mime = file.path.endsWith('.json') ? 'application/json' : 'text/markdown'
    const blob = new Blob([file.contents], { type: `${mime};charset=utf-8` })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = file.path.split('/').pop()!
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(url), 0)
  }

  /**
   * Download every file.
   *
   * Staggered because browsers silently drop concurrent downloads from one
   * gesture — without the delay only the first file would arrive.
   */
  function downloadAll(input: McpSetupInput): SetupFile[] {
    const files = buildSetupFiles(input)
    files.forEach((file, i) => setTimeout(() => downloadFile(file), i * 120))
    return files
  }

  return { downloadFile, downloadAll, API_KEY_ENV_VAR }
}
