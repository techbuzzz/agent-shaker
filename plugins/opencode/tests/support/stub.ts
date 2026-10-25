import type {
  AddContextInput,
  AgentStatus,
  AnnounceCapabilityInput,
  HeartbeatInput,
  SearchContextsInput,
  ShakerAddContextResult,
  ShakerAnnounceResult,
  ShakerApi,
  ShakerBlockerList,
  ShakerContextList,
  ShakerContextSearch,
  ShakerDependencyList,
  ShakerHeartbeat,
  ShakerIdentity,
  ShakerStatusUpdate,
  ShakerTask,
  ShakerTaskList,
} from "../../src/core/client.js";

export interface RecordedCall {
  readonly name: string;
  readonly args: Record<string, unknown>;
}

export interface StubOptions {
  readonly identity?: ShakerIdentity;
  readonly tasks?: ShakerTask[];
  readonly heartbeat?: Partial<ShakerHeartbeat>;
  readonly contexts?: ShakerContextList["contexts"];
  readonly searchResults?: ShakerContextSearch["contexts"];
  readonly blockers?: ShakerBlockerList["blockers"];
  readonly dependencies?: ShakerDependencyList;
  /** Method name to the error it should raise. */
  readonly failOn?: Readonly<Record<string, Error>>;
}

export function makeTask(overrides: Partial<ShakerTask> = {}): ShakerTask {
  return {
    id: "task-1",
    project_id: "project-1",
    title: "Ship the plugin",
    description: "Deliver the OpenCode plugin",
    status: "in_progress",
    priority: "high",
    assigned_to: "agent-1",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

/**
 * A coordination server that never runs.
 *
 * The core is written against the ShakerApi interface precisely so that the
 * bridges can be exercised without a live server.
 */
export class StubShaker implements ShakerApi {
  readonly calls: RecordedCall[] = [];
  private readonly options: StubOptions;

  constructor(options: StubOptions = {}) {
    this.options = options;
  }

  private record(name: string, args: Record<string, unknown>): void {
    this.calls.push({ name, args });
    const failure = this.options.failOn?.[name];
    if (failure) throw failure;
  }

  names(): string[] {
    return this.calls.map((call) => call.name);
  }

  callsTo(name: string): RecordedCall[] {
    return this.calls.filter((call) => call.name === name);
  }

  async getIdentity(): Promise<ShakerIdentity> {
    this.record("get_my_identity", {});
    return (
      this.options.identity ?? {
        configured: true,
        project_id: "project-1",
        agent_id: "agent-1",
        agent: {
          name: "shaker-agent",
          role: "engineer",
          status: "active",
          project_id: "project-1",
          tool_type: "opencode",
        },
      }
    );
  }

  async getMyTasks(status?: string): Promise<ShakerTaskList> {
    this.record("get_my_tasks", status === undefined ? {} : { status });
    const tasks = this.options.tasks ?? [];
    return { agent_id: "agent-1", count: tasks.length, tasks };
  }

  async updateMyStatus(status: AgentStatus): Promise<ShakerStatusUpdate> {
    this.record("update_my_status", { status });
    return { success: true, agent_id: "agent-1", status, message: "updated" };
  }

  async sendHeartbeat(input: HeartbeatInput): Promise<ShakerHeartbeat> {
    this.record("send_heartbeat", { status: input.status });
    return {
      success: true,
      agent_id: "agent-1",
      status: input.status,
      lease_ttl_seconds: 60,
      next_heartbeat_before: "2026-01-01T00:01:00Z",
      broadcast: true,
      ...this.options.heartbeat,
    };
  }

  async announceCapability(input: AnnounceCapabilityInput): Promise<ShakerAnnounceResult> {
    this.record("announce_capability", {
      capabilities: input.capabilities,
      host: input.host ?? "",
      plugin_version: input.pluginVersion ?? "",
    });
    return {
      success: true,
      agent_id: "agent-1",
      capabilities: input.capabilities,
      host: input.host,
      protocol_version: "2024-11-05",
    };
  }

  async addContext(input: AddContextInput): Promise<ShakerAddContextResult> {
    this.record("add_context", { title: input.title, content: input.content, tags: input.tags });
    return {
      success: true,
      id: "context-1",
      title: input.title,
      agent_id: "agent-1",
      agent_name: "shaker-agent",
      tags: input.tags ?? [],
      preview: input.content.slice(0, 32),
      format: "markdown",
      created_at: "2026-01-01T00:00:00Z",
      shared_with: "All agents in the project can now read this context",
    };
  }

  async listContexts(projectId?: string): Promise<ShakerContextList> {
    this.record("list_contexts", projectId === undefined ? {} : { project_id: projectId });
    const contexts = this.options.contexts ?? [];
    return { contexts, count: contexts.length };
  }

  async searchContexts(input: SearchContextsInput): Promise<ShakerContextSearch> {
    this.record("search_contexts", { query: input.query, project_id: input.projectId });
    const contexts = this.options.searchResults ?? [];
    return { contexts, count: contexts.length, query: input.query };
  }

  async getMyBlockers(status?: string): Promise<ShakerBlockerList> {
    this.record("get_my_blockers", status === undefined ? {} : { status });
    const blockers = this.options.blockers ?? [];
    return { agent_id: "agent-1", status: status ?? "", count: blockers.length, blockers };
  }

  async getTaskDependencies(taskId: string): Promise<ShakerDependencyList> {
    this.record("get_task_dependencies", { task_id: taskId });
    return this.options.dependencies ?? { task_id: taskId, depends_on: [], depends_for: [] };
  }
}

export interface FakeTimer {
  readonly delayMs: number;
  readonly handler: () => void;
  cancelled: boolean;
}

/** Timers the tests can inspect and fire by hand. */
export class FakeTimers {
  readonly scheduled: FakeTimer[] = [];

  setTimeout = (handler: () => void, delayMs: number): unknown => {
    const timer: { delayMs: number; handler: () => void; cancelled: boolean } = {
      delayMs,
      handler,
      cancelled: false,
    };
    this.scheduled.push(timer);
    return timer;
  };

  clearTimeout = (handle: unknown): void => {
    if (handle && typeof handle === "object" && "cancelled" in handle) {
      (handle as FakeTimer).cancelled = true;
    }
  };

  get last(): FakeTimer | undefined {
    return this.scheduled[this.scheduled.length - 1];
  }
}
