import type { AgentStatus, HeartbeatStatus, ShakerApi } from "./client.js";
import type { ResolvedHeartbeatOptions } from "./config.js";
import { errorMessage, type Logger } from "./logger.js";

export interface SessionEvent {
  readonly type: string;
  readonly sessionId?: string;
  readonly projectId?: string;
  readonly error?: string;
}

export interface LifecycleCall {
  readonly action: "heartbeat" | "status";
  readonly agentStatus?: AgentStatus;
  readonly heartbeatStatus?: HeartbeatStatus;
}

/**
 * The event to call mapping.
 *
 * One table, shared by both host adapters: the v1 and v2 event names differ,
 * but the coordination reaction must not, or the same session would behave
 * differently depending on which host loaded the plugin.
 */
export const EVENT_CALLS: Readonly<Record<string, readonly LifecycleCall[]>> = {
  // The agent becomes live and takes a lease.
  "session.created": [{ action: "heartbeat", heartbeatStatus: "working" }],
  // The lease is still held; the agent is simply not doing anything right now.
  "session.idle": [
    { action: "status", agentStatus: "idle" },
    { action: "heartbeat", heartbeatStatus: "idle" },
  ],
  // Work stopped on something the agent cannot resolve alone.
  "session.error": [{ action: "status", agentStatus: "blocked" }],
  // The v2 host reports a failed run instead of a session error.
  "session.execution.failed": [{ action: "status", agentStatus: "blocked" }],
  // The session is gone: drop the lease instead of waiting for the reaper.
  "session.deleted": [{ action: "status", agentStatus: "offline" }],
};

export interface LifecycleTimers {
  setTimeout(handler: () => void, delayMs: number): unknown;
  clearTimeout(handle: unknown): void;
}

export const systemTimers: LifecycleTimers = {
  setTimeout: (handler, delayMs) => setTimeout(handler, delayMs),
  clearTimeout: (handle) => clearTimeout(handle as ReturnType<typeof setTimeout>),
};

/**
 * The delay until the next heartbeat, derived from the lease the server granted.
 *
 * The server hands back `lease_ttl_seconds`; a beat scheduled from anything
 * other than that number is a guess, and a guess that runs late loses the lease
 * and takes the agent's tasks away from it. The fraction leaves room for one
 * lost beat without turning every lease into three requests.
 */
export function nextHeartbeatDelayMs(
  leaseTtlSeconds: number,
  options: Pick<ResolvedHeartbeatOptions, "leaseFraction" | "minIntervalMs" | "maxIntervalMs">,
): number {
  const leaseMs = Math.max(1, Math.floor(leaseTtlSeconds * 1000));
  const target = Math.floor(leaseMs * options.leaseFraction);
  const upper = Math.min(options.maxIntervalMs, leaseMs);
  const lower = Math.min(options.minIntervalMs, upper);
  return Math.max(lower, Math.min(target, upper));
}

export interface LifecycleBridgeOptions {
  readonly heartbeat: ResolvedHeartbeatOptions;
  readonly timers?: LifecycleTimers;
  readonly logger?: Logger;
}

export class LifecycleBridge {
  private readonly client: ShakerApi;
  private readonly options: LifecycleBridgeOptions;
  private readonly timers: LifecycleTimers;
  private readonly logger: Logger;
  private timer: unknown = null;
  private stopped = false;

  constructor(client: ShakerApi, options: LifecycleBridgeOptions, logger: Logger) {
    this.client = client;
    this.options = options;
    this.timers = options.timers ?? systemTimers;
    this.logger = logger;
  }

  /**
   * Applies the coordination reaction for one session event.
   *
   * Never throws: a coordination outage must not take the coding session down
   * with it. The calls that were attempted are returned so the behaviour can be
   * asserted without a server.
   */
  async handleEvent(event: SessionEvent): Promise<readonly LifecycleCall[]> {
    const calls = EVENT_CALLS[event.type];
    if (!calls) return [];

    if (event.type === "session.created") {
      // A new session takes the lease again after the previous one was deleted.
      this.stopped = false;
    }
    if (this.stopped) return [];

    const performed: LifecycleCall[] = [];
    for (const call of calls) {
      performed.push(call);
      await this.perform(call);
    }
    return performed;
  }

  /** Stops the heartbeat. Called when the host tears the plugin down. */
  stop(): void {
    this.stopped = true;
    this.cancelTimer();
  }

  private async perform(call: LifecycleCall): Promise<void> {
    if (call.action === "status") {
      const status = call.agentStatus;
      if (!status) return;
      try {
        await this.client.updateMyStatus(status);
      } catch (error) {
        this.logger.warn("agent-shaker: status update failed", { status, error: errorMessage(error) });
      }
      if (status === "offline") {
        this.stopped = true;
        this.cancelTimer();
      }
      return;
    }

    const heartbeatStatus = call.heartbeatStatus ?? "working";
    try {
      const result = await this.client.sendHeartbeat({ status: heartbeatStatus });
      this.scheduleFrom(result.lease_ttl_seconds);
    } catch (error) {
      this.logger.warn("agent-shaker: heartbeat failed", { status: heartbeatStatus, error: errorMessage(error) });
      this.scheduleRetry();
    }
  }

  private scheduleFrom(leaseTtlSeconds: number): void {
    if (this.stopped) return;
    const delay = nextHeartbeatDelayMs(leaseTtlSeconds, this.options.heartbeat);
    this.cancelTimer();
    this.timer = this.timers.setTimeout(() => {
      void this.perform({ action: "heartbeat", heartbeatStatus: "working" });
    }, delay);
  }

  /**
   * Retries soon after a failed beat.
   *
   * The lease clock is unknown at that point, so the only safe assumption is
   * that it is running out: the server takes the agent offline when the lease
   * expires, and a recovered connection should restore the lease before that.
   */
  private scheduleRetry(): void {
    if (this.stopped) return;
    this.cancelTimer();
    this.timer = this.timers.setTimeout(() => {
      void this.perform({ action: "heartbeat", heartbeatStatus: "working" });
    }, this.options.heartbeat.minIntervalMs);
  }

  private cancelTimer(): void {
    if (this.timer === null) return;
    this.timers.clearTimeout(this.timer);
    this.timer = null;
  }
}
