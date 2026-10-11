/**
 * Single source of truth for site-level deployment constants.
 *
 * `basePath` is declared once in `next.config.js`. Next.js applies it
 * automatically to `next/link` and `next/navigation` hrefs, so those must be
 * written WITHOUT the prefix. `next/image` and plain `<img src>` do NOT get the
 * prefix for free — build absolute public URLs with `asset()` instead, so the
 * deployment prefix lives in exactly one place.
 */

// eslint-disable-next-line @typescript-eslint/no-var-requires
const nextConfig = require('../next.config.js') as { basePath?: string }

/** Deployment prefix, e.g. '/agent-shaker' for a project GitHub Pages site. */
export const BASE_PATH: string = nextConfig.basePath ?? ''

/**
 * Build an absolute URL to a file in `public/`, including the deployment prefix.
 *
 * @example asset('/screenshots/main-dashboard-ui.jpeg')
 * // => '/agent-shaker/screenshots/main-dashboard-ui.jpeg'
 */
export function asset(path: string): string {
  const normalised = path.startsWith('/') ? path : `/${path}`
  return `${BASE_PATH}${normalised}`
}

/**
 * Public origin of the deployed site, used for canonical/OG metadata.
 * Falls back to a same-origin absolute URL when building locally.
 */
export const SITE_URL = 'https://techbuzzz.github.io/agent-shaker'