import type { ResolvedOptions } from "./config.js";

export type AgentStatus = "active" | "idle" | "blocked" | "offline";
export type HeartbeatStatus = "working" | "idle" | "blocked";

export interface ShakerIdentity {
  readonly configured: boolean;
  readonly project_id?: string;
  readonly project?: { readonly name: string; readonly description: string; readonly status: string };
  readonly agent_id?: string;
  readonly agent?: {
    readonly name: string;
    readonly role: string;
    readonly status: string;
    readonly project_id: string;
    readonly tool_type: string;
  };
  readonly message?: string;
}

export interface ShakerTask {
  readonly id: string;
  readonly project_id: string;
  readonly title: string;
  readonly description: string | null;
  readonly status: string;
  readonly priority: string;
  readonly assigned_to: string | null;
  readonly created_at: string;
  readonly updated_at: string;
}

export interface ShakerTaskList {
  readonly agent_id: string;
  readonly count: number;
  readonly tasks: readonly ShakerTask[];
}

export interface ShakerStatusUpdate {
  readonly success: boolean;
  readonly agent_id: string;
  readonly status: string;
  readonly message: string;
}

export interface ShakerHeartbeat {
  readonly success: boolean;
  readonly agent_id: string;
  readonly status: string;
  readonly lease_ttl_seconds: number;
  readonly next_heartbeat_before: string;
  readonly broadcast: boolean;
}

/** The server's answer to a capability announcement. */
export interface ShakerAnnounceResult {
  readonly success: boolean;
  readonly agent_id: string;
  readonly capabilities: readonly string[];
  readonly host?: string;
  readonly host_version?: string;
  readonly protocol_version?: string;
}

export interface ShakerContextEntry {
  readonly id: string;
  readonly project_id?: string;
  readonly agent_id?: string;
  readonly agent_name?: string;
  readonly title: string;
  readonly content?: string;
  readonly preview?: string;
  readonly format?: string;
  readonly tags?: readonly string[];
  readonly created_at?: string;
  readonly rank?: number;
}

export interface ShakerContextList {
  readonly contexts: readonly ShakerContextEntry[];
  readonly count: number;
  readonly note?: string;
}

export interface ShakerContextSearch {
  readonly contexts: readonly ShakerContextEntry[];
  readonly count: number;
  readonly query: string;
}

export interface ShakerAddContextResult {
  readonly success: boolean;
  readonly id: string;
  readonly title: string;
  readonly agent_id: string;
  readonly agent_name: string;
  readonly tags: readonly string[];
  readonly preview: string;
  readonly format: string;
  readonly created_at: string;
  readonly shared_with: string;
}

export interface ShakerBlocker {
  readonly id: string;
  readonly task_id: string;
  readonly agent_id: string;
  readonly project_id: string;
  readonly blocker_type: string;
  readonly title: string;
  readonly description: string;
  readonly status: string;
  readonly resolution?: string | null;
  readonly created_at: string;
  readonly updated_at: string;
}

export interface ShakerBlockerList {
  readonly agent_id: string;
  readonly status: string;
  readonly count: number;
  readonly blockers: readonly ShakerBlocker[];
}

export interface ShakerDependencySummary {
  readonly task_id: string;
  readonly title?: string;
  readonly status?: string;
}

export interface ShakerDependencyList {
  readonly task_id: string;
  readonly depends_on: readonly ShakerDependencySummary[];
  readonly depends_for: readonly ShakerDependencySummary[];
}

export interface HeartbeatInput {
  readonly status: HeartbeatStatus;
  readonly taskId?: string;
  readonly progressPct?: number;
  readonly metadata?: Readonly<Record<string, unknown>>;
}

/** What this client says it can do. */
export interface AnnounceCapabilityInput {
  readonly capabilities: readonly string[];
  readonly host?: string;
  readonly pluginVersion?: string;
}

export interface AddContextInput {
  readonly title: string;
  readonly content: string;
  readonly tags?: readonly string[];
}

export interface SearchContextsInput {
  readonly query: string;
  readonly projectId: string;
  readonly limit?: number;
}

/**
 * The slice of the coordination plane this plugin depends on.
 *
 * The core is written against this interface rather than the concrete client so
 * that the bridges can be exercised without a server.
 */
export interface ShakerApi {
  getIdentity(): Promise<ShakerIdentity>;
  getMyTasks(status?: string): Promise<ShakerTaskList>;
  updateMyStatus(status: AgentStatus): Promise<ShakerStatusUpdate>;
  sendHeartbeat(input: HeartbeatInput): Promise<ShakerHeartbeat>;
  announceCapability(input: AnnounceCapabilityInput): Promise<ShakerAnnounceResult>;
  addContext(input: AddContextInput): Promise<ShakerAddContextResult>;
  listContexts(projectId?: string): Promise<ShakerContextList>;
  searchContexts(input: SearchContextsInput): Promise<ShakerContextSearch>;
  getMyBlockers(status?: string): Promise<ShakerBlockerList>;
  getTaskDependencies(taskId: string): Promise<ShakerDependencyList>;
}

export class ShakerError extends Error {
  readonly tool: string;

  constructor(tool: string, message: string) {
    super(message);
    this.name = "ShakerError";
    this.tool = tool;
  }
}

/** The coordination server did not answer inside the configured timeout. */
export class ShakerTimeoutError extends ShakerError {
  constructor(tool: string, message: string) {
    super(tool, message);
    this.name = "ShakerTimeoutError";
  }
}

/** The coordination server was unreachable or answered with a transport failure. */
export class ShakerTransportError extends ShakerError {
  constructor(tool: string, message: string) {
    super(tool, message);
    this.name = "ShakerTransportError";
  }
}

export interface HttpResponseLike {
  readonly ok: boolean;
  readonly status: number;
  text(): Promise<string>;
}

export interface FetchInitLike {
  readonly method: string;
  readonly headers: Readonly<Record<string, string>>;
  readonly body: string;
  readonly signal: AbortSignal;
}

export type FetchLike = (url: string, init: FetchInitLike) => Promise<HttpResponseLike>;

export interface ShakerClientOptions {
  readonly endpoint: string;
  readonly apiKey: string;
  readonly projectId?: string;
  readonly agentId?: string;
  readonly timeoutMs?: number;
  readonly fetchImpl?: FetchLike;
}

interface JsonRpcSuccess {
  readonly content?: ReadonlyArray<{ readonly type?: string; readonly text?: string }>;
  readonly isError?: boolean;
}

interface JsonRpcResponse {
  readonly jsonrpc?: string;
  readonly id?: unknown;
  readonly result?: JsonRpcSuccess;
  readonly error?: { readonly code?: number; readonly message?: string; readonly data?: unknown };
}

const defaultFetch: FetchLike = (url, init) => fetch(url, init);

export class ShakerClient implements ShakerApi {
  private readonly endpoint: string;
  private readonly apiKey: string;
  private readonly projectId?: string;
  private readonly agentId?: string;
  private readonly timeoutMs: number;
  private readonly fetchImpl: FetchLike;
  private requestId = 0;

  constructor(options: ShakerClientOptions) {
    this.endpoint = options.endpoint;
    this.apiKey = options.apiKey;
    this.timeoutMs = options.timeoutMs ?? 10_000;
    this.fetchImpl = options.fetchImpl ?? defaultFetch;
    if (options.projectId !== undefined) this.projectId = options.projectId;
    if (options.agentId !== undefined) this.agentId = options.agentId;
  }

  static fromResolved(options: ResolvedOptions, fetchImpl?: FetchLike): ShakerClient {
    return new ShakerClient({
      endpoint: options.endpoint,
      apiKey: options.apiKey,
      timeoutMs: options.requestTimeoutMs,
      ...(options.projectId === undefined ? {} : { projectId: options.projectId }),
      ...(options.agentId === undefined ? {} : { agentId: options.agentId }),
      ...(fetchImpl === undefined ? {} : { fetchImpl }),
    });
  }

  async call<T>(tool: string, args: Record<string, unknown>): Promise<T> {
    const payload = {
      jsonrpc: "2.0",
      id: (this.requestId += 1),
      method: "tools/call",
      params: { name: tool, arguments: args },
    };

    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.timeoutMs);

    let response: HttpResponseLike;
    try {
      response = await this.fetchImpl(this.endpoint, {
        method: "POST",
        headers: this.headers(),
        body: JSON.stringify(payload),
        signal: controller.signal,
      });
    } catch (error) {
      if (controller.signal.aborted) {
        throw new ShakerTimeoutError(tool, `agent-shaker: ${tool} timed out after ${this.timeoutMs}ms`);
      }
      throw new ShakerTransportError(tool, `agent-shaker: ${tool} request failed: ${describe(error)}`);
    } finally {
      clearTimeout(timer);
    }

    if (!response.ok) {
      const body = await safeText(response);
      throw new ShakerTransportError(tool, `agent-shaker: ${tool} failed with HTTP ${response.status}: ${body}`);
    }

    const raw = await safeText(response);
    const parsed = parseJson(raw);
    if (parsed === undefined) {
      throw new ShakerTransportError(tool, `agent-shaker: ${tool} returned a non-JSON response`);
    }

    const envelope = parsed as JsonRpcResponse;
    if (envelope.error) {
      const code = envelope.error.code === undefined ? "" : ` (code ${envelope.error.code})`;
      throw new ShakerError(tool, `agent-shaker: ${tool} failed${code}: ${envelope.error.message ?? "unknown error"}`);
    }

    const result = envelope.result;
    if (!result) {
      throw new ShakerTransportError(tool, `agent-shaker: ${tool} returned no result`);
    }

    const text = result.content?.find((part) => part.type === "text")?.text ?? "";
    if (result.isError === true) {
      throw new ShakerError(tool, `agent-shaker: ${tool} reported an error: ${extractErrorMessage(text)}`);
    }

    const decoded = parseJson(text);
    if (decoded === undefined) {
      throw new ShakerTransportError(tool, `agent-shaker: ${tool} returned an unreadable payload`);
    }
    return decoded as T;
  }

  async getIdentity(): Promise<ShakerIdentity> {
    return this.call<ShakerIdentity>("get_my_identity", {});
  }

  async getMyTasks(status?: string): Promise<ShakerTaskList> {
    const args: Record<string, unknown> = {};
    if (status !== undefined) args["status"] = status;
    return this.call<ShakerTaskList>("get_my_tasks", args);
  }

  async updateMyStatus(status: AgentStatus): Promise<ShakerStatusUpdate> {
    return this.call<ShakerStatusUpdate>("update_my_status", { status });
  }

  async sendHeartbeat(input: HeartbeatInput): Promise<ShakerHeartbeat> {
    const args: Record<string, unknown> = { status: input.status };
    if (input.taskId !== undefined) args["task_id"] = input.taskId;
    if (input.progressPct !== undefined) args["progress_pct"] = input.progressPct;
    if (input.metadata !== undefined) args["metadata"] = input.metadata;
    return this.call<ShakerHeartbeat>("send_heartbeat", args);
  }

  async announceCapability(input: AnnounceCapabilityInput): Promise<ShakerAnnounceResult> {
    const args: Record<string, unknown> = { capabilities: [...input.capabilities] };
    if (input.host !== undefined) args["host"] = input.host;
    if (input.pluginVersion !== undefined) args["plugin_version"] = input.pluginVersion;
    return this.call<ShakerAnnounceResult>("announce_capability", args);
  }

  async addContext(input: AddContextInput): Promise<ShakerAddContextResult> {
    const args: Record<string, unknown> = { title: input.title, content: input.content };
    if (input.tags !== undefined) args["tags"] = [...input.tags];
    if (this.projectId !== undefined) args["project_id"] = this.projectId;
    return this.call<ShakerAddContextResult>("add_context", args);
  }

  async listContexts(projectId?: string): Promise<ShakerContextList> {
    const args: Record<string, unknown> = {};
    const scope = projectId ?? this.projectId;
    if (scope !== undefined) args["project_id"] = scope;
    return this.call<ShakerContextList>("list_contexts", args);
  }

  async searchContexts(input: SearchContextsInput): Promise<ShakerContextSearch> {
    const args: Record<string, unknown> = { query: input.query, project_id: input.projectId };
    if (input.limit !== undefined) args["limit"] = input.limit;
    return this.call<ShakerContextSearch>("search_contexts", args);
  }

  async getMyBlockers(status?: string): Promise<ShakerBlockerList> {
    const args: Record<string, unknown> = {};
    if (status !== undefined) args["status"] = status;
    return this.call<ShakerBlockerList>("get_my_blockers", args);
  }

  async getTaskDependencies(taskId: string): Promise<ShakerDependencyList> {
    return this.call<ShakerDependencyList>("get_task_dependencies", { task_id: taskId });
  }

  private headers(): Record<string, string> {
    const headers: Record<string, string> = {
      "content-type": "application/json",
      accept: "application/json",
      // The server accepts the key in either header; X-API-Key is the one it
      // documents first and the one every HTTP client can set without ceremony.
      "X-API-Key": this.apiKey,
    };
    if (this.projectId !== undefined) headers["X-Project-ID"] = this.projectId;
    if (this.agentId !== undefined) headers["X-Agent-ID"] = this.agentId;
    return headers;
  }
}

function describe(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  return "unknown error";
}

async function safeText(response: HttpResponseLike): Promise<string> {
  try {
    return await response.text();
  } catch {
    return "";
  }
}

function parseJson(raw: string): unknown {
  if (raw.trim().length === 0) return undefined;
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return undefined;
  }
}

function extractErrorMessage(text: string): string {
  const parsed = parseJson(text);
  if (parsed !== undefined && typeof parsed === "object" && parsed !== null) {
    const error = (parsed as { readonly error?: unknown }).error;
    if (typeof error === "string") return error;
  }
  return text.length > 0 ? text : "no detail";
}
