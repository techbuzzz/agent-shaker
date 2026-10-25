/**
 * The capability announcement.
 *
 * The server has to know what a client can do for it, and it cannot guess: an
 * agent whose client cannot accept a pushed context, or author a compaction
 * summary, is an agent the server must not ask to.
 *
 * The vocabulary here is the server's, not the plugin's. The plugin's own
 * capability probe answers "what does this host expose"; these names answer
 * "what can this client do for the server". They are different questions, and
 * only the second one is worth a round trip.
 */

import type { ShakerApi } from "./client.js";
import { errorMessage, type Logger } from "./logger.js";

export const SERVER_CAPABILITY_COMPACTION = "compaction";
export const SERVER_CAPABILITY_CONTEXT_INJECTION = "context_injection";
export const SERVER_CAPABILITY_POLICY_GATE = "policy_gate";
export const SERVER_CAPABILITY_EVIDENCE_CAPTURE = "evidence_capture";

/** The host surfaces the plugin needs before it can deliver a capability. */
export interface HostSurfaces {
  /** The host can register a session hook — compaction and context injection. */
  readonly sessionHook: boolean;
  /** The host can register a tool hook — the policy gate and the evidence trail. */
  readonly toolHook: boolean;
}

/** What the plugin is configured to do. */
export interface PluginFeatures {
  readonly policyEnabled: boolean;
  readonly contextEnabled: boolean;
  readonly evidenceEnabled: boolean;
}

/**
 * The capabilities this client can deliver.
 *
 * A capability is claimed only when the host surface that carries it exists
 * *and* the plugin is configured to use it. Announcing a capability the client
 * cannot honour is worse than announcing nothing: the server would route work
 * that then failed, with no explanation.
 */
export function serverCapabilities(surfaces: HostSurfaces, features: PluginFeatures): string[] {
  const capabilities: string[] = [];

  if (surfaces.sessionHook) {
    capabilities.push(SERVER_CAPABILITY_COMPACTION);
    if (features.contextEnabled) {
      capabilities.push(SERVER_CAPABILITY_CONTEXT_INJECTION);
    }
  }

  if (surfaces.toolHook) {
    if (features.policyEnabled) {
      capabilities.push(SERVER_CAPABILITY_POLICY_GATE);
    }
    if (features.evidenceEnabled) {
      capabilities.push(SERVER_CAPABILITY_EVIDENCE_CAPTURE);
    }
  }

  return capabilities;
}

/**
 * Announces the capabilities, once.
 *
 * Failures are logged and swallowed: the announcement is a statement about
 * this client, and a client that cannot make the statement is still a client
 * that can hold a lease. Refusing to work because the server did not accept a
 * piece of metadata would be a worse failure than the missing metadata.
 */
export async function announceCapabilities(
  client: ShakerApi,
  capabilities: readonly string[],
  host: string,
  pluginVersion: string,
  logger: Logger,
): Promise<void> {
  if (capabilities.length === 0) return;

  try {
    await client.announceCapability({
      capabilities: [...capabilities],
      host,
      pluginVersion,
    });
  } catch (error) {
    logger.warn("agent-shaker: capability announcement failed", { error: errorMessage(error) });
  }
}
