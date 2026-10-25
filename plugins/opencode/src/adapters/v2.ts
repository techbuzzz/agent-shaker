import { Plugin } from "@opencode/plugin";
import { resolveOptions } from "../core/config.js";
import { announceCapabilities, serverCapabilities } from "../core/announce.js";
import {
  CAPABILITY_SESSION_HOOK,
  CAPABILITY_TOOL_HOOK,
  type CapabilityReport,
  assertLifecycleCapability,
  describeMissing,
  hasCapability,
  probeCapabilities,
} from "../core/capabilities.js";
import {
  buildCompactionMetadata,
  buildCompactionSummary,
  collectSnapshot,
  persistDigest,
  resolveCompactionMode,
} from "../core/compaction.js";
import { captureEvidence, extractCommand, isTestOrBuildCommand } from "../core/evidence.js";
import { errorMessage, type Logger } from "../core/logger.js";
import { type RuntimeDeps, createRuntime } from "../core/runtime.js";
import { PLUGIN_VERSION } from "../version.js";
import type { SessionEvent } from "../core/lifecycle.js";

export type V2Context = Plugin.Context;

interface Registration {
  readonly dispose: () => Promise<void>;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readEventData(event: { readonly type: string }): Record<string, unknown> {
  const candidate = (event as { readonly data?: unknown }).data;
  return isRecord(candidate) ? candidate : {};
}

/**
 * Normalises a host event into the small shape the lifecycle bridge knows.
 *
 * The v2 event union is wide and versioned; the bridge only cares about the
 * session it belongs to, so the conversion happens once, here, instead of in
 * every call site.
 */
function toSessionEvent(event: { readonly type: string }): SessionEvent {
  const data = readEventData(event);
  const sessionId = typeof data["sessionID"] === "string" ? data["sessionID"] : undefined;
  const projectId = typeof data["projectID"] === "string" ? data["projectID"] : undefined;
  const rawError = data["error"];
  const message = isRecord(rawError) ? errorMessage(rawError) : undefined;

  return {
    type: event.type,
    ...(sessionId === undefined ? {} : { sessionId }),
    ...(projectId === undefined ? {} : { projectId }),
    ...(message === undefined ? {} : { error: message }),
  };
}

const COMPACTION_EVENTS = new Set([
  "session.compaction.started",
  "session.compaction.ended",
  "session.compaction.failed",
]);

/**
 * Registers the plugin against the v2 host API.
 *
 * The hook set is deliberately narrow: session lifecycle for the lease, the
 * three model-request hooks for context and compaction, and the two tool hooks
 * for the gate and the evidence trail.
 */
export async function setupV2(ctx: V2Context, deps: RuntimeDeps = {}): Promise<() => Promise<void>> {
  const options = resolveOptions(ctx.options, deps.env ?? process.env);
  const runtime = createRuntime(options, deps);
  const report = probeCapabilities(ctx);

  // The event stream is the lease. Without it there is no plugin, only a
  // process that believes it is connected to a coordination server.
  assertLifecycleCapability(report);
  warnAboutMissingCapabilities(report, runtime.logger);

  // Tell the server what this client can do for it, once, before the first
  // heartbeat. The capability list is what the host surface and the plugin's
  // own configuration together allow: a capability the client cannot honour
  // would send the server looking for a feature that is not there.
  await announceCapabilities(
    runtime.client,
    serverCapabilities(
      {
        sessionHook: hasCapability(report, CAPABILITY_SESSION_HOOK),
        toolHook: hasCapability(report, CAPABILITY_TOOL_HOOK),
      },
      {
        policyEnabled: options.policy.enabled,
        contextEnabled: options.context.enabled,
        evidenceEnabled: options.evidence.enabled,
      },
    ),
    "opencode",
    PLUGIN_VERSION,
    runtime.logger,
  );

  const registrations: Registration[] = [];
  let compactionAuthored = false;
  let compactionObserved = false;
  let reportedMode: string | undefined;

  if (report.present.includes(CAPABILITY_SESSION_HOOK)) {
    registrations.push(
      await ctx.session.hook("prompt", (input) => {
        runtime.cache.beginPrompt(String(input.sessionID));
      }),
    );

    // The same injection runs on all three model-request hooks on purpose:
    // registering it on one of them makes the compaction request differ from
    // every normal turn and busts the prompt-cache prefix at the most
    // expensive request of the session's life.
    const inject = async (sessionId: string, system: Array<{ type: "text"; text: string }>): Promise<void> => {
      if (!options.context.enabled) return;
      const payload = await runtime.cache.get(sessionId);
      if (payload.text.length === 0) return;
      system.push({ type: "text", text: payload.text });
    };

    registrations.push(
      await ctx.session.hook("context", async (input) => {
        await inject(String(input.sessionID), input.system);
      }),
    );

    registrations.push(
      await ctx.session.hook("generate", async (input) => {
        await inject(String(input.sessionID), input.system);
      }),
    );

    registrations.push(
      await ctx.session.hook("compaction", async (input) => {
        const sessionId = String(input.sessionID);
        await inject(sessionId, input.system);

        const projectId = await runtime.projectId();
        const snapshot = await collectSnapshot(
          runtime.client,
          {
            sessionId,
            ...(projectId === undefined ? {} : { projectId }),
            maxContextItems: options.context.maxItems,
            ...(options.context.query === undefined ? {} : { contextQuery: options.context.query }),
          },
          runtime.logger,
        );
        const summary = buildCompactionSummary(snapshot);

        // Setting `result` is what makes the compaction the plugin's own: the
        // host records it and skips the model request.
        input.result = {
          summary,
          metadata: buildCompactionMetadata(snapshot),
        };
        compactionAuthored = true;

        try {
          await persistDigest(runtime.client, snapshot, summary);
        } catch (error) {
          runtime.logger.warn("agent-shaker: compaction digest was not stored", {
            error: errorMessage(error),
          });
        }
      }),
    );
  }

  if (report.present.includes(CAPABILITY_TOOL_HOOK)) {
    registrations.push(
      await ctx.tool.hook("execute.before", async (input) => {
        const decision = await runtime.policy.evaluate(input.tool);
        if (decision.allowed) return;
        throw new Error(decision.reason);
      }),
    );

    registrations.push(
      await ctx.tool.hook("execute.after", async (input) => {
        if (!options.evidence.enabled) return;
        const command = extractCommand(input.tool, input.input);
        if (!isTestOrBuildCommand(input.tool, command)) return;
        await captureEvidence(
          runtime.client,
          readToolOutcome(input, input.tool, String(input.sessionID), command),
          runtime.logger,
        );
      }),
    );
  }

  const stream = ctx.event.subscribe();
  void (async () => {
    try {
      for await (const event of stream) {
        await runtime.bridge.handleEvent(toSessionEvent(event));
        if (COMPACTION_EVENTS.has(event.type)) {
          compactionObserved = true;
        }
        const mode = resolveCompactionMode(compactionAuthored, compactionObserved);
        if (mode === reportedMode) continue;
        reportedMode = mode;
        runtime.logger.info(`agent-shaker: compaction mode is ${mode}`);
      }
    } catch (error) {
      runtime.logger.error("agent-shaker: the host event stream ended", {
        error: errorMessage(error),
      });
    }
  })();

  runtime.logger.info("agent-shaker: plugin loaded", {
    host: "v2",
    endpoint: options.endpoint,
    project: options.projectId ?? "resolved from credential",
    policy: options.policy.enabled ? "on" : "off",
    policy_fail_closed: options.policy.failClosed,
  });

  return async () => {
    runtime.bridge.stop();
    for (const registration of registrations) {
      await registration.dispose();
    }
  };
}

function warnAboutMissingCapabilities(report: CapabilityReport, logger: Logger): void {
  if (report.missing.length === 0) return;
  logger.warnOnce(
    "capabilities",
    `agent-shaker: this host does not expose ${describeMissing(report)}; the matching features are disabled`,
  );
}

interface ToolOutcome {
  readonly tool: string;
  readonly status: "completed" | "error";
  readonly output?: string;
  readonly errorMessage?: string;
  readonly sessionId: string;
  readonly command?: string;
}

/**
 * Reads the outcome of a tool call.
 *
 * The hook's payload is an intersection with a discriminated union, so the
 * narrow is done here once rather than at every call site.
 */
function readToolOutcome(
  input: { readonly status: string; readonly result?: unknown; readonly error?: unknown },
  tool: string,
  sessionId: string,
  command: string | undefined,
): ToolOutcome {
  if (input.status === "error") {
    const error = input.error;
    const message = isRecord(error) && typeof error["message"] === "string" ? error["message"] : undefined;
    return {
      tool,
      status: "error",
      ...(message === undefined ? {} : { errorMessage: message }),
      sessionId,
      ...(command === undefined ? {} : { command }),
    };
  }
  const result = input.result;
  const output = isRecord(result) && typeof result["output"] === "string" ? result["output"] : undefined;
  return {
    tool,
    status: "completed",
    ...(output === undefined ? {} : { output }),
    sessionId,
    ...(command === undefined ? {} : { command }),
  };
}
