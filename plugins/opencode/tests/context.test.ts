import assert from "node:assert/strict";
import { test } from "node:test";
import { ContextCache, buildContextPayload, orderByRelevance } from "../src/core/context.js";

const OPTIONS = { maxChars: 4_000, maxItems: 8 };

test("the shared context is fetched once per prompt and reused for the rest of the turn", async () => {
  let fetches = 0;
  const cache = new ContextCache(
    async () => {
      fetches += 1;
      return [{ title: "note", content: "body" }];
    },
    OPTIONS,
  );

  await cache.get("session-1");
  await cache.get("session-1");
  await cache.get("session-1");
  assert.equal(fetches, 1, "a turn must not refetch on every model call");

  cache.beginPrompt("session-1");
  await cache.get("session-1");
  assert.equal(fetches, 2, "a new prompt must fetch again");
});

test("the cache is per session", async () => {
  let fetches = 0;
  const cache = new ContextCache(
    async () => {
      fetches += 1;
      return [{ title: "note", content: "body" }];
    },
    OPTIONS,
  );

  await cache.get("session-1");
  await cache.get("session-2");
  assert.equal(fetches, 2);
  await cache.get("session-1");
  assert.equal(fetches, 2);
});

test("a concurrent read shares one fetch", async () => {
  let fetches = 0;
  const cache = new ContextCache(
    async () => {
      fetches += 1;
      await new Promise<void>((resolve) => setTimeout(resolve, 1));
      return [{ title: "note", content: "body" }];
    },
    OPTIONS,
  );

  await Promise.all([cache.get("session-1"), cache.get("session-1"), cache.get("session-1")]);
  assert.equal(fetches, 1);
});

test("the payload is ordered by relevance", () => {
  const ordered = orderByRelevance([
    { title: "stale", content: "a", createdAt: "2026-01-01T00:00:00Z" },
    { title: "relevant", content: "b", rank: 0.9 },
    { title: "less relevant", content: "c", rank: 0.4 },
  ]);
  assert.deepEqual(
    ordered.map((entry) => entry.title),
    ["relevant", "less relevant", "stale"],
  );
});

test("the injected payload never exceeds the character cap", () => {
  const payload = buildContextPayload(
    [
      { title: "one", content: "x".repeat(200), rank: 3 },
      { title: "two", content: "y".repeat(200), rank: 2 },
    ],
    { maxChars: 120, maxItems: 5 },
  );

  assert.ok(payload.text.length <= 120, `payload was ${payload.text.length} characters`);
  assert.equal(payload.truncated, true);
  assert.match(payload.text, /\[shared context truncated\]/);
});

test("the payload is capped by item count as well as by characters", () => {
  const payload = buildContextPayload(
    [
      { title: "one", content: "a", rank: 3 },
      { title: "two", content: "b", rank: 2 },
      { title: "three", content: "c", rank: 1 },
    ],
    { maxChars: 10_000, maxItems: 2 },
  );

  assert.equal(payload.entries, 2);
  assert.equal(payload.truncated, false);
  assert.match(payload.text, /### one/);
  assert.doesNotMatch(payload.text, /### three/);
});
