import assert from "node:assert/strict";
import { test } from "node:test";
import { createLogger } from "../src/core/logger.js";
import { PolicyGate } from "../src/core/policy.js";
import { StubShaker, makeTask } from "./support/stub.js";

const logger = createLogger({});
const READ_ONLY = new Set(["read", "grep"]);

function gate(stub: StubShaker, failClosed = false): PolicyGate {
  return new PolicyGate(stub, { enabled: true, failClosed, readOnlyTools: READ_ONLY }, logger);
}

test("a mutating tool is blocked when the agent holds no task", async () => {
  const stub = new StubShaker({ tasks: [] });
  const decision = await gate(stub).evaluate("bash");
  assert.equal(decision.allowed, false);
  assert.match(decision.reason, /no claimed task/);
});

test("a mutating tool is allowed once the agent holds a task", async () => {
  const stub = new StubShaker({ tasks: [makeTask({ status: "in_progress" })] });
  const decision = await gate(stub).evaluate("bash");
  assert.equal(decision.allowed, true);
});

test("a blocked task still counts as claimed work", async () => {
  const stub = new StubShaker({ tasks: [makeTask({ status: "blocked" })] });
  const decision = await gate(stub).evaluate("edit");
  assert.equal(decision.allowed, true);
});

test("a finished task does not authorise anything", async () => {
  const stub = new StubShaker({ tasks: [makeTask({ status: "done" })] });
  const decision = await gate(stub).evaluate("edit");
  assert.equal(decision.allowed, false);
});

test("read-only tools are allowed without a task", async () => {
  const stub = new StubShaker({ tasks: [] });
  const decision = await gate(stub).evaluate("read");
  assert.equal(decision.allowed, true);
});

test("the gate fails open when the coordination server cannot be reached", async () => {
  const stub = new StubShaker({ failOn: { get_my_tasks: new Error("connection refused") } });
  const decision = await gate(stub).evaluate("bash");
  assert.equal(decision.allowed, true);
  assert.match(decision.reason, /failing open/);
});

test("the gate fails closed when the project asks for it", async () => {
  const stub = new StubShaker({ failOn: { get_my_tasks: new Error("connection refused") } });
  const decision = await gate(stub, true).evaluate("bash");
  assert.equal(decision.allowed, false);
  assert.match(decision.reason, /fail closed/);
});

test("the gate is off unless the project turns it on", async () => {
  const stub = new StubShaker({ tasks: [] });
  const disabled = new PolicyGate(stub, { enabled: false, failClosed: false, readOnlyTools: READ_ONLY }, logger);
  const decision = await disabled.evaluate("bash");
  assert.equal(decision.allowed, true);
});
