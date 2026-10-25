import type {
  ShakerApi,
  ShakerBlocker,
  ShakerDependencySummary,
  ShakerTask,
} from "./client.js";
import { type ContextEntry, toContextEntry } from "./context.js";
import { errorMessage, type Logger } from "./logger.js";

export interface CoordinationSnapshot {
  readonly projectId?: string;
  readonly agentId?: string;
  readonly sessionId?: string;
  readonly task?: ShakerTask;
  readonly blockers: readonly ShakerBlocker[];
  readonly dependsOn: readonly ShakerDependencySummary[];
  readonly dependsFor: readonly ShakerDependencySummary[];
  readonly contexts: readonly ContextEntry[];
}

export interface SnapshotOptions {
  readonly sessionId?: string;
  readonly projectId?: string;
  readonly agentId?: string;
  readonly maxContextItems?: number;
  readonly contextQuery?: string;
}

export interface CompactionResult {
  readonly summary: string;
  readonly metadata: Record<string, unknown>;
}

/**
 * Task statuses that mean the agent is holding the task.
 *
 * A blocked task still counts: the agent is blocked *on* it, and letting it
 * write to the repository while blocked is exactly what the gate is there to
 * prevent.
 */
const CLAIMED_TASK_STATUSES = new Set(["pending", "ready", "in_progress", "blocked"]);

const OPEN_BLOCKER_STATUSES = new Set(["open", "in_review", "in-review"]);

function isClaimed(task: ShakerTask): boolean {
  return CLAIMED_TASK_STATUSES.has(task.status);
}

function isOpen(blocker: ShakerBlocker): boolean {
  return OPEN_BLOCKER_STATUSES.has(blocker.status);
}

/**
 * Reads the shared truth the coordination plane already holds.
 *
 * Every part is optional: a partially reachable server still yields a usable
 * snapshot, and a snapshot that skips one section beats no snapshot at all.
 */
export async function collectSnapshot(
  client: ShakerApi,
  options: SnapshotOptions,
  logger?: Logger,
): Promise<CoordinationSnapshot> {
  const identity = await safe(() => client.getIdentity(), logger, "get_my_identity");
  const projectId = options.projectId ?? identity?.agent?.project_id ?? identity?.project_id;
  const agentId = options.agentId ?? identity?.agent_id;
  const tasks = await safe(() => client.getMyTasks(), logger, "get_my_tasks");
  const task = pickTask(tasks?.tasks ?? []);
  const blockers = await safe(() => client.getMyBlockers(), logger, "get_my_blockers");
  const dependencies = task
    ? await safe(() => client.getTaskDependencies(task.id), logger, "get_task_dependencies")
    : undefined;
  const contexts = await loadContexts(client, projectId, options, logger);

  return {
    ...(projectId === undefined ? {} : { projectId }),
    ...(agentId === undefined ? {} : { agentId }),
    ...(options.sessionId === undefined ? {} : { sessionId: options.sessionId }),
    ...(task === undefined ? {} : { task }),
    blockers: (blockers?.blockers ?? []).filter(isOpen),
    dependsOn: dependencies?.depends_on ?? [],
    dependsFor: dependencies?.depends_for ?? [],
    contexts,
  };
}

function pickTask(tasks: readonly ShakerTask[]): ShakerTask | undefined {
  const inProgress = tasks.find((task) => task.status === "in_progress");
  if (inProgress) return inProgress;
  return tasks.find(isClaimed) ?? tasks[0];
}

async function loadContexts(
  client: ShakerApi,
  projectId: string | undefined,
  options: SnapshotOptions,
  logger?: Logger,
): Promise<ContextEntry[]> {
  const limit = options.maxContextItems ?? 8;
  if (projectId === undefined) {
    // Listing without a project filter returns every project on the server, so
    // the shared context is skipped rather than leaked across projects.
    logger?.warn("agent-shaker: no project id resolved, skipping shared context", {});
    return [];
  }
  const query = options.contextQuery;
  if (query !== undefined && query.length > 0) {
    const found = await safe(
      () => client.searchContexts({ query, projectId, limit }),
      logger,
      "search_contexts",
    );
    return (found?.contexts ?? []).map(toContextEntry);
  }
  const listed = await safe(() => client.listContexts(projectId), logger, "list_contexts");
  return (listed?.contexts ?? []).slice(0, limit).map(toContextEntry);
}

async function safe<T>(
  run: () => Promise<T>,
  logger: Logger | undefined,
  label: string,
): Promise<T | undefined> {
  try {
    return await run();
  } catch (error) {
    logger?.warn(`agent-shaker: ${label} failed while collecting the coordination snapshot`, {
      error: errorMessage(error),
    });
    return undefined;
  }
}

/**
 * Writes the compaction summary.
 *
 * The summary is the artifact the model is about to lose everything else to,
 * so it is built from the coordination plane rather than from the transcript:
 * the task, the blockers, the dependency edges and the shared notes are the
 * facts a compacted session cannot recover on its own.
 */
export function buildCompactionSummary(snapshot: CoordinationSnapshot): string {
  const lines: string[] = ["# Agent Shaker coordination snapshot", ""];

  lines.push(`- Project: ${snapshot.projectId ?? "unknown"}`);
  lines.push(`- Agent: ${snapshot.agentId ?? "unknown"}`);
  lines.push(`- Session: ${snapshot.sessionId ?? "unknown"}`);

  if (snapshot.task) {
    lines.push(
      `- Task: ${snapshot.task.title} (${snapshot.task.id}) — status ${snapshot.task.status}, priority ${snapshot.task.priority}`,
    );
    const description = snapshot.task.description?.trim();
    if (description) lines.push(`  ${description}`);
  } else {
    lines.push("- Task: none claimed");
  }

  if (snapshot.blockers.length > 0) {
    lines.push(`- Open blockers (${snapshot.blockers.length}):`);
    for (const blocker of snapshot.blockers) {
      lines.push(`  - [${blocker.status}] ${blocker.title}: ${blocker.description}`);
    }
  } else {
    lines.push("- Open blockers: none");
  }

  if (snapshot.dependsOn.length > 0) {
    lines.push("- Waiting on:");
    for (const dependency of snapshot.dependsOn) {
      lines.push(`  - ${dependency.title ?? dependency.task_id} [${dependency.status ?? "unknown"}]`);
    }
  }

  if (snapshot.dependsFor.length > 0) {
    lines.push("- Blocking:");
    for (const dependency of snapshot.dependsFor) {
      lines.push(`  - ${dependency.title ?? dependency.task_id} [${dependency.status ?? "unknown"}]`);
    }
  }

  if (snapshot.contexts.length > 0) {
    lines.push(`- Shared context (${snapshot.contexts.length}):`);
    for (const context of snapshot.contexts) {
      const excerpt = context.content.trim().split("\n")[0] ?? "";
      lines.push(`  - ${context.title}${excerpt ? `: ${excerpt}` : ""}`);
    }
  }

  return lines.join("\n");
}

/**
 * The identifiers that let the next session pick the work up again.
 *
 * They travel with the summary because the summary replaces the conversation:
 * without them the compacted session knows what it was doing but not which
 * coordination records it owns.
 */
export function buildCompactionMetadata(snapshot: CoordinationSnapshot): Record<string, unknown> {
  return {
    source: "agent-shaker",
    generated_at: new Date().toISOString(),
    project_id: snapshot.projectId ?? null,
    agent_id: snapshot.agentId ?? null,
    session_id: snapshot.sessionId ?? null,
    task_id: snapshot.task?.id ?? null,
    task_status: snapshot.task?.status ?? null,
    blockers: snapshot.blockers.length,
    depends_on: snapshot.dependsOn.length,
    depends_for: snapshot.dependsFor.length,
    contexts: snapshot.contexts.length,
  };
}

/**
 * Stores the digest in the coordination plane.
 *
 * The model is about to see a summary that only exists in its own context; the
 * plane has to hold the same artifact, or the next agent that reads the project
 * sees a session that decided something nobody can find.
 */
export async function persistDigest(
  client: ShakerApi,
  snapshot: CoordinationSnapshot,
  summary: string,
): Promise<void> {
  const title = `Compaction digest${snapshot.sessionId ? ` — ${snapshot.sessionId}` : ""}`;
  await client.addContext({
    title,
    content: summary,
    tags: ["compaction", "digest"],
  });
}

/**
 * Which compaction path is actually in play.
 *
 * The v2 compaction hook is new enough that its firing is not confirmed
 * outside this plugin, so the event stream is observed as well and the mode is
 * reported instead of assumed.
 */
export type CompactionMode = "authoring" | "observation" | "authoring+observation";

export function resolveCompactionMode(authored: boolean, observed: boolean): CompactionMode {
  if (authored && observed) return "authoring+observation";
  if (authored) return "authoring";
  return "observation";
}
