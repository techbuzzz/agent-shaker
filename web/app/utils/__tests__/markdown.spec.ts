import { describe, expect, it } from 'vitest'
import { renderMarkdown } from '../markdown'

/**
 * These payloads are the ones the previous regex sanitiser let through on the
 * server. They are regression tests for a stored-XSS hole, not a general
 * DOMPurify conformance suite: every case here executed in the SSR payload
 * before hydration when markdown came from an agent.
 */
describe('renderMarkdown', () => {
  it('returns an empty string for empty input', () => {
    expect(renderMarkdown('')).toBe('')
    expect(renderMarkdown(null)).toBe('')
    expect(renderMarkdown(undefined)).toBe('')
  })

  it('keeps ordinary markdown intact', () => {
    const html = renderMarkdown('# Title\n\nSome **bold** and `code`.')

    expect(html).toContain('<h1>Title</h1>')
    expect(html).toContain('<strong>bold</strong>')
    expect(html).toContain('<code>code</code>')
  })

  it('keeps GFM tables, which the context viewer relies on', () => {
    const html = renderMarkdown('| a | b |\n| - | - |\n| 1 | 2 |')

    expect(html).toContain('<table>')
    expect(html).toContain('<th>a</th>')
  })

  it('strips an image error handler that the regex sanitiser allowed', () => {
    const html = renderMarkdown('<img src=x onerror=alert(1)>')

    expect(html).not.toContain('onerror')
    expect(html.toLowerCase()).not.toContain('alert(1)')
  })

  it('strips single-quoted and unquoted event handlers', () => {
    expect(renderMarkdown("<div onclick='steal()'>x</div>")).not.toContain('onclick')
    expect(renderMarkdown('<body onload=steal()>')).not.toContain('onload')
    expect(renderMarkdown('<svg onload=steal()>')).not.toContain('onload')
  })

  it('removes script tags outright', () => {
    const html = renderMarkdown('<script>alert(1)</script>')

    expect(html.toLowerCase()).not.toContain('<script')
    expect(html.toLowerCase()).not.toContain('alert(1)')
  })

  it('drops javascript: URLs', () => {
    const html = renderMarkdown('[click me](javascript:alert(1))')

    expect(html.toLowerCase()).not.toContain('javascript:')
  })

  it('keeps ordinary links and their href', () => {
    const html = renderMarkdown('[docs](https://example.com/docs)')

    expect(html).toContain('href="https://example.com/docs"')
  })

  it('drops embedded frames and inline styles', () => {
    expect(renderMarkdown('<iframe src="https://evil.example"></iframe>')).not.toContain('<iframe')
    expect(renderMarkdown('<div style="position:fixed">x</div>')).not.toContain('position:fixed')
  })

  it('does not let agent content inject a style block', () => {
    const html = renderMarkdown('<style>body{display:none}</style>')

    expect(html.toLowerCase()).not.toContain('<style')
    expect(html.toLowerCase()).not.toContain('display:none')
  })
})
