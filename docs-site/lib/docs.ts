import fs from 'fs'
import path from 'path'
import matter from 'gray-matter'
import { remark } from 'remark'
import html from 'remark-html'
import remarkGfm from 'remark-gfm'

// Path to the main docs folder (relative to docs-site)
const DOCS_DIR = path.join(process.cwd(), '..', 'docs')

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
