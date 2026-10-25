import type { Event } from "@opencode-ai/sdk";
import type { Hooks, PluginInput, PluginOptions } from "@opencode-ai/plugin";
import { resolveOptions } from "../core/config.js";
import { announceCapabilities, serverCapabilities } from "../core/announce.js";
import {
  buildCompactionMetadata,
  buildCompactionSummary,
  collectSnapshot,
  persistDigest,
} from "../core/compaction.js";
import { captureEvidence, extractCommand, isTestOrBuildCommand } from "../core/evidence.js";
import { errorMessage } from "../core/logger.js";
import type { SessionEvent } from "../core/lifecycle.js";
import { type RuntimeDeps, createRuntime } from "../core/runtime.js";
import { PLUGIN_VERSION } from "../version.js";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function errorText(error: unknown): string | undefined {
  if (error === undefined || error === null) return undefined;
  if (isRecord(error) && typeof error["message"] === "string") return error["message"];
  return errorMessage(error);
}

/**
 * Normalises a v1 host event.
 *
 * The v1 event union carries the session id in different places depending on
 * the event, so the conversion is spelled out once here.
 */
function toSessionEvent(event: Event): SessionEvent | null {
  switch (event.type) {
    case "session.created":
      return {
        type: event.type,
        sessionId: event.properties.info.id,
        projectId: event.properties.info.projectID,
      };
    case "session.deleted":
      return { type: event.type, sessionId: event.properties.info.id };
    case "session.idle":
      return { type: event.type, sessionId: event.properties.sessionID };
    case "session.compacted":
      return { type: event.type, sessionId: event.properties.sessionID };
    case "session.error":
      return {
        type: event.type,
        ...(event.properties.sessionID === undefined ? {} : { sessionId: event.properties.sessionID }),
        ...(errorText(event.properties.error) === undefined
          ? {}
          : { error: errorText(event.properties.error) as string }),
      };
    default:
      return null;
  }
}

/**
 * Classifies the outcome of a v1 tool call.
 *
 * The v1 hook reports a title and an output but no status, so the verdict has
 * to be read from what the host wrote.
 */
function classifyOutcome(title: string, output: string): { status: "completed" | "error"; errorMessage?: string } {
  const trimmedTitle = title.trim();
  if (/^error\b/i.test(trimmedTitle)) {
    return { status: "error", errorMessage: trimmedTitle };
  }
  const firstLine = (output.split("\n")[0] ?? "").trim();
  if (/^error\b/i.test(firstLine)) {
    return { status: "error", errorMessage: firstLine };
  }
  return { status: "completed" };
}

/**
 * Registers the plugin against the v1 host API.
 *
 * The v1 host has no compaction hook that can replace the summary, so the
 * plugin influences the host's own summary and stores the same digest in the
 * coordination plane.
 */
export async function createV1Hooks(
  _input: PluginInput,
  options?: PluginOptions,
  deps: RuntimeDeps = {},
): Promise<Hooks> {
  const resolved = resolveOptions(options ?? {}, deps.env ?? process.env);
  const runtime = createRuntime(resolved, deps);
  const logger = runtime.logger;

  // Tell the server what this client can do for it, once, before the first
  // heartbeat. The v1 host always exposes the compaction hook and the two tool
  // hooks, so the surfaces are known; the configuration decides the rest.
  await announceCapabilities(
    runtime.client,
    serverCapabilities(
      { sessionHook: true, toolHook: true },
      {
        policyEnabled: resolved.policy.enabled,
        contextEnabled: resolved.context.enabled,
        evidenceEnabled: resolved.evidence.enabled,
      },
    ),
    "opencode",
    PLUGIN_VERSION,
    logger,
  );

  const inject = async (sessionId: string | undefined, system: string[]): Promise<void> => {
    if (!resolved.context.enabled) return;
    const key = sessionId ?? "";
    const payload = await runtime.cache.get(key);
    if (payload.text.length === 0) return;
    system.push(payload.text);
  };

  const hooks: Hooks = {
    event: async ({ event }) => {
      const sessionEvent = toSessionEvent(event);
      if (!sessionEvent) return;
      await runtime.bridge.handleEvent(sessionEvent);
    },

    "chat.message": async (input) => {
      runtime.cache.beginPrompt(input.sessionID);
    },

    "experimental.chat.system.transform": async (input, output) => {
      await inject(input.sessionID, output.system);
    },

    "experimental.session.compacting": async (input, output) => {
      const sessionId = input.sessionID;
      const projectId = await runtime.projectId();
      const snapshot = await collectSnapshot(
        runtime.client,
        {
          sessionId,
          ...(projectId === undefined ? {} : { projectId }),
          maxContextItems: resolved.context.maxItems,
          ...(resolved.context.query === undefined ? {} : { contextQuery: resolved.context.query }),
        },
        logger,
      );
      const summary = buildCompactionSummary(snapshot);
      const metadata = buildCompactionMetadata(snapshot);

      // The host owns the summary on this API, so the plugin contributes the
      // coordination facts to the prompt that produces it.
      output.context.push(summary, `Coordination identifiers: ${JSON.stringify(metadata)}`);

      try {
        await persistDigest(runtime.client, snapshot, summary);
      } catch (error) {
        logger.warn("agent-shaker: compaction digest was not stored", { error: errorMessage(error) });
      }
    },

    "tool.execute.before": async (input) => {
      const decision = await runtime.policy.evaluate(input.tool);
      if (decision.allowed) return;
      throw new Error(decision.reason);
    },

    "tool.execute.after": async (input, output) => {
      if (!resolved.evidence.enabled) return;
      const command = extractCommand(input.tool, input.args);
      if (!isTestOrBuildCommand(input.tool, command)) return;
      const outcome = classifyOutcome(output.title, output.output);
      await captureEvidence(
        runtime.client,
        {
          tool: input.tool,
          status: outcome.status,
          sessionId: input.sessionID,
          ...(command === undefined ? {} : { command }),
          ...(outcome.errorMessage === undefined ? {} : { errorMessage: outcome.errorMessage }),
          output: output.output,
        },
        logger,
      );
    },

    dispose: async () => {
      runtime.bridge.stop();
    },
  };

  logger.info("agent-shaker: plugin loaded", {
    host: "v1",
    endpoint: resolved.endpoint,
    project: resolved.projectId ?? "resolved from credential",
    policy: resolved.policy.enabled ? "on" : "off",
    policy_fail_closed: resolved.policy.failClosed,
  });

  return hooks;
}
