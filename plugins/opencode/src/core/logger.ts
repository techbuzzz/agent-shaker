export type LogDetail = Readonly<Record<string, unknown>>;

export interface Logger {
  info(message: string, detail?: LogDetail): void;
  warn(message: string, detail?: LogDetail): void;
  error(message: string, detail?: LogDetail): void;
  /**
   * Emits at most one warning for a given key.
   *
   * Capability losses are reported once per plugin load: a missing hook fires on
   * every model call, and a warning per call turns one degradation into a log
   * flood that hides the degradation itself.
   */
  warnOnce(key: string, message: string, detail?: LogDetail): void;
}

export interface LoggerSink {
  info?(message: string, detail?: LogDetail): void;
  warn?(message: string, detail?: LogDetail): void;
  error?(message: string, detail?: LogDetail): void;
}

function stringify(value: unknown): string {
  if (typeof value === "string") return value;
  if (value === null || value === undefined) return String(value);
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    return JSON.stringify(value) ?? String(value);
  } catch {
    return "[unserialisable]";
  }
}

function withDetail(message: string, detail?: LogDetail): string {
  if (!detail) return message;
  const parts = Object.entries(detail).map(([key, value]) => `${key}=${stringify(value)}`);
  if (parts.length === 0) return message;
  return `${message} ${parts.join(" ")}`;
}

export function createLogger(sink: LoggerSink = {}): Logger {
  const warned = new Set<string>();
  return {
    info: (message, detail) => sink.info?.(withDetail(message, detail)),
    warn: (message, detail) => sink.warn?.(withDetail(message, detail)),
    error: (message, detail) => sink.error?.(withDetail(message, detail)),
    warnOnce: (key, message, detail) => {
      if (warned.has(key)) return;
      warned.add(key);
      sink.warn?.(withDetail(message, detail));
    },
  };
}

export const consoleLogger: Logger = createLogger({
  info: (message) => console.info(message),
  warn: (message) => console.warn(message),
  error: (message) => console.error(message),
});

/** Extracts a printable message from an unknown thrown value. */
export function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  return stringify(error);
}
