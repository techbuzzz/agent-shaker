import assert from "node:assert/strict";
import { test } from "node:test";
import { buildEvidence, captureEvidence, extractCommand, isTestOrBuildCommand } from "../src/core/evidence.js";
import { StubShaker } from "./support/stub.js";

test("a test run is recognised", () => {
  assert.equal(isTestOrBuildCommand("bash", "npm test"), true);
  assert.equal(isTestOrBuildCommand("bash", "go test ./..."), true);
  assert.equal(isTestOrBuildCommand("bash", "pnpm run build"), true);
  assert.equal(isTestOrBuildCommand("bash", "cargo build --release"), true);
  assert.equal(isTestOrBuildCommand("bash", "pytest -q"), true);
});

test("a command that only mentions a test is not a test run", () => {
  assert.equal(isTestOrBuildCommand("bash", "git commit -m 'add tests for the gate'"), false);
  assert.equal(isTestOrBuildCommand("bash", "ls -la"), false);
  assert.equal(isTestOrBuildCommand("read", undefined), false);
});

test("a tool named after a build counts on its own", () => {
  assert.equal(isTestOrBuildCommand("build", undefined), true);
  assert.equal(isTestOrBuildCommand("run_tests", undefined), true);
});

test("the command is read out of the tool input", () => {
  assert.equal(extractCommand("bash", { command: "npm test" }), "npm test");
  assert.equal(extractCommand("bash", { script: "npm test" }), "npm test");
  assert.equal(extractCommand("edit", { filePath: "/tmp/a" }), undefined);
});

test("a passing run is recorded with its outcome", async () => {
  const stub = new StubShaker();
  await captureEvidence(stub, {
    tool: "bash",
    command: "npm test",
    status: "completed",
    output: "12 passing",
    sessionId: "session-1",
  });

  const stored = stub.callsTo("add_context");
  assert.equal(stored.length, 1);
  const title = String(stored[0]?.args["title"]);
  const content = String(stored[0]?.args["content"]);
  assert.match(title, /npm test — passed/);
  assert.match(content, /12 passing/);
  assert.deepEqual(stored[0]?.args["tags"], ["evidence", "success"]);
});

test("a failing run keeps the error message", async () => {
  const stub = new StubShaker();
  await captureEvidence(stub, {
    tool: "bash",
    command: "npm test",
    status: "error",
    errorMessage: "exit code 1",
    sessionId: "session-1",
  });

  const stored = stub.callsTo("add_context");
  const content = String(stored[0]?.args["content"]);
  assert.match(content, /exit code 1/);
  assert.deepEqual(stored[0]?.args["tags"], ["evidence", "failure"]);
});

test("the recorded output is bounded", () => {
  const entry = buildEvidence({
    tool: "bash",
    command: "npm test",
    status: "completed",
    output: "z".repeat(10_000),
  });
  assert.ok(entry.content.length < 10_000);
  assert.match(entry.content, /output truncated/);
});
