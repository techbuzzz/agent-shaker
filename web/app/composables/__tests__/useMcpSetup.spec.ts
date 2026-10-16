import { describe, expect, it } from 'vitest'
import { useMcpSetup } from '../useMcpSetup'
import { API_KEY_ENV_VAR } from '../useAgentConfig'

/**
 * What is left of this composable is the part that genuinely needs a browser:
 * turning strings into files. The URL builder and the `.mcp.json` templates moved
 * to `internal/configgen`, because the generator existed in the browser bundle
 * and would have had to exist again in any CLI — two implementations of "what
 * should this config say", drifting apart.
 *
 * Their behaviour is pinned by the Go tests in internal/configgen, including the
 * "no credential in a rendered file" invariant that used to be checked here.
 *
 * API_KEY_ENV_VAR is asserted from useAgentConfig rather than from here on
 * purpose: it lives with the placeholder and the substitution, and a second
 * export would be dropped by Nuxt's auto-import scan with only a warning.
 */
describe('useMcpSetup', () => {
  it('exposes both download helpers', () => {
    const { downloadFile, downloadAll } = useMcpSetup()
    expect(typeof downloadFile).toBe('function')
    expect(typeof downloadAll).toBe('function')
  })

  it('names the environment variable the generated files reference', () => {
    // The value has to match what the server rendered, or the operator is told
    // to set the wrong variable and the agent cannot authenticate at all.
    expect(API_KEY_ENV_VAR).toBe('AGENT_SHAKER_API_KEY')
  })
})