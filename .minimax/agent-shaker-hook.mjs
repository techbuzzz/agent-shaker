// MiniMax Code → Agent Shaker bridge.
//
// MiniMax Code has hooks, so it needs no plugin: a project hook is enough to
// make a session visible to Agent Shaker and to keep the session's context
// alive across a compaction.
//
// The hook is side-effect only. It never blocks a turn, never prints to
// stdout (an empty stdout is what tells the runtime "nothing to add"), and
// always exits 0 — a bridge that could fail a turn would be turned off.
//
// Enabled from Settings → Hooks → Project.
//
// Configuration comes from the environment:
//   AGENT_SHAKER_URL        base URL of the Agent Shaker API (required)
//   AGENT_SHAKER_API_KEY    the agent's credential (required)
//   AGENT_SHAKER_PROJECT_ID the project the session belongs to (required)
//   AGENT_SHAKER_AGENT_ID   the agent this session is (required)

const TIMEOUT_MS = 4000

const config = {
  url: (process.env.AGENT_SHAKER_URL || '').replace(/\/+$/, ''),
  apiKey: process.env.AGENT_SHAKER_API_KEY || '',
  projectId: process.env.AGENT_SHAKER_PROJECT_ID || '',
  agentId: process.env.AGENT_SHAKER_AGENT_ID || '',
}

function configured() {
  return Boolean(config.url && config.apiKey && config.projectId && config.agentId)
}

async function call(path, method, body) {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS)

  try {
    const response = await fetch(`${config.url}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': config.apiKey,
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
    })
    if (!response.ok) {
      throw new Error(`${method} ${path} → ${response.status}`)
    }
    return await response.json().catch(() => ({}))
  } finally {
    clearTimeout(timer)
  }
}

// heartbeat renews the lease the reaper enforces. An agent that stops
// reporting goes stale, and its tasks go back to the backlog.
async function heartbeat(status, taskId) {
  await call(
    `/api/projects/${config.projectId}/agents/${config.agentId}/heartbeat`,
    'POST',
    { status, current_task_id: taskId ?? null },
  )
}

// pushContext stores a piece of shared truth. It is what makes the session's
// work visible to the rest of the team.
async function pushContext(title, content) {
  await call('/api/contexts', 'POST', {
    project_id: config.projectId,
    agent_id: config.agentId,
    title,
    content,
    tags: ['minimax-code'],
  })
}

// readTranscript returns the tail of the session transcript, which is what
// survives into the shared store when the context is compacted.
async function readTranscript(path) {
  if (!path) return ''

  const { readFile } = await import('node:fs/promises')
  const raw = await readFile(path, 'utf8')

  const lines = raw.split('\n').filter(Boolean).slice(-40)
  const rendered = lines
    .map((line) => {
      try {
        const entry = JSON.parse(line)
        const role = entry.role ?? entry.type ?? 'entry'
        const text =
          entry.content?.text ??
          entry.text ??
          entry.message?.content ??
          ''
        return typeof text === 'string' ? `${role}: ${text}` : ''
      } catch {
        return ''
      }
    })
    .filter(Boolean)

  return rendered.join('\n').slice(-8000)
}

async function main() {
  const chunks = []
  for await (const chunk of process.stdin) chunks.push(chunk)

  let event = {}
  try {
    event = JSON.parse(Buffer.concat(chunks).toString('utf8') || '{}')
  } catch {
    return
  }

  if (!configured()) return

  const name = event.hook_event_name

  if (name === 'SessionStart') {
    await heartbeat('working')
    return
  }

  if (name === 'SessionEnd') {
    await heartbeat('offline')
    return
  }

  if (name === 'PostCompact') {
    const transcript = await readTranscript(event.transcript_path)
    if (!transcript.trim()) return

    await pushContext(
      `MiniMax Code session ${event.session_id ?? ''} after compaction`.trim(),
      transcript,
    )
  }
}

main().catch(() => {
  // Fail open, silently. A bridge that could fail a turn would be disabled.
})
