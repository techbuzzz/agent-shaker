// Proves the MiniMax Code → Agent Shaker bridge.
//
// The bridge is a project hook (`.minimax/hooks.json`) that calls a script
// (`.minimax/agent-shaker-hook.mjs`). The hook fires inside MiniMax Code, and
// MiniMax Code is not running here — so this drives the *script* directly,
// with the same payload on stdin and the same environment MiniMax Code would
// give it. That proves the part that is ours: the script reads the payload,
// resolves the agent, renews the lease and pushes the compacted transcript.
//
// It does not prove that MiniMax Code invokes the hook; that needs the host,
// and the hook stays inert until an operator enables it in Settings → Hooks →
// Project.
//
// Usage (inside the test stack's network):
//   AGENT_SHAKER_URL=http://api:8080 node scripts/minimax-hook-e2e.mjs

import { execFile } from 'node:child_process'
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const BASE = (process.env.AGENT_SHAKER_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '')
const ADMIN_KEY = process.env.AGENT_SHAKER_ADMIN_KEY || 'e2e-only'
const HOOK_SCRIPT = process.env.AGENT_SHAKER_HOOK_SCRIPT || '.minimax/agent-shaker-hook.mjs'

function fail(message) {
  console.error(`✗ ${message}`)
  process.exit(1)
}

async function call(path, method, body, key = ADMIN_KEY) {
  const response = await fetch(`${BASE}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', 'X-API-Key': key },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const text = await response.text()
  if (!response.ok) {
    throw new Error(`${method} ${path} → ${response.status}: ${text.slice(0, 300)}`)
  }
  return text ? JSON.parse(text) : {}
}

/** Runs the hook script the way MiniMax Code would: payload on stdin, environment in place. */
function runHook(payload, env) {
  return new Promise((resolve, reject) => {
    const child = execFile(
      process.execPath,
      [HOOK_SCRIPT],
      { env: { ...process.env, ...env } },
      (error, stdout, stderr) => {
        if (error) {
          reject(new Error(`${error.message}\n${stderr}`))
          return
        }
        resolve({ stdout, stderr })
      }
    )
    child.stdin.end(JSON.stringify(payload))
  })
}

const stamp = Date.now()

const project = await call('/api/projects', 'POST', {
  name: `minimax-hook-e2e-${stamp}`,
  description: 'proves the MiniMax Code hook bridge',
})
console.log(`✓ project ${project.id}`)

const agent = await call('/api/agents', 'POST', {
  project_id: project.id,
  name: `minimax-hook-agent-${stamp}`,
  role: 'developer',
})
console.log(`✓ agent ${agent.id}`)

// The credential the hook runs with is bound to this agent, which is the whole
// point: the hook may only renew its own lease and write its own contexts.
const connection = await call(`/api/agents/${agent.id}/connection`, 'POST', { format: 'opencode-v2' })
const agentKey = connection.key.secret
if (!agentKey) {
  fail('the connection response carried no plaintext key')
}
console.log('✓ an agent-bound credential was minted')

const env = {
  AGENT_SHAKER_URL: BASE,
  AGENT_SHAKER_API_KEY: agentKey,
  AGENT_SHAKER_PROJECT_ID: project.id,
  AGENT_SHAKER_AGENT_ID: agent.id
}

// --- SessionStart: the lease -------------------------------------------------
const before = await call(`/api/agents/${agent.id}`, 'GET', undefined, agentKey)
const lastSeenBefore = before.last_seen ?? ''

await runHook({ hook_event_name: 'SessionStart', session_id: `session-${stamp}`, cwd: process.cwd() }, env)

const after = await call(`/api/agents/${agent.id}`, 'GET', undefined, agentKey)
if (after.last_seen === lastSeenBefore) {
  fail(`the lease was not renewed (last_seen is still ${lastSeenBefore})`)
}
console.log(`✓ SessionStart renewed the lease (last_seen ${lastSeenBefore || '∅' } → ${after.last_seen})`)

// --- PostCompact: the context ------------------------------------------------
const workdir = mkdtempSync(join(tmpdir(), 'minimax-hook-'))
const transcriptPath = join(workdir, 'transcript.jsonl')
writeFileSync(
  transcriptPath,
  [
    JSON.stringify({ role: 'user', content: { text: 'Wire the hook bridge into the coordination plane.' } }),
    JSON.stringify({ role: 'assistant', content: { text: 'The bridge reads the payload and renews the lease.' } })
  ].join('\n') + '\n'
)

await runHook(
  {
    hook_event_name: 'PostCompact',
    session_id: `session-${stamp}`,
    trigger: 'auto',
    transcript_path: transcriptPath
  },
  env
)

const contexts = await call(`/api/contexts?project_id=${project.id}`, 'GET', undefined, agentKey)
const pushed = (contexts.contexts ?? contexts ?? []).find((entry) =>
  String(entry.title ?? '').includes(`session-${stamp}`)
)
if (!pushed) {
  fail('the compacted transcript was not pushed as a context')
}
if (!String(pushed.content ?? '').includes('Wire the hook bridge')) {
  fail(`the pushed context does not carry the transcript: ${pushed.content}`)
}
console.log(`✓ PostCompact pushed the transcript as context ${pushed.id}`)

rmSync(workdir, { recursive: true, force: true })

console.log('\n✓ the MiniMax Code hook bridge reaches the server')
