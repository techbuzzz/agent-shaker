import type { ShakerContextEntry } from "./client.js";
import { errorMessage, type Logger } from "./logger.js";

export interface ContextEntry {
  readonly id?: string;
  readonly title: string;
  readonly content: string;
  readonly createdAt?: string;
  /** Full-text relevance rank, when the entry came from a search. */
  readonly rank?: number;
  readonly tags?: readonly string[];
}

export interface ContextPayload {
  readonly text: string;
  readonly entries: number;
  readonly truncated: boolean;
}

export interface ContextPayloadOptions {
  readonly maxChars: number;
  readonly maxItems: number;
}

const EMPTY_PAYLOAD: ContextPayload = { text: "", entries: 0, truncated: false };

export function toContextEntry(entry: ShakerContextEntry): ContextEntry {
  return {
    title: entry.title,
    content: entry.content ?? entry.preview ?? "",
    ...(entry.id === undefined ? {} : { id: entry.id }),
    ...(entry.created_at === undefined ? {} : { createdAt: entry.created_at }),
    ...(entry.rank === undefined ? {} : { rank: entry.rank }),
    ...(entry.tags === undefined ? {} : { tags: entry.tags }),
  };
}

/**
 * Orders shared contexts by how likely they are to matter.
 *
 * A search result carries a relevance rank; a plain listing carries only a
 * creation time, and the most recent note is the most likely to be the one the
 * agent is currently working against.
 */
export function orderByRelevance(entries: readonly ContextEntry[]): ContextEntry[] {
  return [...entries].sort((left, right) => {
    const rankDelta = (right.rank ?? 0) - (left.rank ?? 0);
    if (rankDelta !== 0) return rankDelta;
    return (right.createdAt ?? "").localeCompare(left.createdAt ?? "");
  });
}

/**
 * Builds the injected payload under a hard character budget.
 *
 * The cap is not a nicety: the payload is prepended to every model call in the
 * turn, so an unbounded injection grows the prompt until the provider rejects
 * it. Truncation is marked so the model can tell a short note from a cut one.
 */
export function buildContextPayload(
  entries: readonly ContextEntry[],
  options: ContextPayloadOptions,
): ContextPayload {
  const ordered = orderByRelevance(entries).slice(0, Math.max(0, Math.trunc(options.maxItems)));
  const maxChars = Math.max(0, Math.trunc(options.maxChars));
  const marker = "\n[shared context truncated]";
  // The marker is part of the budget: a payload that only fits because the
  // truncation notice was not counted would still blow the cap on the wire.
  const contentBudget = Math.max(0, maxChars - marker.length - 1);

  const chunks: string[] = [];
  let used = 0;
  let truncated = false;

  for (const entry of ordered) {
    if (used >= contentBudget) {
      truncated = true;
      break;
    }
    const body = entry.content.trim();
    const block = `### ${entry.title}\n${body}\n`;
    if (used + block.length <= contentBudget) {
      chunks.push(block);
      used += block.length;
      continue;
    }
    const remaining = contentBudget - used;
    if (remaining > 0) {
      chunks.push(block.slice(0, remaining));
      used += remaining;
    }
    truncated = true;
    break;
  }

  if (truncated) {
    chunks.push(marker);
  }

  let text = chunks.join("\n");
  if (text.length > maxChars) {
    text = `${text.slice(0, Math.max(0, maxChars - marker.length))}${marker}`;
  }

  return { text, entries: ordered.length, truncated };
}

export type ContextFetcher = (sessionId: string) => Promise<readonly ContextEntry[]>;

/**
 * Per-session cache of the shared context.
 *
 * The context hook fires on every model call, including the tool-driven
 * continuations inside one prompt. Fetching per call would multiply the server
 * load by the number of steps in a turn and change the injected prefix between
 * two calls of the same prompt, which busts the prompt cache exactly when the
 * conversation is longest. The cache is therefore cleared when a new prompt
 * starts, not when a model call starts.
 */
export class ContextCache {
  private readonly fetcher: ContextFetcher;
  private readonly options: ContextPayloadOptions;
  private readonly logger: Logger | undefined;
  private readonly generations = new Map<string, number>();
  private readonly payloads = new Map<string, ContextPayload>();
  private readonly inFlight = new Map<string, Promise<ContextPayload>>();

  constructor(fetcher: ContextFetcher, options: ContextPayloadOptions, logger?: Logger) {
    this.fetcher = fetcher;
    this.options = options;
    this.logger = logger;
  }

  /** Marks the start of a new prompt: the next read fetches again. */
  beginPrompt(sessionId: string): void {
    this.generations.set(sessionId, (this.generations.get(sessionId) ?? 0) + 1);
    this.payloads.delete(sessionId);
    this.inFlight.delete(sessionId);
  }

  async get(sessionId: string): Promise<ContextPayload> {
    const cached = this.payloads.get(sessionId);
    if (cached) return cached;
    const pending = this.inFlight.get(sessionId);
    if (pending) return pending;
    const generation = this.generations.get(sessionId) ?? 0;
    const promise = this.load(sessionId, generation);
    this.inFlight.set(sessionId, promise);
    return promise;
  }

  /** Number of sessions holding cached state; used by the tests to prove cleanup. */
  get size(): number {
    return this.payloads.size;
  }

  private async load(sessionId: string, generation: number): Promise<ContextPayload> {
    try {
      const entries = await this.fetcher(sessionId);
      const payload = buildContextPayload(entries, this.options);
      if ((this.generations.get(sessionId) ?? 0) === generation) {
        this.payloads.set(sessionId, payload);
        this.inFlight.delete(sessionId);
      }
      return payload;
    } catch (error) {
      // A coordination outage must not break the turn, and it must not be
      // retried on every model call inside the same prompt either.
      this.logger?.warn("agent-shaker: shared context fetch failed", {
        session: sessionId,
        error: errorMessage(error),
      });
      if ((this.generations.get(sessionId) ?? 0) === generation) {
        this.inFlight.delete(sessionId);
      }
      return EMPTY_PAYLOAD;
    }
  }
}
