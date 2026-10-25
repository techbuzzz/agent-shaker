// Proves the idempotency keys on the mutating MCP tools.
//
// The 2026-07-28 revision of the MCP specification makes retries ordinary
// traffic: a broken response stream loses the in-flight request and the client
// re-issues it. The failure that would otherwise cause is not a failed request
// — it is a task claimed twice, and a state the server then reports as truth.
//
// So the proof is behavioural, not structural:
//
//   1. claim a task with a key, and keep the answer;
//   2. claim it again with the same key, and get the *same* answer — not a
//      second claim;
//   3. claim a *different* task with the same key, and get a real execution,
//      because the key identifies one logical operation and the request's
//      fingerprint says which one.
//
// The third step is the one that catches a naive implementation. Scoping the
// key by (tool, agent) alone would replay the first answer for the second
// task, and the agent would believe it had claimed a task it never touched.
//
// Usage:
//   node scripts/idempotency-e2e.mjs

const BASE = (process.env.AGENT_SHAKER_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '')
const ADMIN_KEY = process.env.AGENT_SHAKER_ADMIN_KEY || 'e2e-only'

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

function fail(message) {
  console.error(`✗ ${message}`)
  process.exit(1)
}

let requestId = 0

/**
 * One MCP tools/call over the JSON-RPC endpoint.
 *
 * The MCP endpoint resolves the agent from the URL, not from the key: the
 * connection URL carries `?agent_id=…`, and a call without it is answered
 * with "no agent_id configured" — which is the right answer, because an MCP
 * connection with no agent behind it has no business claiming tasks.
 */
async function callTool(mcpURL, apiKey, tool, args) {
  requestId += 1
  const response = await fetch(mcpURL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-API-Key': apiKey },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: requestId,
      method: 'tools/call',
      params: { name: tool, arguments: args }
    })
  })
  const text = await response.text()
  if (!response.ok) {
    throw new Error(`tools/call ${tool} → ${response.status}: ${text.slice(0, 300)}`)
  }
  const body = JSON.parse(text)
  if (body.error) {
    throw new Error(`tools/call ${tool} → ${JSON.stringify(body.error)}`)
  }
  return body.result
}

const stamp = Date.now()

const project = await call('/api/projects', 'POST', {
  name: `idempotency-e2e-${stamp}`,
  description: 'proves a retry does not claim a task twice'
})
console.log(`✓ project ${project.id}`)

const agent = await call('/api/agents', 'POST', {
  project_id: project.id,
  name: `idempotency-agent-${stamp}`,
  role: 'developer'
})
console.log(`✓ agent ${agent.id}`)

const first = await call('/api/tasks', 'POST', {
  project_id: project.id,
  title: 'Claimed once',
  description: 'the task a retry must not claim twice',
  priority: 'high',
  created_by: agent.id
})
const second = await call('/api/tasks', 'POST', {
  project_id: project.id,
  title: 'Claimed for real',
  description: 'the task a reused key must not confuse with the first',
  priority: 'high',
  created_by: agent.id
})
console.log(`✓ tasks ${first.id} ${second.id}`)

const connection = await call(`/api/agents/${agent.id}/connection`, 'POST', { format: 'opencode-v2' })
const apiKey = connection.key.secret
if (!apiKey) fail('the connection response carried no plaintext key')

const mcpURL = `${BASE}/mcp?agent_id=${agent.id}&project_id=${project.id}`

// --- the retry ---------------------------------------------------------------
const key = `e2e-key-${stamp}`

const answer = await callTool(mcpURL, apiKey, 'claim_task', { task_id: first.id, idempotency_key: key })
console.log(`✓ first claim: ${JSON.stringify(answer).slice(0, 120)}`)

const replay = await callTool(mcpURL, apiKey, 'claim_task', { task_id: first.id, idempotency_key: key })
if (JSON.stringify(answer) !== JSON.stringify(replay)) {
  fail(`the retry did not replay the first answer:\n  ${JSON.stringify(answer)}\n  ${JSON.stringify(replay)}`)
}
console.log('✓ the retry replayed the first answer')

// The observable outcome: the task is claimed, and claimed once.
const claimed = await call(`/api/tasks/${first.id}`, 'GET')
if (claimed.status !== 'in_progress') {
  fail(`the task is ${claimed.status}, want in_progress`)
}
if (claimed.assigned_to !== agent.id) {
  fail(`the task is assigned to ${claimed.assigned_to}, want ${agent.id}`)
}
console.log('✓ the task is claimed exactly once')

// --- the reused key ----------------------------------------------------------
const other = await callTool(mcpURL, apiKey, 'claim_task', { task_id: second.id, idempotency_key: key })
if (JSON.stringify(other) === JSON.stringify(answer)) {
  fail('a key reused for a different task replayed the first answer — the agent would believe it claimed a task it never touched')
}
const secondClaimed = await call(`/api/tasks/${second.id}`, 'GET')
if (secondClaimed.status !== 'in_progress' || secondClaimed.assigned_to !== agent.id) {
  fail(`the second task is ${secondClaimed.status} assigned to ${secondClaimed.assigned_to}`)
}
console.log('✓ the same key on a different task executed for real')

console.log('\n✓ a retry does not claim a task twice')
