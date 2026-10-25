import type { PluginInput } from "@opencode-ai/plugin";
import type { V2Context } from "../../src/adapters/v2.js";

export type HostHook = (input: never) => Promise<void> | void;

export interface HostEvent {
  readonly type: string;
  readonly data?: unknown;
}

interface StreamState {
  readonly queue: HostEvent[];
  notify?: () => void;
  closed: boolean;
}

function createStream(): { iterable: AsyncIterable<HostEvent>; push: (event: HostEvent) => void; close: () => void } {
  const state: StreamState = { queue: [], closed: false };

  const iterable: AsyncIterable<HostEvent> = {
    async *[Symbol.asyncIterator](): AsyncIterator<HostEvent> {
      for (;;) {
        const next = state.queue.shift();
        if (next !== undefined) {
          yield next;
          continue;
        }
        if (state.closed) return;
        await new Promise<void>((resolve) => {
          state.notify = resolve;
        });
        state.notify = undefined;
      }
    },
  };

  return {
    iterable,
    push: (event) => {
      state.queue.push(event);
      state.notify?.();
    },
    close: () => {
      state.closed = true;
      state.notify?.();
    },
  };
}

export interface FakeV2Host {
  readonly ctx: V2Context;
  readonly sessionHooks: Map<string, HostHook>;
  readonly toolHooks: Map<string, HostHook>;
  readonly emit: (event: HostEvent) => Promise<void>;
}

/**
 * A v2 host reduced to the three domains the plugin uses.
 *
 * The real context is a large interface; the plugin only touches the session
 * hooks, the tool hooks and the event stream, so the fake provides exactly
 * those and records what was registered.
 */
export function createFakeV2Host(options: Record<string, unknown> = {}): FakeV2Host {
  const sessionHooks = new Map<string, HostHook>();
  const toolHooks = new Map<string, HostHook>();
  const stream = createStream();

  const ctx = {
    options,
    session: {
      hook: async (name: string, callback: HostHook) => {
        sessionHooks.set(name, callback);
        return {
          dispose: async () => {
            sessionHooks.delete(name);
          },
        };
      },
    },
    tool: {
      hook: async (name: string, callback: HostHook) => {
        toolHooks.set(name, callback);
        return {
          dispose: async () => {
            toolHooks.delete(name);
          },
        };
      },
    },
    event: {
      subscribe: () => stream.iterable,
    },
  };

  return {
    ctx: ctx as unknown as V2Context,
    sessionHooks,
    toolHooks,
    emit: async (event) => {
      stream.push(event);
      // The adapter consumes the stream in a background task; yielding to the
      // event loop lets the consumer pick the event up.
      await new Promise<void>((resolve) => setTimeout(resolve, 0));
    },
  };
}

export function fakeV1Input(): PluginInput {
  return {
    directory: "/tmp/project",
    worktree: "/tmp/project",
    serverUrl: new URL("http://localhost:4096"),
  } as unknown as PluginInput;
}
