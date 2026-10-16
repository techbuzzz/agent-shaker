import { describe, it, expect } from 'vitest'
import {
  API_KEY_ENV_VAR,
  API_KEY_PLACEHOLDER,
  fileLocation,
  injectSecret
} from '../useAgentConfig'

describe('injectSecret', () => {
  const config = `{
  "headers": {
    "Authorization": "Bearer ${API_KEY_PLACEHOLDER}"
  }
}`

  it('substitutes the plaintext', () => {
    const out = injectSecret(config, 'ask_abc123')
    expect(out).toContain('Bearer ask_abc123')
    expect(out).not.toContain(API_KEY_PLACEHOLDER)
  })

  it('substitutes every occurrence', () => {
    const twice = `${API_KEY_PLACEHOLDER} and ${API_KEY_PLACEHOLDER}`
    const out = injectSecret(twice, 'ask_x')
    expect(out).toBe('ask_x and ask_x')
  })

  // The reason this function exists rather than a one-line String.replace.
  //
  // In a replacement *string*, JavaScript treats $&, $', $`, $1 and $$ as
  // patterns referring to parts of the match. A secret containing any of those
  // sequences would be silently rewritten — producing a credential that looks
  // right, copies right, and authenticates as nobody. There is no warning and
  // no error; the failure surfaces later as a 401 the developer cannot explain.
  describe('secrets containing dollar sequences', () => {
    const dangerous = [
      'ask_$&abc',
      "ask_$'abc",
      'ask_$`abc',
      'ask_$1abc',
      'ask_$$abc',
      'ask_$&$`$\''
    ]

    for (const secret of dangerous) {
      it(`is inserted literally: ${secret}`, () => {
        const out = injectSecret(config, secret)
        expect(out).toContain(`Bearer ${secret}`)
      })
    }
  })

  it('leaves content without a placeholder untouched', () => {
    const noCredential = '# Agent Shaker\n\nNo credentials here.\n'
    expect(injectSecret(noCredential, 'ask_abc')).toBe(noCredential)
  })

  it('returns the original when there is no secret', () => {
    // The placeholder survives rather than being replaced with an empty
    // string: a config with `Bearer ` is a config that fails at runtime, while
    // one with the placeholder still tells the operator what is missing.
    expect(injectSecret(config, '')).toBe(config)
  })

  it('returns the original for empty content', () => {
    expect(injectSecret('', 'ask_abc')).toBe('')
  })
})

describe('API_KEY_PLACEHOLDER', () => {
  it('names the environment variable rather than the key', () => {
    // The literal text, not an interpolated value: this string is what a
    // generated file contains and what the browser searches for.
    expect(API_KEY_PLACEHOLDER).toBe('${env:AGENT_SHAKER_API_KEY}')
    expect(API_KEY_ENV_VAR).toBe('AGENT_SHAKER_API_KEY')
  })
})

describe('fileLocation', () => {
  it('prefers the repository filename', () => {
    expect(fileLocation({ filename: '.claude/mcp.json', target: '~/.codex/mcp.json' }))
      .toBe('.claude/mcp.json')
  })

  it('falls back to a user-level target', () => {
    expect(fileLocation({ target: '~/.codex/mcp.json' })).toBe('~/.codex/mcp.json')
  })

  it('has a last resort rather than returning empty', () => {
    expect(fileLocation({})).toBe('(unnamed)')
  })
})