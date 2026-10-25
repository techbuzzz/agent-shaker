// End-to-end proof of the hook layer.
//
// It creates a project, an agent and a task, binds a hook to task.completed,
// completes the task, and waits for the hook to run. The point is the last
// step: the run log has to show the command's own output, captured from a
// command the server really executed.
//
// Run it against a server with the hook layer enabled:
//   AGENT_SHAKER_HOOKS_ENABLED=true
//   AGENT_SHAKER_HOOK_EXECUTABLES=sh
//
// Configuration comes from the environment:
//   AGENT_SHAKER_URL     base URL of the API (default http://127.0.0.1:8080)
//   AGENT_SHAKER_API_KEY the credential (default e2e-only)

const BASE = (process.env.AGENT_SHAKER_URL || 'http://127.0.0.1:8080').replace(/\/+$/, '')
const KEY = process.env.AGENT_SHAKER_API_KEY || 'e2e-only'

async function call(path, method, body) {
  const response = await fetch(`${BASE}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', 'X-API-Key': KEY },
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

const stamp = Date.now()

const project = await call('/api/projects', 'POST', {
  name: `hook-e2e-${stamp}`,
  description: 'hook layer end-to-end proof',
})
console.log(`✓ project ${project.id}`)

const agent = await call('/api/agents', 'POST', {
  project_id: project.id,
  name: `hook-e2e-agent-${stamp}`,
  role: 'developer',
})
console.log(`✓ agent ${agent.id}`)

const task = await call('/api/tasks', 'POST', {
  project_id: project.id,
  title: 'prove the hook layer runs a command',
  description: 'created by the end-to-end proof',
  priority: 'high',
  created_by: agent.id,
})
console.log(`✓ task ${task.id}`)

const created = await call(`/api/projects/${project.id}/hooks`, 'POST', {
  event: 'task.completed',
  matcher: { field: 'title', pattern: '^prove the hook layer' },
  command: ['sh', '-c', 'echo "hook ran for: $AGENT_SHAKER_ENTITY_TITLE"'],
  stdin: { mode: 'prompt', template: 'Task {{entity.title}} finished.' },
  timeout_seconds: 30,
  enabled: true,
})
const hook = created.hook
console.log(`✓ hook ${hook.id}`)

const test = await call(`/api/hooks/${hook.id}/test`, 'POST', {})
console.log(`✓ dry run: status=${test.run.status} exit=${test.run.exit_code}`)
if (test.run.status !== 'succeeded') {
  fail(`the dry run did not succeed: ${test.run.stderr || test.run.stdout}`)
}

// The real thing: complete the task and let the event fire the hook.
await call(`/api/tasks/${task.id}/status`, 'PUT', { status: 'done' })
console.log('✓ task completed')

let runs = []
for (let attempt = 0; attempt < 30; attempt += 1) {
  await new Promise((resolve) => setTimeout(resolve, 500))
  const response = await call(`/api/projects/${project.id}/hook-runs`, 'GET')
  runs = response.runs ?? []
  if (runs.length > 0 && runs.some((run) => run.event === 'task.completed')) break
}

if (runs.length === 0) {
  fail('no hook run was recorded')
}

const run = runs.find((entry) => entry.event === 'task.completed')
if (!run) {
  fail(`no run for task.completed, got: ${runs.map((entry) => entry.event).join(', ')}`)
}
if (run.status !== 'succeeded') {
  fail(`the hook run failed: ${run.status} (exit ${run.exit_code}) ${run.stderr}`)
}

console.log(`✓ hook run: status=${run.status} exit=${run.exit_code} duration=${run.duration_ms}ms`)
console.log(`✓ captured stdout: ${run.stdout.trim()}`)

if (!run.stdout.includes('hook ran for: prove the hook layer runs a command')) {
  fail(`the captured output does not carry the event payload: ${run.stdout}`)
}

console.log('\n✓ the hook layer ran a real command against a real event')
