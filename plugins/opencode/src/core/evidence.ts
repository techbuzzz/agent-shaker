import type { ShakerApi } from "./client.js";
import { errorMessage, type Logger } from "./logger.js";

export interface EvidenceOutcome {
  readonly tool: string;
  readonly command?: string;
  readonly status: "completed" | "error";
  readonly output?: string;
  readonly errorMessage?: string;
  readonly sessionId?: string;
}

export interface EvidenceEntry {
  readonly title: string;
  readonly content: string;
  readonly tags: readonly string[];
}

/**
 * Commands that produce a verdict about the work.
 *
 * The check insists on a known build or test runner in front of the verb, so a
 * commit message that happens to contain the word "test" is not mistaken for a
 * test run.
 */
const BUILD_TOOLS = new Set([
  "npm",
  "pnpm",
  "yarn",
  "bun",
  "deno",
  "go",
  "cargo",
  "mvn",
  "gradle",
  "gradlew",
  "make",
  "cmake",
  "ninja",
  "bazel",
  "dotnet",
  "docker",
  "podman",
]);

/** Tools that are a verification run on their own, without a separate verb. */
const SELF_SUFFICIENT_TOOLS = new Set([
  "pytest",
  "vitest",
  "jest",
  "mocha",
  "phpunit",
  "rspec",
  "tox",
  "ctest",
  "tsc",
]);

const VERBS = new Set(["test", "tests", "build", "compile", "check", "typecheck", "lint", "package"]);

const TOOL_NAME_PATTERN = /test|build|compile|typecheck|check/i;

const MAX_OUTPUT_CHARS = 4_000;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function tokenize(command: string): string[] {
  return command
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter((token) => token.length > 0);
}

/** Pulls the shell command out of a tool call, when there is one. */
export function extractCommand(tool: string, input: unknown): string | undefined {
  if (!isRecord(input)) return undefined;
  const direct = input["command"];
  if (typeof direct === "string" && direct.trim().length > 0) return direct;
  const script = input["script"];
  if (typeof script === "string" && script.trim().length > 0) return script;
  if (tool.toLowerCase() === "bash" || tool.toLowerCase() === "shell") {
    const args = input["args"];
    if (typeof args === "string" && args.trim().length > 0) return args;
  }
  return undefined;
}

export function isTestOrBuildCommand(tool: string, command: string | undefined): boolean {
  if (TOOL_NAME_PATTERN.test(tool)) return true;
  if (command === undefined) return false;

  const tokens = tokenize(command);
  const hasRunner = tokens.some((token) => BUILD_TOOLS.has(token));
  const hasSelfSufficient = tokens.some((token) => SELF_SUFFICIENT_TOOLS.has(token));
  const hasVerb = tokens.some((token) => VERBS.has(token));
  if (hasSelfSufficient) return true;
  return hasRunner && hasVerb;
}

export function buildEvidence(outcome: EvidenceOutcome): EvidenceEntry {
  const verdict = outcome.status === "error" ? "failed" : "passed";
  const title = `Evidence: ${outcome.command ?? outcome.tool} — ${verdict}`;
  const lines: string[] = [];
  lines.push(`- Tool: ${outcome.tool}`);
  if (outcome.command !== undefined) lines.push(`- Command: ${outcome.command}`);
  lines.push(`- Outcome: ${verdict}`);
  if (outcome.sessionId !== undefined) lines.push(`- Session: ${outcome.sessionId}`);
  if (outcome.errorMessage !== undefined) lines.push(`- Error: ${outcome.errorMessage}`);
  if (outcome.output !== undefined && outcome.output.length > 0) {
    lines.push("", "## Output", "", truncate(outcome.output, MAX_OUTPUT_CHARS));
  }
  return {
    title,
    content: lines.join("\n"),
    tags: outcome.status === "error" ? ["evidence", "failure"] : ["evidence", "success"],
  };
}

/**
 * Writes the outcome of a test or build run into the project's shared context.
 *
 * The outcome of a verification run is the part of a session that nobody can
 * reconstruct afterwards: the next agent inherits a repository state and a
 * claim about that state, and the claim is only worth anything if it was
 * written down when it was true.
 */
export async function captureEvidence(
  client: ShakerApi,
  outcome: EvidenceOutcome,
  logger?: Logger,
): Promise<void> {
  const entry = buildEvidence(outcome);
  try {
    await client.addContext({ title: entry.title, content: entry.content, tags: entry.tags });
  } catch (error) {
    logger?.warn("agent-shaker: evidence capture failed", { error: errorMessage(error) });
  }
}

function truncate(value: string, limit: number): string {
  if (value.length <= limit) return value;
  return `${value.slice(0, limit)}\n[output truncated]`;
}
