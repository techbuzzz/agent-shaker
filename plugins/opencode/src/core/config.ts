export type EnvLike = Readonly<Record<string, string | undefined>>;

/**
 * Plugin options as written in the host configuration.
 *
 * Keys are snake_case because the host hands the options through verbatim and
 * the same file has to read well next to the Agent Shaker MCP tool names, which
 * are snake_case on the wire.
 */
export interface AgentShakerOptions {
  /** Absolute URL of the Agent Shaker MCP endpoint, e.g. `http://localhost:8080/mcp`. */
  endpoint?: string;
  /** API key issued by the coordination server; it is bound to exactly one agent. */
  api_key?: string;
  /** Optional project id. The server resolves the agent from the credential. */
  project_id?: string;
  /** Optional agent id. The server rejects it when it disagrees with the credential. */
  agent_id?: string;
  /** Per-request timeout for MCP calls. Defaults to 10000. */
  request_timeout_ms?: number;
  heartbeat?: {
    /** Fraction of the server lease after which the next beat is sent. Defaults to 0.6. */
    lease_fraction?: number;
    /** Lower bound for the beat interval, in milliseconds. Defaults to 5000. */
    min_interval_ms?: number;
    /** Upper bound for the beat interval, in milliseconds. Defaults to 300000. */
    max_interval_ms?: number;
  };
  policy?: {
    /** Turns the local gate on. Off by default: the gate is opt-in per project. */
    enabled?: boolean;
    /** Blocks mutating tools when the coordination server cannot be reached. */
    fail_closed?: boolean;
    /** Tool names that are allowed without a claimed task. */
    read_only_tools?: string[];
  };
  context?: {
    /** Turns shared-context re-injection on. On by default. */
    enabled?: boolean;
    /** Hard character cap for the injected payload. Defaults to 4000. */
    max_chars?: number;
    /** Maximum number of shared contexts considered. Defaults to 8. */
    max_items?: number;
    /** Full-text query used against `search_contexts` instead of `list_contexts`. */
    query?: string;
  };
  evidence?: {
    /** Turns test/build evidence capture on. On by default. */
    enabled?: boolean;
  };
}

export interface ResolvedHeartbeatOptions {
  readonly leaseFraction: number;
  readonly minIntervalMs: number;
  readonly maxIntervalMs: number;
}

export interface ResolvedPolicyOptions {
  readonly enabled: boolean;
  readonly failClosed: boolean;
  readonly readOnlyTools: ReadonlySet<string>;
}

export interface ResolvedContextOptions {
  readonly enabled: boolean;
  readonly maxChars: number;
  readonly maxItems: number;
  readonly query?: string;
}

export interface ResolvedEvidenceOptions {
  readonly enabled: boolean;
}

export interface ResolvedOptions {
  readonly endpoint: string;
  readonly apiKey: string;
  readonly projectId?: string;
  readonly agentId?: string;
  readonly requestTimeoutMs: number;
  readonly heartbeat: ResolvedHeartbeatOptions;
  readonly policy: ResolvedPolicyOptions;
  readonly context: ResolvedContextOptions;
  readonly evidence: ResolvedEvidenceOptions;
}

export class ConfigurationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ConfigurationError";
  }
}

const DEFAULT_READ_ONLY_TOOLS = [
  "read",
  "glob",
  "grep",
  "list",
  "ls",
  "search",
  "webfetch",
  "todoread",
] as const;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readString(source: Record<string, unknown>, key: string): string | undefined {
  const value = source[key];
  if (typeof value !== "string") return undefined;
  const trimmed = value.trim();
  return trimmed.length === 0 ? undefined : trimmed;
}

function readNumber(source: Record<string, unknown>, key: string): number | undefined {
  const value = source[key];
  if (typeof value !== "number" || !Number.isFinite(value)) return undefined;
  return value;
}

function readBoolean(source: Record<string, unknown>, key: string): boolean | undefined {
  const value = source[key];
  if (typeof value === "boolean") return value;
  return undefined;
}

function readStringArray(source: Record<string, unknown>, key: string): string[] | undefined {
  const value = source[key];
  if (!Array.isArray(value)) return undefined;
  const items = value.filter((item): item is string => typeof item === "string" && item.trim().length > 0);
  return items.map((item) => item.trim());
}

function readSection(source: Record<string, unknown>, key: string): Record<string, unknown> {
  const value = source[key];
  if (!isRecord(value)) return {};
  return value;
}

function firstDefined(...values: Array<string | undefined>): string | undefined {
  for (const value of values) {
    if (value !== undefined && value.length > 0) return value;
  }
  return undefined;
}

/**
 * Turns raw host options into a fully-populated configuration.
 *
 * Throws on a missing endpoint or key: a coordination plugin that cannot reach
 * the server cannot heartbeat, and a plugin that loads anyway looks healthy
 * while the agent's lease quietly expires.
 */
export function resolveOptions(raw: unknown, env: EnvLike = {}): ResolvedOptions {
  const source = isRecord(raw) ? raw : {};
  const heartbeat = readSection(source, "heartbeat");
  const policy = readSection(source, "policy");
  const context = readSection(source, "context");
  const evidence = readSection(source, "evidence");

  const endpoint = firstDefined(readString(source, "endpoint"), env["AGENT_SHAKER_ENDPOINT"]);
  const apiKey = firstDefined(readString(source, "api_key"), env["AGENT_SHAKER_API_KEY"]);
  const projectId = firstDefined(readString(source, "project_id"), env["AGENT_SHAKER_PROJECT_ID"]);
  const agentId = firstDefined(readString(source, "agent_id"), env["AGENT_SHAKER_AGENT_ID"]);

  if (!endpoint) {
    throw new ConfigurationError(
      "agent-shaker: `endpoint` is required (set the plugin option `endpoint` or AGENT_SHAKER_ENDPOINT).",
    );
  }
  if (!apiKey) {
    throw new ConfigurationError(
      "agent-shaker: `api_key` is required (set the plugin option `api_key` or AGENT_SHAKER_API_KEY).",
    );
  }

  const leaseFraction = readNumber(heartbeat, "lease_fraction") ?? 0.6;
  const minIntervalMs = readNumber(heartbeat, "min_interval_ms") ?? 5_000;
  const maxIntervalMs = readNumber(heartbeat, "max_interval_ms") ?? 300_000;

  const readOnlyTools = readStringArray(policy, "read_only_tools") ?? [...DEFAULT_READ_ONLY_TOOLS];
  const query = readString(context, "query");

  const resolvedContext: ResolvedContextOptions = {
    enabled: readBoolean(context, "enabled") ?? true,
    maxChars: readNumber(context, "max_chars") ?? 4_000,
    maxItems: readNumber(context, "max_items") ?? 8,
    ...(query === undefined ? {} : { query }),
  };

  const resolved: ResolvedOptions = {
    endpoint,
    apiKey,
    requestTimeoutMs: readNumber(source, "request_timeout_ms") ?? 10_000,
    heartbeat: {
      leaseFraction: Math.min(Math.max(leaseFraction, 0.1), 0.9),
      minIntervalMs: Math.max(minIntervalMs, 1_000),
      maxIntervalMs: Math.max(maxIntervalMs, 1_000),
    },
    policy: {
      enabled: readBoolean(policy, "enabled") ?? false,
      failClosed: readBoolean(policy, "fail_closed") ?? false,
      readOnlyTools: new Set(readOnlyTools.map((tool) => tool.toLowerCase())),
    },
    context: resolvedContext,
    evidence: {
      enabled: readBoolean(evidence, "enabled") ?? true,
    },
    ...(projectId === undefined ? {} : { projectId }),
    ...(agentId === undefined ? {} : { agentId }),
  };

  return resolved;
}
