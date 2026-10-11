/**
 * Post-build guard for the GitHub Pages static export.
 *
 * Verifies that every internal href/src in `out/` is correctly prefixed with
 * basePath exactly once and actually resolves to a file that the export
 * produced. Run after `npm run build`.
 *
 *   node scripts/check-export-links.mjs
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const outDir = path.join(root, 'out')
const { basePath = '' } = (
  await import(pathToFileURL(path.join(root, 'next.config.js')).href)
).default ?? {}

if (!fs.existsSync(outDir)) {
  console.error(`FAIL: export directory not found: ${outDir}`)
  process.exit(1)
}

/**
 * URLs that legitimately do not resolve in the static export.
 *
 * These appear inside documentation *samples* (literal HTML snippets quoted in
 * markdown), not in site navigation. They are illustrative build output from an
 * unrelated project and must not be treated as broken links.
 */
const IGNORED_URL_PREFIXES = [
  '/assets/', // sample Vite build output quoted in docs/NGINX_FIX.md
]

const htmlFiles = []
const walk = (dir) => {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) walk(full)
    else if (entry.name.endsWith('.html')) htmlFiles.push(full)
  }
}
walk(outDir)

const doubledPrefix = []
const missing = []
const unprefixed = []
const seen = new Set()

for (const file of htmlFiles) {
  const html = fs.readFileSync(file, 'utf-8')
  const page = path.relative(outDir, file).replace(/\\/g, '/')

  for (const m of html.matchAll(/(?:href|src)="([^"]+)"/g)) {
    const url = m[1]
    if (!url.startsWith('/')) continue // external, hash, mailto, data:
    if (IGNORED_URL_PREFIXES.some((p) => url.startsWith(p))) continue

    if (basePath && url.startsWith(`${basePath}/${basePath}`)) {
      doubledPrefix.push(`${page} -> ${url}`)
      continue
    }

    const isRouteLink = !url.includes('/_next/')
    if (basePath && isRouteLink && !url.startsWith(basePath)) {
      unprefixed.push(`${page} -> ${url}`)
      continue
    }

    const rel = url.slice(basePath.length).replace(/^\//, '')
    if (rel === '') continue
    // Reference HTML has percent-encoded route segments such as %5Bslug%5D.
    const decoded = decodeURIComponent(rel)
    const target = path.join(outDir, rel)
    const candidates = [target, `${target}.html`, path.join(target, 'index.html')]
    const decodedTarget = path.join(outDir, decoded)
    candidates.push(decodedTarget, `${decodedTarget}.html`, path.join(decodedTarget, 'index.html'))
    if (!candidates.some((c) => fs.existsSync(c) && fs.statSync(c).isFile())) {
      const key = `${url}`
      if (!seen.has(key)) {
        seen.add(key)
        missing.push(`${page} -> ${url}`)
      }
    }
  }
}

const report = (label, items) => {
  console.log(`${label}: ${items.length}`)
  for (const item of items.slice(0, 40)) console.log(`    ${item}`)
  if (items.length > 40) console.log(`    ...and ${items.length - 40} more`)
}

console.log(`Scanned ${htmlFiles.length} HTML files in out/ (basePath='${basePath}')`)
report('DOUBLED basePath', doubledPrefix)
report('MISSING basePath', unprefixed)
report('BROKEN links (404)', missing)

const failed = doubledPrefix.length + unprefixed.length + missing.length
if (failed > 0) {
  console.error(`\nFAIL: ${failed} problem(s) found`)
  process.exit(1)
}
console.log('\nOK: all internal links resolve')