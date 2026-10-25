import assert from "node:assert/strict";
import { test } from "node:test";
import {
  ShakerClient,
  ShakerError,
  ShakerTimeoutError,
  ShakerTransportError,
  type FetchInitLike,
} from "../src/core/client.js";

function rpcResponse(payload: unknown, isError = false): string {
  return JSON.stringify({
    jsonrpc: "2.0",
    id: 1,
    result: { content: [{ type: "text", text: JSON.stringify(payload) }], isError },
  });
}

test("the client speaks the MCP JSON-RPC envelope", async () => {
  const requests: Array<{ url: string; init: FetchInitLike }> = [];
  const client = new ShakerClient({
    endpoint: "http://shaker.test/mcp",
    apiKey: "secret",
    fetchImpl: async (url, init) => {
      requests.push({ url, init });
      return {
        ok: true,
        status: 200,
        text: async () => rpcResponse({ success: true, agent_id: "agent-1", status: "active", message: "ok" }),
      };
    },
  });

  const result = await client.updateMyStatus("active");
  assert.equal(result.status, "active");

  const request = requests[0];
  assert.ok(request);
  assert.equal(request.url, "http://shaker.test/mcp");
  assert.equal(request.init.headers["X-API-Key"], "secret");

  const body = JSON.parse(request.init.body) as {
    jsonrpc: string;
    method: string;
    params: { name: string; arguments: Record<string, unknown> };
  };
  assert.equal(body.jsonrpc, "2.0");
  assert.equal(body.method, "tools/call");
  assert.equal(body.params.name, "update_my_status");
  assert.deepEqual(body.params.arguments, { status: "active" });
});

test("the heartbeat is sent with the server's vocabulary", async () => {
  const requests: Array<{ url: string; init: FetchInitLike }> = [];
  const client = new ShakerClient({
    endpoint: "http://shaker.test/mcp",
    apiKey: "secret",
    fetchImpl: async (url, init) => {
      requests.push({ url, init });
      return {
        ok: true,
        status: 200,
        text: async () =>
          rpcResponse({
            success: true,
            agent_id: "agent-1",
            status: "active",
            lease_ttl_seconds: 60,
            next_heartbeat_before: "2026-01-01T00:01:00Z",
            broadcast: true,
          }),
      };
    },
  });

  const beat = await client.sendHeartbeat({ status: "working" });
  assert.equal(beat.lease_ttl_seconds, 60);

  const body = JSON.parse(requests[0]?.init.body ?? "{}") as { params: { arguments: Record<string, unknown> } };
  assert.deepEqual(body.params.arguments, { status: "working" });
});

test("a tool error becomes a ShakerError", async () => {
  const client = new ShakerClient({
    endpoint: "http://shaker.test/mcp",
    apiKey: "secret",
    fetchImpl: async () => ({
      ok: true,
      status: 200,
      text: async () => rpcResponse({ error: "status is required" }, true),
    }),
  });

  await assert.rejects(() => client.updateMyStatus("active"), ShakerError);
});

test("a transport failure is not a tool failure", async () => {
  const client = new ShakerClient({
    endpoint: "http://shaker.test/mcp",
    apiKey: "secret",
    fetchImpl: async () => ({ ok: false, status: 503, text: async () => "unavailable" }),
  });

  await assert.rejects(() => client.getMyTasks(), ShakerTransportError);
});

test("a slow server is a timeout, not a hang", async () => {
  const client = new ShakerClient({
    endpoint: "http://shaker.test/mcp",
    apiKey: "secret",
    timeoutMs: 20,
    fetchImpl: (_url, init) =>
      new Promise((_resolve, reject) => {
        init.signal.addEventListener("abort", () => reject(new Error("aborted")));
      }),
  });

  await assert.rejects(() => client.getMyTasks(), ShakerTimeoutError);
});

test("a JSON-RPC error is surfaced with its code", async () => {
  const client = new ShakerClient({
    endpoint: "http://shaker.test/mcp",
    apiKey: "secret",
    fetchImpl: async () => ({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ jsonrpc: "2.0", id: 1, error: { code: -32001, message: "identity mismatch" } }),
    }),
  });

  await assert.rejects(() => client.getIdentity(), /identity mismatch/);
});
