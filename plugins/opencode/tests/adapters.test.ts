import assert from "node:assert/strict";
import { test } from "node:test";
import type { Event } from "@opencode-ai/sdk";
import { setupV2 } from "../src/adapters/v2.js";
import { createV1Hooks } from "../src/adapters/v1.js";
import { createFakeV2Host, fakeV1Input } from "./support/hosts.js";
import { FakeTimers, StubShaker, makeTask } from "./support/stub.js";

const OPTIONS = { endpoint: "http://shaker.test/mcp", api_key: "key" };

test("the v2 host drives the lifecycle bridge from its session events", async () => {
  const stub = new StubShaker();
  const host = createFakeV2Host(OPTIONS);
  await setupV2(host.ctx, { createClient: () => stub, env: {}, timers: new FakeTimers() });

  await host.emit({ type: "session.created", data: { sessionID: "session-1", projectID: "project-1" } });
  // The announcement comes first: the server is told what this client can do
  // before the client takes a lease.
  assert.deepEqual(stub.names(), ["announce_capability", "send_heartbeat"]);

  await host.emit({ type: "session.idle", data: { sessionID: "session-1" } });
  assert.deepEqual(stub.names(), [
    "announce_capability",
    "send_heartbeat",
    "update_my_status",
    "send_heartbeat",
  ]);

  await host.emit({ type: "session.deleted", data: { sessionID: "session-1" } });
  assert.equal(stub.callsTo("update_my_status").at(-1)?.args["status"], "offline");
});

test("the v2 host injects the shared context into the model request", async () => {
  const stub = new StubShaker({
    contexts: [
      {
        id: "context-1",
        title: "Deployment rules",
        content: "Ship behind the flag.",
        created_at: "2026-01-02T00:00:00Z",
      },
    ],
  });
  const host = createFakeV2Host(OPTIONS);
  await setupV2(host.ctx, { createClient: () => stub, env: {}, timers: new FakeTimers() });

  const hook = host.sessionHooks.get("context");
  assert.ok(hook, "the context hook must be registered");

  const system: Array<{ type: string; text: string }> = [];
  await hook?.({ sessionID: "session-1", system } as never);

  assert.equal(system.length, 1);
  assert.match(system[0]?.text ?? "", /Deployment rules/);
});

test("the v2 host injects the same context on the compaction and generate hooks", async () => {
  const host = createFakeV2Host(OPTIONS);
  await setupV2(host.ctx, { createClient: () => new StubShaker(), env: {}, timers: new FakeTimers() });

  assert.ok(host.sessionHooks.has("context"));
  assert.ok(host.sessionHooks.has("compaction"));
  assert.ok(host.sessionHooks.has("generate"));
});

test("the v2 tool gate blocks a mutating call when the agent has no task", async () => {
  const stub = new StubShaker({ tasks: [] });
  const host = createFakeV2Host({ ...OPTIONS, policy: { enabled: true } });
  await setupV2(host.ctx, { createClient: () => stub, env: {}, timers: new FakeTimers() });

  const hook = host.toolHooks.get("execute.before");
  assert.ok(hook, "the tool hook must be registered");

  await assert.rejects(
    async () => {
      await hook?.({ tool: "bash", sessionID: "session-1", input: { command: "rm -rf /" } } as never);
    },
    /no claimed task/,
  );
});

test("the v2 tool gate lets a mutating call through once a task is claimed", async () => {
  const stub = new StubShaker({ tasks: [makeTask()] });
  const host = createFakeV2Host({ ...OPTIONS, policy: { enabled: true } });
  await setupV2(host.ctx, { createClient: () => stub, env: {}, timers: new FakeTimers() });

  const hook = host.toolHooks.get("execute.before");
  await hook?.({ tool: "bash", sessionID: "session-1", input: { command: "ls" } } as never);
});

test("the v2 tool hook records the outcome of a test run", async () => {
  const stub = new StubShaker();
  const host = createFakeV2Host(OPTIONS);
  await setupV2(host.ctx, { createClient: () => stub, env: {}, timers: new FakeTimers() });

  const hook = host.toolHooks.get("execute.after");
  await hook?.({
    tool: "bash",
    sessionID: "session-1",
    status: "error",
    error: { message: "exit code 1" },
    input: { command: "npm test" },
  } as never);

  const stored = stub.callsTo("add_context");
  assert.equal(stored.length, 1);
  assert.match(String(stored[0]?.args["content"]), /exit code 1/);
});

test("the v1 host drives the lifecycle bridge from its session events", async () => {
  const stub = new StubShaker();
  const hooks = await createV1Hooks(fakeV1Input(), OPTIONS, {
    createClient: () => stub,
    env: {},
    timers: new FakeTimers(),
  });

  const created = {
    type: "session.created",
    properties: { info: { id: "session-1", projectID: "project-1" } },
  } as unknown as Event;

  await hooks.event?.({ event: created });
  assert.deepEqual(stub.names(), ["announce_capability", "send_heartbeat"]);

  await hooks.event?.({ event: { type: "session.idle", properties: { sessionID: "session-1" } } as unknown as Event });
  assert.deepEqual(stub.names(), [
    "announce_capability",
    "send_heartbeat",
    "update_my_status",
    "send_heartbeat",
  ]);
});

test("the v1 host injects the shared context into the system prompt", async () => {
  const stub = new StubShaker({
    contexts: [
      {
        id: "context-1",
        title: "House rules",
        content: "Never touch the migration folder.",
        created_at: "2026-01-02T00:00:00Z",
      },
    ],
  });
  const hooks = await createV1Hooks(fakeV1Input(), OPTIONS, {
    createClient: () => stub,
    env: {},
    timers: new FakeTimers(),
  });

  const output = { system: [] as string[] };
  await hooks["experimental.chat.system.transform"]?.({ sessionID: "session-1" } as never, output);

  assert.equal(output.system.length, 1);
  assert.match(output.system[0] ?? "", /House rules/);
});

test("the v1 tool gate blocks a mutating call when the agent has no task", async () => {
  const stub = new StubShaker({ tasks: [] });
  const hooks = await createV1Hooks(fakeV1Input(), { ...OPTIONS, policy: { enabled: true } }, {
    createClient: () => stub,
    env: {},
    timers: new FakeTimers(),
  });

  await assert.rejects(
    async () => {
      await hooks["tool.execute.before"]?.(
        { tool: "bash", sessionID: "session-1", callID: "1" } as never,
        { args: {} },
      );
    },
    /no claimed task/,
  );
});
