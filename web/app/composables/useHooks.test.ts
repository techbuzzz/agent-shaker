import { describe, expect, it, vi } from 'vitest'
import type { CreateHookInput, Hook, HookRun } from '../types/hooks'
import {
  createHooksStore,
  describeHookError,
  sortRuns,
  type HooksTransport
} from './useHooks'

function makeHook(overrides: Partial<Hook> = {}): Hook {
  return {
    id: 'h1',
    project_id: 'p1',
    event: 'task.completed',
    matcher: null,
    command: ['mcode', 'exec'],
    stdin: { mode: 'event' },
    post: null,
    timeout_seconds: 120,
    enabled: true,
    created_by: null,
    created_at: '2026-10-25T10:00:00Z',
    updated_at: '2026-10-25T10:00:00Z',
    ...overrides
  }
}

function makeRun(overrides: Partial<HookRun> = {}): HookRun {
  return {
    id: 'r1',
    hook_id: 'h1',
    delivery_id: 'd1',
    event: 'task.completed',
    entity_type: 'task',
    entity_id: 't1',
    status: 'succeeded',
    exit_code: 0,
    stdout: 'ok',
    stderr: '',
    duration_ms: 120,
    started_at: '2026-10-25T10:00:00Z',
    finished_at: '2026-10-25T10:00:02Z',
    ...overrides
  }
}

function makeTransport(overrides: Partial<HooksTransport> = {}): HooksTransport {
  return {
    listHooks: vi.fn().mockResolvedValue([]),
    createHook: vi.fn().mockResolvedValue(makeHook()),
    updateHook: vi.fn().mockResolvedValue(makeHook()),
    deleteHook: vi.fn().mockResolvedValue(undefined),
    listHookRuns: vi.fn().mockResolvedValue([]),
    testHook: vi.fn().mockResolvedValue(makeRun()),
    getHookSettings: vi.fn().mockResolvedValue(false),
    setHookSettings: vi.fn().mockResolvedValue(false),
    ...overrides
  }
}

const input: CreateHookInput = {
  event: 'task.completed',
  matcher: null,
  command: ['mcode', 'exec'],
  stdin: { mode: 'event' },
  post: null,
  timeout_seconds: 120,
  enabled: true
}

describe('createHooksStore', () => {
  it('loads the hooks and the run log for a project', async () => {
    const hook = makeHook()
    const run = makeRun()
    const transport = makeTransport({
      listHooks: vi.fn().mockResolvedValue([hook]),
      listHookRuns: vi.fn().mockResolvedValue([run])
    })
    const store = createHooksStore({ transport })

    await store.load('p1')

    expect(transport.listHooks).toHaveBeenCalledWith('p1')
    expect(transport.listHookRuns).toHaveBeenCalledWith('p1')
    expect(store.hooks.value).toEqual([hook])
    expect(store.runs.value).toEqual([run])
    expect(store.loading.value).toBe(false)
    expect(store.loadError.value).toBeNull()
  })

  it('keeps the hooks when only the run log fails', async () => {
    // Two endpoints, two resources. A server that answers the hook list but
    // not the run log should still show the hooks — one failed promise must
    // not empty the other list.
    const hook = makeHook()
    const transport = makeTransport({
      listHooks: vi.fn().mockResolvedValue([hook]),
      listHookRuns: vi.fn().mockRejectedValue({ data: { error: { message: 'hook_runs is missing' } } })
    })
    const store = createHooksStore({ transport })

    await store.load('p1')

    expect(store.hooks.value).toEqual([hook])
    expect(store.loadError.value).toBeNull()
    expect(store.runError.value).toBe('hook_runs is missing')
  })

  it('reports the server message when the list fails', async () => {
    const transport = makeTransport({
      listHooks: vi.fn().mockRejectedValue({ data: { error: { message: 'hooks are disabled for this project' } } })
    })
    const store = createHooksStore({ transport })

    await store.load('p1')

    expect(store.loadError.value).toBe('hooks are disabled for this project')
    expect(store.loading.value).toBe(false)
  })

  it('creates a hook and puts it in the list', async () => {
    const created = makeHook({ id: 'h2', event: 'task.ready' })
    const transport = makeTransport({ createHook: vi.fn().mockResolvedValue(created) })
    const store = createHooksStore({ transport })

    const result = await store.create('p1', { ...input, event: 'task.ready' })

    expect(transport.createHook).toHaveBeenCalledWith('p1', { ...input, event: 'task.ready' })
    expect(result).toEqual(created)
    expect(store.hooks.value).toEqual([created])
  })

  it('updates a hook in place rather than appending a second copy', async () => {
    const original = makeHook()
    const updated = makeHook({ enabled: false })
    const transport = makeTransport({
      listHooks: vi.fn().mockResolvedValue([original]),
      updateHook: vi.fn().mockResolvedValue(updated)
    })
    const store = createHooksStore({ transport })
    await store.load('p1')

    await store.update('h1', { enabled: false })

    expect(store.hooks.value).toEqual([updated])
    expect(store.hooks.value).toHaveLength(1)
  })

  it('toggles a hook without deleting it', async () => {
    const original = makeHook({ enabled: true })
    const updated = makeHook({ enabled: false })
    const transport = makeTransport({
      listHooks: vi.fn().mockResolvedValue([original]),
      updateHook: vi.fn().mockResolvedValue(updated)
    })
    const store = createHooksStore({ transport })
    await store.load('p1')

    const result = await store.toggle(original)

    expect(transport.updateHook).toHaveBeenCalledWith('h1', { enabled: false })
    expect(transport.deleteHook).not.toHaveBeenCalled()
    expect(result.enabled).toBe(false)
    expect(store.hooks.value).toHaveLength(1)
  })

  it('re-enables a disabled hook', async () => {
    const disabled = makeHook({ enabled: false })
    const enabled = makeHook({ enabled: true })
    const transport = makeTransport({ updateHook: vi.fn().mockResolvedValue(enabled) })
    const store = createHooksStore({ transport })

    await store.toggle(disabled)

    expect(transport.updateHook).toHaveBeenCalledWith('h1', { enabled: true })
  })

  it('sends the environment back on update', async () => {
    // The server replaces the stored environment with whatever the body
    // carries, so an update that leaves it out would drop the hook's
    // `{secret:NAME}` references. The caller decides what to send; the store
    // must not drop it on the way.
    const env = { TOKEN: '{secret:REVIEW_TOKEN}' }
    const transport = makeTransport({ updateHook: vi.fn().mockResolvedValue(makeHook({ env })) })
    const store = createHooksStore({ transport })

    await store.update('h1', { command: ['mcode', 'exec', '--review'], env })

    expect(transport.updateHook).toHaveBeenCalledWith('h1', {
      command: ['mcode', 'exec', '--review'],
      env
    })
  })

  it('deletes the hook and keeps its past runs', async () => {
    // The log outlives the hook: "this hook ran twice and was then deleted"
    // is exactly the row an operator needs.
    const hook = makeHook()
    const run = makeRun()
    const transport = makeTransport({
      listHooks: vi.fn().mockResolvedValue([hook]),
      listHookRuns: vi.fn().mockResolvedValue([run])
    })
    const store = createHooksStore({ transport })
    await store.load('p1')

    await store.remove('h1')

    expect(transport.deleteHook).toHaveBeenCalledWith('h1')
    expect(store.hooks.value).toEqual([])
    expect(store.runs.value).toEqual([run])
  })

  it('runs a test and keeps the result', async () => {
    const run = makeRun({ status: 'failed', exit_code: 2, stdout: 'boom' })
    const transport = makeTransport({ testHook: vi.fn().mockResolvedValue(run) })
    const store = createHooksStore({ transport })

    const result = await store.test('h1')

    expect(transport.testHook).toHaveBeenCalledWith('h1')
    expect(result).toEqual(run)
    expect(store.lastTestRun.value).toEqual(run)
    expect(store.runs.value[0]).toEqual(run)
  })

  it('re-throws a failed write with the server message', async () => {
    // The form dialog needs the rejection: a failed create that closed the
    // dialog silently would leave the operator with nothing to act on.
    const transport = makeTransport({
      createHook: vi.fn().mockRejectedValue({
        data: { error: { message: 'executable "rm" is not on the project allowlist' } }
      })
    })
    const store = createHooksStore({ transport })

    await expect(store.create('p1', input)).rejects.toBeTruthy()
    expect(store.actionError.value).toBe('executable "rm" is not on the project allowlist')
    expect(store.hooks.value).toEqual([])
  })

  it('reports a failed delete with the server message', async () => {
    const transport = makeTransport({
      deleteHook: vi.fn().mockRejectedValue({ data: { error: { message: 'hook is running' } } })
    })
    const store = createHooksStore({ transport })

    await expect(store.remove('h1')).rejects.toBeTruthy()
    expect(store.actionError.value).toBe('hook is running')
  })

  it('reports a failed test with the server message', async () => {
    const transport = makeTransport({
      testHook: vi.fn().mockRejectedValue({ data: { error: { message: 'hooks are disabled globally' } } })
    })
    const store = createHooksStore({ transport })

    await expect(store.test('h1')).rejects.toBeTruthy()
    expect(store.actionError.value).toBe('hooks are disabled globally')
    expect(store.lastTestRun.value).toBeNull()
  })

  it('clears the in-flight flags whether the action succeeds or fails', async () => {
    const ok = createHooksStore({ transport: makeTransport() })
    await ok.create('p1', input)
    expect(ok.saving.value).toBe(false)

    const bad = createHooksStore({
      transport: makeTransport({ createHook: vi.fn().mockRejectedValue(new Error('boom')) })
    })
    await expect(bad.create('p1', input)).rejects.toBeTruthy()
    expect(bad.saving.value).toBe(false)
  })

  it('clears the testing flag when the test fails', async () => {
    const store = createHooksStore({
      transport: makeTransport({ testHook: vi.fn().mockRejectedValue(new Error('boom')) })
    })

    await expect(store.test('h1')).rejects.toBeTruthy()
    expect(store.testingId.value).toBeNull()
  })

  it('marks the hook being tested so only its button waits', async () => {
    let release: (() => void) | null = null
    const transport = makeTransport({
      testHook: vi.fn().mockImplementation(() => new Promise<HookRun>((resolve) => {
        release = () => resolve(makeRun())
      }))
    })
    const store = createHooksStore({ transport })

    const pending = store.test('h1')
    expect(store.testingId.value).toBe('h1')

    release!()
    await pending
    expect(store.testingId.value).toBeNull()
  })
})

describe('describeHookError', () => {
  it('prefers the API envelope message', () => {
    expect(describeHookError({ data: { error: { message: 'command must not be empty' } } }, 'fallback'))
      .toBe('command must not be empty')
  })

  it('falls back to a flat body message', () => {
    expect(describeHookError({ data: { message: 'bad request' } }, 'fallback')).toBe('bad request')
  })

  it('falls back to a transport message', () => {
    expect(describeHookError(new Error('Failed to fetch'), 'fallback')).toBe('Failed to fetch')
  })

  it('does not show ofetch\'s method prefix as the reason', () => {
    // ofetch puts "[POST] ..." in `message`; the API's own words are the ones
    // written for a reader.
    expect(describeHookError({ message: '[POST] 500', data: { error: { message: 'not allowed' } } }, 'x'))
      .toBe('not allowed')
  })

  it('names the status code when there is nothing better to show', () => {
    expect(describeHookError({ message: '[POST] 500', statusCode: 500 }, 'Could not save the hook'))
      .toBe('Could not save the hook (HTTP 500)')
  })

  it('uses the supplied default when there is nothing to show', () => {
    expect(describeHookError(null, 'Could not load the hooks')).toBe('Could not load the hooks')
    expect(describeHookError({}, 'Could not load the hooks')).toBe('Could not load the hooks')
  })
})

describe('sortRuns', () => {
  it('puts the newest run first', () => {
    const older = makeRun({ id: 'old', started_at: '2026-10-25T09:00:00Z' })
    const newer = makeRun({ id: 'new', started_at: '2026-10-25T11:00:00Z' })

    expect(sortRuns([older, newer]).map((r) => r.id)).toEqual(['new', 'old'])
  })

  it('sinks a run with an unreadable timestamp to the bottom', () => {
    const broken = makeRun({ id: 'broken', started_at: 'not-a-date' })
    const good = makeRun({ id: 'good', started_at: '2026-10-25T11:00:00Z' })

    expect(sortRuns([broken, good]).map((r) => r.id)).toEqual(['good', 'broken'])
  })

  it('does not mutate the list it was given', () => {
    const runs = [
      makeRun({ id: 'old', started_at: '2026-10-25T09:00:00Z' }),
      makeRun({ id: 'new', started_at: '2026-10-25T11:00:00Z' })
    ]
    sortRuns(runs)
    expect(runs.map((r) => r.id)).toEqual(['old', 'new'])
  })

  it('reads the project switch along with the hooks', async () => {
    // The switch decides whether anything in the list runs at all, so the
    // interface has to know its value rather than assume it.
    const transport = makeTransport({ getHookSettings: vi.fn().mockResolvedValue(true) })
    const store = createHooksStore({ transport })

    await store.load('p1')

    expect(transport.getHookSettings).toHaveBeenCalledWith('p1')
    expect(store.projectEnabled.value).toBe(true)
  })

  it('still loads the hooks when the switch cannot be read', async () => {
    // The switch's value is advisory until it is written. A server that
    // answers the hook list but not the settings should still show the hooks.
    const hook = makeHook()
    const transport = makeTransport({
      listHooks: vi.fn().mockResolvedValue([hook]),
      getHookSettings: vi.fn().mockRejectedValue(new Error('boom'))
    })
    const store = createHooksStore({ transport })

    await store.load('p1')

    expect(store.hooks.value).toEqual([hook])
    expect(store.loadError.value).toBeNull()
  })

  it('turns the project switch on', async () => {
    const transport = makeTransport({ setHookSettings: vi.fn().mockResolvedValue(true) })
    const store = createHooksStore({ transport })

    await store.setProjectEnabled('p1', true)

    expect(transport.setHookSettings).toHaveBeenCalledWith('p1', true)
    expect(store.projectEnabled.value).toBe(true)
  })

  it('keeps the server state when the switch refuses to change', async () => {
    // The switch shows the server's state, so a refused change must not leave
    // the interface claiming something the server did not do.
    const transport = makeTransport({
      setHookSettings: vi.fn().mockRejectedValue({ data: { error: { message: 'forbidden' } } })
    })
    const store = createHooksStore({ transport })
    store.projectEnabled.value = true

    await expect(store.setProjectEnabled('p1', false)).rejects.toBeTruthy()

    expect(store.projectEnabled.value).toBe(true)
    expect(store.actionError.value).toBe('forbidden')
    expect(store.saving.value).toBe(false)
  })
})
