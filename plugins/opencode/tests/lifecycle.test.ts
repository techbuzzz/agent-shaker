import assert from "node:assert/strict";
import { test } from "node:test";
import { EVENT_CALLS, LifecycleBridge, nextHeartbeatDelayMs } from "../src/core/lifecycle.js";
import { createLogger } from "../src/core/logger.js";
import { FakeTimers, StubShaker } from "./support/stub.js";

const logger = createLogger({});
const HEARTBEAT = { leaseFraction: 0.6, minIntervalMs: 5_000, maxIntervalMs: 300_000 };

test("the event table maps every lifecycle event to its coordination calls", async () => {
  const expectations = [
    { event: "session.created", calls: ["send_heartbeat"], args: [{ status: "working" }] },
    {
      event: "session.idle",
      calls: ["update_my_status", "send_heartbeat"],
      args: [{ status: "idle" }, { status: "idle" }],
    },
    { event: "session.error", calls: ["update_my_status"], args: [{ status: "blocked" }] },
    { event: "session.deleted", calls: ["update_my_status"], args: [{ status: "offline" }] },
  ] as const;

  for (const expectation of expectations) {
    const stub = new StubShaker();
    const bridge = new LifecycleBridge(stub, { heartbeat: HEARTBEAT, timers: new FakeTimers() }, logger);
    await bridge.handleEvent({ type: expectation.event, sessionId: "session-1" });
    assert.deepEqual(stub.names(), [...expectation.calls], expectation.event);
    assert.deepEqual(
      stub.calls.map((call) => call.args),
      expectation.args.map((args) => ({ ...args })),
      expectation.event,
    );
  }
});

test("the table itself covers the four lifecycle events and nothing else", () => {
  assert.deepEqual(Object.keys(EVENT_CALLS).sort(), [
    "session.created",
    "session.deleted",
    "session.error",
    "session.execution.failed",
    "session.idle",
  ]);
});

test("an event the table does not know about does nothing", async () => {
  const stub = new StubShaker();
  const bridge = new LifecycleBridge(stub, { heartbeat: HEARTBEAT, timers: new FakeTimers() }, logger);
  const calls = await bridge.handleEvent({ type: "file.edited", sessionId: "session-1" });
  assert.deepEqual(calls, []);
  assert.deepEqual(stub.names(), []);
});

test("a deleted session drops the lease instead of waiting for the reaper", async () => {
  const stub = new StubShaker();
  const timers = new FakeTimers();
  const bridge = new LifecycleBridge(stub, { heartbeat: HEARTBEAT, timers }, logger);

  await bridge.handleEvent({ type: "session.created", sessionId: "session-1" });
  assert.equal(timers.scheduled.length, 1);

  await bridge.handleEvent({ type: "session.deleted", sessionId: "session-1" });
  assert.equal(stub.names().at(-1), "update_my_status");
  assert.equal(timers.last?.cancelled, true);
});

test("the next beat is scheduled from the lease the server granted", async () => {
  const timers = new FakeTimers();
  const stub = new StubShaker({ heartbeat: { lease_ttl_seconds: 120 } });
  const bridge = new LifecycleBridge(stub, { heartbeat: HEARTBEAT, timers }, logger);

  await bridge.handleEvent({ type: "session.created", sessionId: "session-1" });
  assert.equal(timers.last?.delayMs, 72_000);
});

test("the heartbeat interval is derived from the lease, not from a constant", () => {
  assert.equal(nextHeartbeatDelayMs(60, HEARTBEAT), 36_000);
  assert.equal(nextHeartbeatDelayMs(120, HEARTBEAT), 72_000);
  assert.equal(nextHeartbeatDelayMs(10, HEARTBEAT), 6_000);
});

test("the heartbeat never falls outside the lease", () => {
  // A one second lease cannot be beaten in 5 seconds: the agent would lose the
  // lease before the beat was sent.
  assert.equal(nextHeartbeatDelayMs(1, HEARTBEAT), 1_000);
  assert.equal(nextHeartbeatDelayMs(0, HEARTBEAT), 1);
});

test("a failed heartbeat retries soon instead of waiting out the lease", async () => {
  const timers = new FakeTimers();
  const stub = new StubShaker({ failOn: { send_heartbeat: new Error("connection refused") } });
  const bridge = new LifecycleBridge(stub, { heartbeat: HEARTBEAT, timers }, logger);

  await bridge.handleEvent({ type: "session.created", sessionId: "session-1" });
  assert.equal(timers.last?.delayMs, HEARTBEAT.minIntervalMs);
});
