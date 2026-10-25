import assert from "node:assert/strict";
import { test } from "node:test";
import { setupV2 } from "../src/adapters/v2.js";
import { createV1Hooks } from "../src/adapters/v1.js";
import { buildCompactionMetadata, buildCompactionSummary, resolveCompactionMode } from "../src/core/compaction.js";
import { createFakeV2Host, fakeV1Input } from "./support/hosts.js";
import { StubShaker, makeTask } from "./support/stub.js";

const OPTIONS = { endpoint: "http://shaker.test/mcp", api_key: "key" };

test("the v2 compaction hook authors the summary and skips the model call", async () => {
  const stub = new StubShaker({
    tasks: [makeTask({ title: "Wire the compaction bridge" })],
    contexts: [
      {
        id: "context-1",
        title: "Lease rules",
        content: "The lease is renewed every beat.",
        created_at: "2026-01-02T00:00:00Z",
      },
    ],
  });
  const host = createFakeV2Host(OPTIONS);
  await setupV2(host.ctx, { createClient: () => stub, env: {} });

  const hook = host.sessionHooks.get("compaction");
  assert.ok(hook, "the compaction hook must be registered");

  const input: {
    sessionID: string;
    system: Array<{ type: string; text: string }>;
    result?: { summary: string; metadata?: Record<string, unknown> };
  } = { sessionID: "session-1", system: [] };

  await hook(input as never);

  assert.ok(input.result, "the compaction must be authored by the plugin");
  assert.match(input.result.summary, /Wire the compaction bridge/);
  assert.match(input.result.summary, /Lease rules/);
  assert.equal(input.result.metadata?.["project_id"], "project-1");
  assert.equal(input.result.metadata?.["task_id"], "task-1");
  assert.equal(input.result.metadata?.["agent_id"], "agent-1");
  assert.equal(input.result.metadata?.["session_id"], "session-1");
  assert.equal(input.result.metadata?.["source"], "agent-shaker");
});

test("whatever the plugin authors is also stored in the coordination plane", async () => {
  const stub = new StubShaker({ tasks: [makeTask()] });
  const host = createFakeV2Host(OPTIONS);
  await setupV2(host.ctx, { createClient: () => stub, env: {} });

  const hook = host.sessionHooks.get("compaction");
  const input: { sessionID: string; system: Array<{ type: string; text: string }>; result?: unknown } = {
    sessionID: "session-1",
    system: [],
  };
  await hook?.(input as never);

  const stored = stub.callsTo("add_context");
  assert.equal(stored.length, 1);
  assert.match(String(stored[0]?.args["content"]), /Ship the plugin/);
});

test("the v1 compaction hook influences the host summary and stores the digest", async () => {
  const stub = new StubShaker({ tasks: [makeTask({ title: "Influence the host summary" })] });
  const hooks = await createV1Hooks(fakeV1Input(), OPTIONS, { createClient: () => stub, env: {} });

  const output = { context: [] as string[], prompt: undefined as string | undefined };
  await hooks["experimental.session.compacting"]?.({ sessionID: "session-1" }, output);

  assert.ok(output.context.length > 0, "the plugin must contribute to the host prompt");
  assert.match(output.context.join("\n"), /Influence the host summary/);
  assert.match(output.context.join("\n"), /Coordination identifiers/);

  const stored = stub.callsTo("add_context");
  assert.equal(stored.length, 1);
  assert.match(String(stored[0]?.args["content"]), /Influence the host summary/);
});

test("the summary carries the shared truth, not the transcript", () => {
  const summary = buildCompactionSummary({
    projectId: "project-1",
    agentId: "agent-1",
    sessionId: "session-1",
    task: makeTask({ title: "Do the thing" }),
    blockers: [
      {
        id: "blocker-1",
        task_id: "task-1",
        agent_id: "agent-1",
        project_id: "project-1",
        blocker_type: "needs_human",
        title: "Waiting on credentials",
        description: "The staging token is missing",
        status: "open",
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
    ],
    dependsOn: [{ task_id: "task-0", title: "Set up the pipeline", status: "in_progress" }],
    dependsFor: [],
    contexts: [{ title: "Runbook", content: "Deploy with make ship" }],
  });

  assert.match(summary, /Do the thing/);
  assert.match(summary, /Waiting on credentials/);
  assert.match(summary, /Set up the pipeline/);
  assert.match(summary, /Runbook/);
});

test("the metadata names the coordination identifiers", () => {
  const metadata = buildCompactionMetadata({
    projectId: "project-1",
    agentId: "agent-1",
    sessionId: "session-1",
    task: makeTask(),
    blockers: [],
    dependsOn: [],
    dependsFor: [],
    contexts: [],
  });

  assert.equal(metadata["project_id"], "project-1");
  assert.equal(metadata["agent_id"], "agent-1");
  assert.equal(metadata["session_id"], "session-1");
  assert.equal(metadata["task_id"], "task-1");
  assert.equal(metadata["source"], "agent-shaker");
});

test("the compaction mode is reported from what actually happened", () => {
  assert.equal(resolveCompactionMode(true, false), "authoring");
  assert.equal(resolveCompactionMode(false, true), "observation");
  assert.equal(resolveCompactionMode(true, true), "authoring+observation");
});
