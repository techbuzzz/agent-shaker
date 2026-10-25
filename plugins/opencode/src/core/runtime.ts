import type { ShakerApi } from "./client.js";
import { ShakerClient } from "./client.js";
import { type EnvLike, type ResolvedOptions } from "./config.js";
import { ContextCache, toContextEntry } from "./context.js";
import { type LifecycleTimers, LifecycleBridge } from "./lifecycle.js";
import { type Logger, consoleLogger } from "./logger.js";
import { PolicyGate } from "./policy.js";

export interface RuntimeDeps {
  readonly createClient?: (options: ResolvedOptions) => ShakerApi;
  readonly logger?: Logger;
  readonly env?: EnvLike;
  readonly timers?: LifecycleTimers;
}

export interface ShakerRuntime {
  readonly options: ResolvedOptions;
  readonly client: ShakerApi;
  readonly logger: Logger;
  readonly bridge: LifecycleBridge;
  readonly cache: ContextCache;
  readonly policy: PolicyGate;
  /** Project id resolved from the credential, once. */
  projectId(): Promise<string | undefined>;
}

/**
 * Builds the shared core both host adapters drive.
 *
 * The adapters differ in how they receive events and how they are handed a
 * system prompt, but the coordination behaviour has to be identical, so it is
 * assembled exactly once here.
 */
export function createRuntime(options: ResolvedOptions, deps: RuntimeDeps = {}): ShakerRuntime {
  const logger = deps.logger ?? consoleLogger;
  const client = deps.createClient ? deps.createClient(options) : ShakerClient.fromResolved(options);

  let projectIdPromise: Promise<string | undefined> | undefined;
  const projectId = (): Promise<string | undefined> => {
    if (!projectIdPromise) {
      projectIdPromise = client
        .getIdentity()
        .then((identity) => identity.agent?.project_id ?? identity.project_id ?? options.projectId)
        .catch(() => options.projectId);
    }
    return projectIdPromise;
  };

  const cache = new ContextCache(
    async () => {
      const scope = await projectId();
      if (scope === undefined) return [];
      const query = options.context.query;
      if (query !== undefined && query.length > 0) {
        const found = await client.searchContexts({ query, projectId: scope, limit: options.context.maxItems });
        return found.contexts.map(toContextEntry);
      }
      const listed = await client.listContexts(scope);
      return listed.contexts.map(toContextEntry);
    },
    { maxChars: options.context.maxChars, maxItems: options.context.maxItems },
    logger,
  );

  const bridge = new LifecycleBridge(
    client,
    { heartbeat: options.heartbeat, ...(deps.timers ? { timers: deps.timers } : {}) },
    logger,
  );

  const policy = new PolicyGate(client, options.policy, logger);

  return { options, client, logger, bridge, cache, policy, projectId };
}
