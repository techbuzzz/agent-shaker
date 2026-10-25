/**
 * Capability probing.
 *
 * The host API surface is not negotiated: the plugin is handed whichever
 * context the host has, and has to decide what it can do with it. Probing is
 * explicit so that a degradation is reported once, by name, instead of showing
 * up as a feature that quietly does nothing.
 */

export const CAPABILITY_EVENT_STREAM = "event-stream";
export const CAPABILITY_SESSION_HOOK = "session-hook";
export const CAPABILITY_TOOL_HOOK = "tool-hook";

export type CapabilityName =
  | typeof CAPABILITY_EVENT_STREAM
  | typeof CAPABILITY_SESSION_HOOK
  | typeof CAPABILITY_TOOL_HOOK;

export interface CapabilitySurface {
  readonly session?: { readonly hook?: unknown };
  readonly tool?: { readonly hook?: unknown };
  readonly event?: { readonly subscribe?: unknown };
}

export interface CapabilityReport {
  readonly present: readonly CapabilityName[];
  readonly missing: readonly CapabilityName[];
}

export class CapabilityError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "CapabilityError";
  }
}

const ALL_CAPABILITIES: readonly CapabilityName[] = [
  CAPABILITY_EVENT_STREAM,
  CAPABILITY_SESSION_HOOK,
  CAPABILITY_TOOL_HOOK,
];

function hasHook(surface: CapabilitySurface, domain: "session" | "tool"): boolean {
  const value = surface[domain];
  if (!value || typeof value !== "object") return false;
  return typeof value.hook === "function";
}

function hasSubscribe(surface: CapabilitySurface): boolean {
  const value = surface.event;
  if (!value || typeof value !== "object") return false;
  return typeof value.subscribe === "function";
}

export function probeCapabilities(surface: CapabilitySurface): CapabilityReport {
  const present: CapabilityName[] = [];
  const missing: CapabilityName[] = [];

  for (const capability of ALL_CAPABILITIES) {
    const available =
      capability === CAPABILITY_EVENT_STREAM
        ? hasSubscribe(surface)
        : hasHook(surface, capability === CAPABILITY_SESSION_HOOK ? "session" : "tool");
    if (available) {
      present.push(capability);
      continue;
    }
    missing.push(capability);
  }

  return { present, missing };
}

export function hasCapability(report: CapabilityReport, capability: CapabilityName): boolean {
  return report.present.includes(capability);
}

/**
 * Refuses to load when the session lifecycle cannot be observed.
 *
 * The event stream is the only source of `session.created`, `session.idle` and
 * `session.deleted`. Without it the plugin cannot take a lease, cannot renew
 * it, and cannot stop it — which is a silently dead plugin rather than a
 * degraded one. Refusing to load is the honest outcome.
 */
export function assertLifecycleCapability(report: CapabilityReport): void {
  if (hasCapability(report, CAPABILITY_EVENT_STREAM)) return;
  throw new CapabilityError(
    "agent-shaker: this OpenCode host exposes no session event stream, so the plugin cannot observe " +
      "session lifecycle and therefore cannot hold a coordination lease. Refusing to load.",
  );
}

/** Human-readable list of the capabilities that are not available. */
export function describeMissing(report: CapabilityReport): string {
  return report.missing.join(", ");
}
