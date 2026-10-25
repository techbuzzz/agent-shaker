import assert from "node:assert/strict";
import { test } from "node:test";
import {
  CAPABILITY_EVENT_STREAM,
  CAPABILITY_SESSION_HOOK,
  CAPABILITY_TOOL_HOOK,
  assertLifecycleCapability,
  describeMissing,
  probeCapabilities,
} from "../src/core/capabilities.js";

test("a host without an event stream is refused", () => {
  const report = probeCapabilities({ session: { hook: () => undefined } });
  assert.equal(report.present.includes(CAPABILITY_EVENT_STREAM), false);
  assert.throws(() => assertLifecycleCapability(report), /event stream/);
});

test("an optional capability is reported as missing rather than fatal", () => {
  const report = probeCapabilities({ event: { subscribe: () => undefined } });
  assert.equal(report.present.includes(CAPABILITY_EVENT_STREAM), true);
  assert.equal(report.present.includes(CAPABILITY_SESSION_HOOK), false);
  assert.equal(report.present.includes(CAPABILITY_TOOL_HOOK), false);
  assert.doesNotThrow(() => assertLifecycleCapability(report));
});

test("the missing capabilities are named", () => {
  const report = probeCapabilities({ event: { subscribe: () => undefined } });
  assert.equal(describeMissing(report), "session-hook, tool-hook");
});

test("a full host reports nothing missing", () => {
  const report = probeCapabilities({
    event: { subscribe: () => undefined },
    session: { hook: () => undefined },
    tool: { hook: () => undefined },
  });
  assert.deepEqual(report.missing, []);
  assert.doesNotThrow(() => assertLifecycleCapability(report));
});
