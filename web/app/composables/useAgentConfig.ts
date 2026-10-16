/**
 * Client-side rendering of a generated agent configuration.
 *
 * The server owns what a config contains (internal/configgen) and never sends a
 * credential inside one. This composable owns the one step that has to happen in
 * the browser: putting the plaintext the mint response carried into the
 * placeholder, at copy time, in memory.
 *
 * Nothing here is persisted. The secret lives in the ref the connection modal
 * holds and is dropped when that modal closes, because the server cannot return
 * it a second time.
 */

/**
 * The placeholder every generated credential references.
 *
 * Must match `configgen.APIKeyPlaceholder` on the Go side. It is re-declared
 * rather than fetched: a generated file is supposed to be readable without a
 * round trip, and the API response carries the same value in `env_var` for
 * display.
 *
 * API_KEY_ENV_VAR lives here and only here — Nuxt auto-imports composables, so
 * a second export of the same name is silently dropped with a warning and the
 * loser is whichever file Nuxt scans first.
 */
export const API_KEY_ENV_VAR = 'AGENT_SHAKER_API_KEY'
export const API_KEY_PLACEHOLDER = `\${env:${API_KEY_ENV_VAR}}`

/**
 * OpenCode's spelling of the same idea — one brace pair, no dollar.
 *
 * Per https://opencode.ai/docs/mcp-servers/ OpenCode expands `{env:VAR}` and
 * leaves `${env:VAR}` alone. It is included here rather than treated as an
 * OpenCode quirk because a file carrying the wrong one is valid JSON with a
 * real-looking header: it fails at the server as a 401 that reads like a revoked
 * key, and the operator has no way to guess it is a formatting difference.
 *
 * Note this is a substring of API_KEY_PLACEHOLDER, so a replacement ordered
 * from longest to shortest must not match `${…}` with the short pattern first.
 */
export const OPENCODE_API_KEY_PLACEHOLDER = `{env:${API_KEY_ENV_VAR}}`

/** Every spelling the server can render, longest first so none is a prefix of another. */
const PLACEHOLDERS: readonly string[] = [
  API_KEY_PLACEHOLDER,
  OPENCODE_API_KEY_PLACEHOLDER
].sort((a, b) => b.length - a.length)

/**
 * Substitute the plaintext into a generated file.
 *
 * Implemented as split/join rather than `String.replace`, and that is the whole
 * point of this function.
 *
 * The placeholder contains `$`, and in a replacement *string* JavaScript treats
 * `$&`, `$'`, `` $` ``, `$1` and `$$` as substitution patterns referring to
 * parts of the match. A secret containing any of those sequences — which is
 * plausible for a base64url key with an unlucky character run — would be
 * silently rewritten by `replace`, producing a credential that looks right,
 * copies right, and authenticates as nobody. Splitting on a string and joining
 * the parts treats the secret as literal text, which is the only correct
 * behaviour here.
 *
 * Content with no placeholder is returned untouched, so a caller can pipe every
 * file through this without checking which ones carry a credential.
 */
export function injectSecret(content: string, secret: string): string {
  if (!content || !secret) return content
  // Every known spelling, not just the default one: the browser does not know
  // which client a file is for, and leaving the OpenCode form untouched would
  // hand the operator a file that still says `{env:AGENT_SHAKER_API_KEY}` after
  // they pasted their key.
  return PLACEHOLDERS.reduce((acc, ph) => acc.split(ph).join(secret), content)
}

/**
 * The file contents as an operator would actually paste them, with the secret
 * already substituted.
 *
 * A downloaded bundle keeps the placeholder on purpose: a file that can be
 * committed is worth more than one that works the moment it is written. Copying
 * is the moment the secret is genuinely needed, and it is the moment the
 * operator is looking at the screen.
 */
export function renderedFile(
  file: { filename?: string; target?: string; content: string },
  secret: string
): string {
  return injectSecret(file.content, secret)
}

/** Where a file goes: the repository path, or the user-level target. */
export function fileLocation(file: { filename?: string; target?: string }): string {
  return file.filename || file.target || '(unnamed)'
}

/**
 * Copy text to the clipboard.
 *
 * Returns whether it worked rather than throwing: the clipboard API rejects
 * outside a user gesture and in insecure contexts, and a connection modal that
 * throws on copy leaves the operator holding a secret they cannot retrieve
 * again — a worse outcome than a message saying the copy failed.
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

/** Trigger a file download for one generated config. */
export function downloadFile(name: string, contents: string): void {
  const mime = name.endsWith('.json') ? 'application/json' : 'text/markdown'
  const blob = new Blob([contents], { type: `${mime};charset=utf-8` })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  // Revoked on the next tick rather than immediately: Safari cancels an
  // in-flight download whose object URL disappears in the same turn.
  setTimeout(() => URL.revokeObjectURL(url), 0)
}

export function useAgentConfig() {
  return { injectSecret, renderedFile, fileLocation, copyToClipboard, downloadFile }
}