import type { ShakerApi } from "./client.js";
import type { ResolvedPolicyOptions } from "./config.js";
import { errorMessage, type Logger } from "./logger.js";

export type PolicyDecision =
  | { readonly allowed: true; readonly reason: string }
  | { readonly allowed: false; readonly reason: string };

/**
 * The local gate in front of mutating tools.
 *
 * The rule is simple on purpose: an agent that has not claimed a task has no
 * business changing the project. The gate is opt-in because a project that does
 * not use the coordination plane would otherwise block every tool call.
 */
export class PolicyGate {
  private readonly client: ShakerApi;
  private readonly options: ResolvedPolicyOptions;
  private readonly logger: Logger;

  constructor(client: ShakerApi, options: ResolvedPolicyOptions, logger: Logger) {
    this.client = client;
    this.options = options;
    this.logger = logger;
  }

  async evaluate(tool: string): Promise<PolicyDecision> {
    if (!this.options.enabled) {
      return { allowed: true, reason: "policy gate is disabled for this project" };
    }

    if (this.options.readOnlyTools.has(tool.toLowerCase())) {
      return { allowed: true, reason: `${tool} is a read-only tool` };
    }

    let taskCount: number;
    try {
      const tasks = await this.client.getMyTasks();
      taskCount = tasks.tasks.filter((task) => CLAIMED_TASK_STATUSES.has(task.status)).length;
    } catch (error) {
      return this.onServerFailure(tool, error);
    }

    if (taskCount > 0) {
      return { allowed: true, reason: `the agent holds ${taskCount} claimed task(s)` };
    }

    return {
      allowed: false,
      reason:
        `agent-shaker: ${tool} is blocked because the agent has no claimed task. ` +
        "Claim a task on the coordination server before changing the project.",
    };
  }

  /**
   * Decides what an unreachable coordination server means.
   *
   * The default is to fail open: the gate exists to keep a coordination plane
   * honest, and a server outage must not take the whole coding session with
   * it. A project that cannot accept that risk says so explicitly with
   * `fail_closed`, because a gate that fails open by accident is a gate that
   * does not gate.
   */
  private onServerFailure(tool: string, error: unknown): PolicyDecision {
    const message = errorMessage(error);
    if (this.options.failClosed) {
      this.logger.error("agent-shaker: policy gate failed closed", { tool, error: message });
      return {
        allowed: false,
        reason:
          `agent-shaker: ${tool} is blocked because the coordination server could not be reached ` +
          `and this project is configured to fail closed: ${message}`,
      };
    }
    this.logger.warn("agent-shaker: policy gate failed open", { tool, error: message });
    return {
      allowed: true,
      reason: `agent-shaker: the coordination server could not be reached (${message}); failing open`,
    };
  }
}

const CLAIMED_TASK_STATUSES = new Set(["pending", "ready", "in_progress", "blocked"]);
