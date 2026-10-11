/**
 * Markdown rendering for agent-authored content.
 *
 * Why this is a module and not inline in `MarkdownView.vue`: the sanitiser has
 * to run on **both** sides of the render, and the server side is the one that
 * matters for security. Keeping the logic here makes it a plain function that
 * vitest can exercise in a node environment, without booting Nuxt — same
 * reason `useContextSearch` keeps its state machine out of the page.
 */
import { marked } from 'marked'
import DOMPurify from 'isomorphic-dompurify'

marked.setOptions({ gfm: true, breaks: false })

/**
 * `isomorphic-dompurify` resolves through the `node` condition during SSR
 * (jsdom-backed) and through `browser`/`default` for the client build
 * (`dompurify` directly, no jsdom in the bundle). One import, correct
 * behaviour on both sides.
 */
export interface RenderMarkdownOptions {
  /** Compact rendering hint for dense lists. Reserved for view-level styling. */
  compact?: boolean
}

/**
 * Parse `source` as GitHub-flavoured markdown and return sanitised HTML.
 *
 * Sanitisation is unconditional and happens *before* the string reaches the
 * SSR payload. That ordering is the whole point: the browser-side pass runs
 * only after hydration, so anything that slips through here has already been
 * parsed and executed by the time DOMPurify would get a chance at it.
 *
 * The previous implementation regex-stripped `<script>` and `on*="…"` on the
 * server. That missed the payloads that actually matter for stored content
 * coming from agents — `<img src=x onerror=…>`, single-quoted or unquoted
 * handlers, `javascript:` URLs, `<svg onload>`. Anything authored by an MCP
 * agent is untrusted input.
 */
export function renderMarkdown(source?: string | null): string {
  if (!source) return ''

  // marked.parse is synchronous unless an async extension is configured, and
  // none is: `async: false` makes that a checked contract rather than a cast.
  const raw = marked.parse(source, { async: false }) as string

  return DOMPurify.sanitize(raw, {
    USE_PROFILES: { html: true },
    // Anchors must not gain `target="_blank"` without `rel`, and links must not
    // be able to run script through a `javascript:` URL. DOMPurify already
    // drops the latter; naming the profile explicitly keeps that guaranteed
    // even if the caller later relaxes USE_PROFILES.
    ALLOWED_TAGS: [
      'a', 'b', 'blockquote', 'br', 'code', 'del', 'em', 'h1', 'h2', 'h3', 'h4',
      'h5', 'h6', 'hr', 'i', 'img', 'li', 'ol', 'p', 'pre', 's', 'strong',
      'sub', 'sup', 'table', 'tbody', 'td', 'th', 'thead', 'tr', 'ul'
    ],
    ALLOWED_ATTR: [
      'href', 'title', 'alt', 'src', 'width', 'height', 'align', 'class'
    ],
    // `data:` in images is the remaining XSS-adjacent vector (SVG data URLs can
    // carry script in some engines), so restrict the scheme to http(s).
    ALLOWED_URI_REGEXP: /^(?:https?:|mailto:|#|\/)/i,
    FORBID_TAGS: ['style', 'form', 'input', 'button', 'iframe', 'object', 'embed', 'script'],
    FORBID_ATTR: ['style', 'target']
  })
}
