import { describe, expect, it } from 'vitest'
import {
  API_KEY_ENV_VAR,
  buildCopilotInstructions,
  buildMcpJson,
  buildMcpUrl,
  buildSetupFiles,
  buildSetupReadme,
  buildVsCodeMcpJson
} from '../useMcpSetup'

const base = {
  origin: 'https://shaker.example.com',
  projectId: 'proj-1',
  projectName: 'Payments Revamp'
}

describe('buildMcpUrl', () => {
  it('points at /mcp on the origin the browser is on', () => {
    expect(buildMcpUrl(base)).toBe('https://shaker.example.com/mcp?project_id=proj-1')
  })

  it('binds the agent when one is chosen', () => {
    const url = buildMcpUrl({ ...base, agentId: 'agent-7', agentName: 'ada' })
    expect(url).toContain('agent_id=agent-7')
    expect(new URL(url).searchParams.get('agent_id')).toBe('agent-7')
  })

  it('omits agent_id entirely when unbound', () => {
    expect(buildMcpUrl(base)).not.toContain('agent_id')
  })

  it('does not produce a double slash when the origin has a trailing one', () => {
    expect(buildMcpUrl({ ...base, origin: 'https://shaker.example.com/' }))
      .toBe('https://shaker.example.com/mcp?project_id=proj-1')
  })

  it('percent-encodes ids rather than emitting a broken query', () => {
    // An id containing & or # would otherwise split the query and silently
    // scope the connection to the wrong project.
    const url = buildMcpUrl({ ...base, projectId: 'a&b#c' })
    expect(new URL(url).searchParams.get('project_id')).toBe('a&b#c')
  })

  it('survives a port, which is how local and edge deployments differ', () => {
    expect(buildMcpUrl({ ...base, origin: 'https://localhost:8443' }))
      .toContain('https://localhost:8443/mcp?')
  })
})

describe('generated JSON configs', () => {
  it('.mcp.json parses and names the server', () => {
    const parsed = JSON.parse(buildMcpJson(base))
    expect(Object.keys(parsed.mcpServers)).toEqual(['agent-shaker'])
    expect(parsed.mcpServers['agent-shaker'].type).toBe('http')
    expect(parsed.mcpServers['agent-shaker'].url).toBe(buildMcpUrl(base))
  })

  it('.vscode/mcp.json uses the servers key VS Code expects', () => {
    const parsed = JSON.parse(buildVsCodeMcpJson(base))
    expect(parsed.mcpServers).toBeUndefined()
    expect(parsed.servers['agent-shaker'].url).toBe(buildMcpUrl(base))
  })

  it('references the key through an env var instead of embedding it', () => {
    // The whole security property of this feature: these files are committed to
    // the agent's repository, so a literal key would be a leaked key.
    for (const build of [buildMcpJson, buildVsCodeMcpJson]) {
      const parsed = JSON.parse(build(base))
      const server = parsed.mcpServers?.['agent-shaker'] ?? parsed.servers['agent-shaker']
      expect(server.headers.Authorization).toBe(`Bearer \${env:${API_KEY_ENV_VAR}}`)
    }
  })

  it('never emits a literal that looks like a credential', () => {
    const all = buildSetupFiles(base).map((f) => f.contents).join('\n')
    // A bcrypt or hex token appearing anywhere would mean a secret was baked
    // into a tracked file.
    expect(all).not.toMatch(/\$2[aby]\$\d\d\$/)
    expect(all).not.toMatch(/\b[0-9a-f]{40,}\b/i)
  })

  it('is stable across calls, so re-downloading produces no spurious diff', () => {
    expect(buildMcpJson(base)).toBe(buildMcpJson(base))
  })
})

describe('buildSetupFiles', () => {
  it('emits the four documented files', () => {
    expect(buildSetupFiles(base).map((f) => f.path)).toEqual([
      '.mcp.json',
      '.vscode/mcp.json',
      '.github/copilot-instructions.md',
      'AGENT_SHAKER_SETUP.md'
    ])
  })

  it('uses repo-relative paths that cannot escape the checkout', () => {
    for (const file of buildSetupFiles(base)) {
      expect(file.path.startsWith('/')).toBe(false)
      expect(file.path).not.toContain('..')
      expect(file.path).not.toContain('\\')
    }
  })

  it('reflects the selected agent in the instructions file', () => {
    const withAgent = buildCopilotInstructions({ ...base, agentId: 'agent-7', agentName: 'Ada' })
    const without = buildCopilotInstructions(base)
    expect(withAgent).toContain('Ada')
    expect(without).toContain('not bound')
  })

  it('documents the tools an agent needs for the core loop', () => {
    const md = buildCopilotInstructions(base)
    for (const tool of ['get_my_identity', 'get_my_tasks', 'claim_task', 'complete_task', 'add_context', 'register_self']) {
      expect(md).toContain(tool)
    }
  })
})

describe('buildSetupReadme', () => {
  it('tells the operator how to supply the key without committing it', () => {
    const md = buildSetupReadme(base)
    expect(md).toContain(API_KEY_ENV_VAR)
    expect(md).toContain('export')
    expect(md.toLowerCase()).toContain('never contain the key')
  })

  it('includes a copy-pasteable verification request', () => {
    expect(buildSetupReadme(base)).toContain('initialize')
  })
})
