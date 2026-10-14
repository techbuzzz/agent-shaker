import fs from 'fs'
import path from 'path'
import matter from 'gray-matter'
import { remark } from 'remark'
import html from 'remark-html'
import remarkGfm from 'remark-gfm'

// Path to the main docs folder (relative to docs-site)
const DOCS_DIR = path.join(process.cwd(), '..', 'docs')

// Directories under docs/ that are not product documentation.
//
// `releases/` is the release-note archive: a release note is the historical
// record of a given version, so it is kept in the repository but is not part of
// the documentation a visitor is reading to evaluate the product. Excluding the
// directory rather than three filenames means a future release note needs no
// edit here.
const EXCLUDED_DIRS = new Set(['releases'])

// Individual pages under docs/ that must not be published.
//
// This is an explicit list on purpose. The alternative — a broad glob such as
// `*_SUMMARY.md` or `*_REPORT.md` — looks tidier and is wrong: it silently
// swallows a genuinely new page that happens to match, and nobody notices
// because a missing page is invisible in a build log. An explicit list forces a
// deliberate decision per page, and anything genuinely new under docs/ keeps
// publishing without touching this file.
//
// The three groups, and why each is excluded:
//
//   1. Superseded attempts. The VS2026_* set targets a `web/src/` tree that no
//      longer exists; each file carries a SUPERSEDED banner for readers who land
//      on it directly in the repository, but a visitor arriving from search has
//      no way to know that before they have read it.
//   2. Internal work logs — fix reports, verification passes, completion
//      summaries, file-by-file change lists. These are engineering records of
//      what we did, written for us, not documentation of the product. They are
//      the main reason a prospect evaluating the project was reading our
//      internal process instead of our stack.
//   3. Superseded draft specifications. A2A_PROTOCOL_FEATURE.md shipped, but
//      its status line and several stories describe work as future; it is kept
//      as the design record and carries an implementation-status note.
const EXCLUDED_FILES = new Set([
  // 1. Superseded attempts (docs/VS2026_*.md)
  'VS2026_MCP_SETUP.md',
  'VS2026_IMPLEMENTATION_GUIDE.md',
  'VS2026_FEATURE_SUMMARY.md',
  'VS2026_DOCUMENTATION_INDEX.md',
  'VS2026_COMPLETION_SUMMARY.md',

  // 2. Internal work logs
  'AGENTS_PAGE_IMPLEMENTATION.md',
  'A2A_COMPATIBILITY_FIX.md',
  'A2A_EXTERNAL_AGENT_TEST_REPORT.md',
  'API_FIXES_SUMMARY.md',
  'ARCHITECTURE_SUMMARY.md',
  'ARCHITECTURE_VERIFICATION.md',
  'CONTEXT_SHARING_SUMMARY.md',
  'DASHBOARD_IMPLEMENTATION.md',
  'DEPLOYMENT_SUCCESS.md',
  'DELETE_ENDPOINTS_FIX.md',
  'DOCS_VIEWER_COMPLETE.md',
  'DUAL_FRONTEND_SUMMARY.md',
  'FILE_CHANGES.md',
  'IMPLEMENTATION_SUMMARY.md',
  'MCP_COMPOSABLE_INTEGRATION_COMPLETE.md',
  'MCP_ENDPOINTS_FIX.md',
  'MCP_SETUP_COMPLETION_SUMMARY.md',
  'MCP_SETUP_VERIFICATION_SUMMARY.md',
  'MARKDOWN_CONTEXT_SHARING_COMPLETE.md',
  'MCP_SETUP_REVIEW.md',
  'MIGRATION_IMPLEMENTATION.md',
  'NGINX_FIX.md',
  'README_COMPLETION_REPORT.md',
  'README_FINAL_STATUS.md',
  'README_UPDATE_SUMMARY.md',
  'README_VERIFICATION.md',
  'REFACTORING_COMPLETE.md',
  'REFACTORING_SUMMARY.md',
  'TASK_ASSIGNMENT_FIX.md',
  'UPDATE_SUMMARY.md',
  'VALIDATION_REPORT.md',
  'VERIFICATION_CHECKLIST.md',
  'VERIFICATION_REPORT_v0.3.5.md',
  'VUE_ACTUALIZATION_SUMMARY.md',
  'VUE_SETUP_COMPLETE.md',
  'VUE_VERIFICATION_REPORT.md',
  'WEBSOCKET_FIX.md',

  // 3. Superseded draft specifications
  'A2A_PROTOCOL_FEATURE.md',
])

/**
 * Reports whether a file under docs/ is excluded from publication.
 *
 * Takes the path relative to DOCS_DIR. Both parts are normalised to forward
 * slashes because the relative path is built with path.sep, which is a
 * backslash on Windows — matching on the raw string would silently exclude
 * nothing on the platform most contributors build on.
 */
function isExcluded(relativeToDocs: string): boolean {
  const normalized = relativeToDocs.split(path.sep).join('/')
  const segments = normalized.split('/')

  // Check every directory level, so a nested obsolete page is caught too and a
  // nested *new* page still publishes.
  for (let i = 0; i < segments.length - 1; i++) {
    if (EXCLUDED_DIRS.has(segments[i])) return true
  }

  return EXCLUDED_FILES.has(segments[segments.length - 1])
}

export interface DocMeta {
  slug: string
  title: string
  description?: string
  category?: string
  fileName: string
  relativePath: string
}

export interface DocContent {
  meta: DocMeta
  content: string // HTML content
  rawContent: string // Raw markdown
}

/**
 * Get all markdown files from the docs directory recursively
 *
 * This is the single choke point for page collection: getAllDocsMeta (the
 * index, the featured list, the search, and generateStaticParams) and
 * getDocBySlug all read through here. Applying the exclusion list here means an
 * excluded page is not listed, not searchable, not statically generated, and
 * not resolvable by slug — rather than being linked and then 404ing.
 */
export function getAllDocFiles(): string[] {
  const files: string[] = []

  function walkDir(dir: string, baseDir: string = dir) {
    if (!fs.existsSync(dir)) return

    const entries = fs.readdirSync(dir, { withFileTypes: true })
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name)
      if (entry.isDirectory()) {
        walkDir(fullPath, baseDir)
      } else if (entry.name.endsWith('.md')) {
        const relativePath = path.relative(baseDir, fullPath)
        if (isExcluded(relativePath)) continue
        files.push(relativePath)
      }
    }
  }

  walkDir(DOCS_DIR)
  return files
}

/**
 * Extract metadata from a markdown file
 */
function extractMeta(filePath: string, fileName: string): DocMeta {
  const slug = fileName
    .replace(/\.md$/, '')
    .toLowerCase()
    .replace(/_/g, '-')
    .replace(/\s+/g, '-')

  // Determine category from path
  const relativePath = path.relative(DOCS_DIR, filePath)
  const parts = relativePath.split(path.sep)
  const category = parts.length > 1 ? parts[0] : 'general'

  // Generate a readable title from filename
  const title = fileName
    .replace(/\.md$/, '')
    .replace(/_/g, ' ')
    .replace(/-/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())

  return {
    slug,
    title,
    fileName: fileName.replace(/\.md$/, ''),
    category: category.charAt(0).toUpperCase() + category.slice(1),
    relativePath: relativePath.replace(/\\/g, '/'),
  }
}

/**
 * Get all docs metadata (for listing/index pages)
 * Only includes top-level docs and docs from subdirectories
 */
export function getAllDocsMeta(): DocMeta[] {
  const files = getAllDocFiles()
  const docs: DocMeta[] = []

  for (const file of files) {
    const filePath = path.join(DOCS_DIR, file)
    const fileName = path.basename(file)
    const meta = extractMeta(filePath, fileName)
    docs.push(meta)
  }

  // Sort by category, then by title
  return docs.sort((a, b) => {
    if (a.category !== b.category) {
      return a.category!.localeCompare(b.category!)
    }
    return a.title.localeCompare(b.title)
  })
}

/**
 * Get a specific doc by slug (filename without extension, lowercased)
 * Searches recursively in the docs directory
 */
export function getDocBySlug(slug: string): DocContent | null {
  const files = getAllDocFiles()

  // Normalize slug for comparison
  const normalizedSlug = slug.toLowerCase().replace(/-/g, '_')

  for (const file of files) {
    const fileName = path.basename(file)
    const fileSlug = fileName
      .replace(/\.md$/, '')
      .toLowerCase()
      .replace(/_/g, '-')
      .replace(/\s+/g, '-')

    // Also try matching with original underscores
    const fileSlugUnderscore = fileName
      .replace(/\.md$/, '')
      .toLowerCase()

    if (
      fileSlug === slug.toLowerCase() ||
      fileSlugUnderscore === normalizedSlug ||
      fileSlug === normalizedSlug
    ) {
      const filePath = path.join(DOCS_DIR, file)
      const fileContent = fs.readFileSync(filePath, 'utf-8')
      const { data, content } = matter(fileContent)

      const meta = extractMeta(filePath, fileName)

      // Override with frontmatter if present
      if (data.title) meta.title = data.title as string
      if (data.description) meta.description = data.description as string
      if (data.category) meta.category = data.category as string

      // Convert markdown to HTML
      const processedContent = remark()
        .use(remarkGfm)
        .use(html, { sanitize: false })
        .processSync(content)

      return {
        meta,
        content: processedContent.toString(),
        rawContent: content,
      }
    }
  }

  return null
}

/**
 * Get docs grouped by category
 */
export function getDocsByCategory(): Record<string, DocMeta[]> {
  const docs = getAllDocsMeta()
  const grouped: Record<string, DocMeta[]> = {}

  for (const doc of docs) {
    const category = doc.category || 'General'
    if (!grouped[category]) {
      grouped[category] = []
    }
    grouped[category].push(doc)
  }

  return grouped
}

/**
 * Get featured/main documentation files (most important ones)
 */
export function getFeaturedDocs(): DocMeta[] {
  const featuredSlugs = [
    'README',
    'QUICKSTART',
    'QUICKSTART_GUIDE',
    'DEPLOYMENT',
    'API',
    'ARCHITECTURE',
    'MCP_SETUP',
    'AUTHENTICATION',
    'MIGRATIONS',
    'CONTRIBUTING',
    'ROADMAP',
  ]

  const allDocs = getAllDocsMeta()
  const featured: DocMeta[] = []

  for (const slug of featuredSlugs) {
    const doc = allDocs.find(
      (d) => d.fileName.toUpperCase() === slug.toUpperCase()
    )
    if (doc) {
      featured.push(doc)
    }
  }

  return featured
}

/**
 * Search docs by title or content
 */
export function searchDocs(query: string): DocMeta[] {
  const docs = getAllDocsMeta()
  const lowerQuery = query.toLowerCase()

  return docs.filter(
    (doc) =>
      doc.title.toLowerCase().includes(lowerQuery) ||
      doc.fileName.toLowerCase().includes(lowerQuery) ||
      (doc.description?.toLowerCase().includes(lowerQuery) ?? false)
  )
}
