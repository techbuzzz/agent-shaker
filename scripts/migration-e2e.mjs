// Proves that the migrations apply to a database that already holds data.
//
// The order matters, and it is the whole point:
//
//   1. Apply every migration but the journal.
//   2. Put rows in the database through the API — a project, an agent, a task
//      and a context.
//   3. Apply the journal migration on top of the data.
//   4. Check the rows are still there, and that the journal now records what
//      happens.
//
// A migration that only ever runs against an empty database proves nothing
// about the case it will actually meet: an `ALTER TABLE` over a table with
// rows in it behaves differently from one over an empty table.
//
// Usage:
//   node scripts/migration-e2e.mjs seed     # step 2
//   node scripts/migration-e2e.mjs verify   # step 4, with the seeded ids

import { readFileSync } from 'node:fs'

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

const mode = process.argv[2]

if (mode === 'seed') {
  const stamp = Date.now()

  const project = await call('/api/projects', 'POST', {
    name: `migration-e2e-${stamp}`,
    description: 'rows that must survive the next migration'
  })
  const agent = await call('/api/agents', 'POST', {
    project_id: project.id,
    name: `migration-agent-${stamp}`,
    role: 'developer'
  })
  const task = await call('/api/tasks', 'POST', {
    project_id: project.id,
    title: 'Survive the migration',
    description: 'a row that exists before the journal migration runs',
    priority: 'high',
    created_by: agent.id
  })
  const context = await call('/api/contexts', 'POST', {
    project_id: project.id,
    agent_id: agent.id,
    title: 'a context written before the migration',
    content: 'this row must still be here afterwards',
    tags: ['migration']
  })

  console.log(JSON.stringify({
    project_id: project.id,
    agent_id: agent.id,
    task_id: task.id,
    context_id: context.id
  }))
  process.exit(0)
}

if (mode === 'verify') {
  // The seeded ids arrive in a file. They used to arrive in an environment
  // variable, and Windows PowerShell 5.1 strips the double quotes out of an
  // argument before it reaches a native executable — so the JSON arrived
  // without them and failed to parse.
  const seededFile = process.env.AGENT_SHAKER_SEEDED_FILE
  const seeded = seededFile
    ? JSON.parse(readFileSync(seededFile, 'utf8').replace(/^﻿/, ''))
    : JSON.parse(process.env.SEEDED_JSON || '{}')

  for (const key of ['project_id', 'agent_id', 'task_id', 'context_id']) {
    if (!seeded[key]) fail(`the seeded ids are missing ${key}`)
  }

  // The rows written before the migration must still be readable.
  const project = await call(`/api/projects/${seeded.project_id}`, 'GET')
  if (project.id !== seeded.project_id) fail('the project did not survive the migration')

  const context = await call(`/api/contexts/${seeded.context_id}`, 'GET')
  if (context.id !== seeded.context_id) fail('the context did not survive the migration')

  const task = await call(`/api/tasks/${seeded.task_id}`, 'GET')
  if (task.id !== seeded.task_id) fail('the task did not survive the migration')

  console.log('✓ the rows written before the migration are still readable')

  // And the journal now records what happens. Finishing the task publishes a
  // domain event, which is what the journal writes down.
  await call(`/api/tasks/${seeded.task_id}/status`, 'PUT', { status: 'done' })

  const events = await call(`/api/activity?project_id=${seeded.project_id}`, 'GET')
  if (!Array.isArray(events.events) || events.events.length === 0) {
    fail('the journal recorded nothing')
  }

  const chained = events.events.filter((event) => event.hash)
  if (chained.length === 0) fail('no journal entry carries a hash')

  const recorded = events.events.some((event) => event.action === 'task.completed')
  if (!recorded) {
    fail(`the journal did not record the completion: ${events.events.map((e) => e.action).join(', ')}`)
  }

  console.log(`✓ the journal recorded ${events.events.length} entries, ${chained.length} of them chained`)
  console.log(`✓ chain head ${chained[0].hash.slice(0, 16)}…`)
  process.exit(0)
}

fail('usage: migration-e2e.mjs seed | verify')
