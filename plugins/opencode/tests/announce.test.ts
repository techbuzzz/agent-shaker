import assert from "node:assert/strict";
import { test } from "node:test";
import { announceCapabilities, serverCapabilities } from "../src/core/announce.js";
import { createLogger } from "../src/core/logger.js";
import { StubShaker } from "./support/stub.js";

const logger = createLogger({});

const ALL_FEATURES = { policyEnabled: true, contextEnabled: true, evidenceEnabled: true };

test("a client with both surfaces claims every configured capability", () => {
  const capabilities = serverCapabilities({ sessionHook: true, toolHook: true }, ALL_FEATURES);
  assert.deepEqual(capabilities.sort(), [
    "compaction",
    "context_injection",
    "evidence_capture",
    "policy_gate",
  ]);
});

test("a client without the session hook claims no compaction or context capability", () => {
  const capabilities = serverCapabilities({ sessionHook: false, toolHook: true }, ALL_FEATURES);
  assert.deepEqual(capabilities.sort(), ["evidence_capture", "policy_gate"]);
});

test("a client without the tool hook claims no gate or evidence capability", () => {
  const capabilities = serverCapabilities({ sessionHook: true, toolHook: false }, ALL_FEATURES);
  assert.deepEqual(capabilities.sort(), ["compaction", "context_injection"]);
});

test("a disabled feature is not claimed even when the host supports it", () => {
  const capabilities = serverCapabilities(
    { sessionHook: true, toolHook: true },
    { policyEnabled: false, contextEnabled: false, evidenceEnabled: false },
  );
  assert.deepEqual(capabilities, ["compaction"]);
});

test("the announcement reaches the server with its capability list", async () => {
  const stub = new StubShaker();
  await announceCapabilities(stub, ["compaction", "policy_gate"], "opencode", "0.1.0", logger);

  const call = stub.calls.find((entry) => entry.name === "announce_capability");
  assert.ok(call, "the announcement was never made");
  assert.deepEqual(call.args["capabilities"], ["compaction", "policy_gate"]);
  assert.equal(call.args["host"], "opencode");
  assert.equal(call.args["plugin_version"], "0.1.0");
});

test("a client with nothing to announce says nothing", async () => {
  const stub = new StubShaker();
  await announceCapabilities(stub, [], "opencode", "0.1.0", logger);

  assert.equal(
    stub.calls.filter((entry) => entry.name === "announce_capability").length,
    0,
    "an empty announcement is a call the server does not need",
  );
});

test("a failed announcement does not take the session down", async () => {
  const stub = new StubShaker({ failOn: { announce_capability: new Error("connection refused") } });

  // The announcement is a statement about this client. A client that cannot
  // make the statement is still a client that can hold a lease.
  await announceCapabilities(stub, ["compaction"], "opencode", "0.1.0", logger);
});
