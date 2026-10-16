/**
 * Download helpers for a generated setup bundle.
 *
 * What used to live here — the URL builder, the `.mcp.json` templates and the
 * instructions document — now lives on the server in `internal/configgen`, and
 * is fetched from `GET /api/agents/{id}/config`.
 *
 * The move was not cosmetic. The generator existed in the browser bundle and in
 * whatever a future CLI would have grown, and two implementations of "what
 * should this config say" drift: one gets a tool type added, the other does not,
 * and a developer is handed a config for the wrong client with nothing to say
 * why. One generator, on the server, is the only version that can be right.
 *
 * What remains is the part that genuinely needs a browser: turning strings into
 * files the operator can save.
 */

/**
 * No API_KEY_ENV_VAR here on purpose.
 *
 * It used to live in this file and now lives in `useAgentConfig`, which owns the
 * placeholder and the substitution. Nuxt auto-imports every composable and
 * silently drops a duplicate export with a warning, so a re-export would look
 * like it worked while one of the two definitions was being ignored.
 */

export interface SetupFile {
  /** Path relative to the agent's repository root, or a user-level target. */
  path: string
  contents: string
}

export function useMcpSetup() {
  /**
   * Download a single file.
   *
   * `type: 'application/json'` for JSON so editors do not treat it as a
   * download prompt, and the blob URL is revoked on the next tick so a
   * long-lived tab does not accumulate them.
   */
  function downloadFile(file: SetupFile): void {
    const mime = file.path.endsWith('.json') ? 'application/json' : 'text/markdown'
    const blob = new Blob([file.contents], { type: `${mime};charset=utf-8` })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = file.path.split('/').pop()!
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(url), 0)
  }

  /**
   * Download every file.
   *
   * Staggered because browsers silently drop concurrent downloads from one
   * gesture — without the delay only the first file would arrive.
   *
   * Takes the files rather than building them: the bundle comes from the server
   * now, and a caller that passed an input object would be asking this
   * composable to regenerate what it no longer owns.
   */
  function downloadAll(files: SetupFile[]): SetupFile[] {
    files.forEach((file, i) => setTimeout(() => downloadFile(file), i * 120))
    return files
  }

  return { downloadFile, downloadAll }
}