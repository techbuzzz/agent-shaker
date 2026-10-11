import type { Metadata } from 'next'
import Link from 'next/link'
import { notFound } from 'next/navigation'
import {
  ArrowLeft,
  ArrowRight,
  Calendar,
  ChevronRight,
  FileText,
  FolderOpen,
} from 'lucide-react'
import { getDocBySlug, getAllDocsMeta } from '@/lib/docs'

interface DocPageProps {
  params: {
    slug: string
  }
}

// Generate static params for all docs at build time
export function generateStaticParams() {
  const docs = getAllDocsMeta()
  return docs.map((doc) => ({
    slug: doc.slug,
  }))
}

// Generate metadata for each doc page
export async function generateMetadata({
  params,
}: DocPageProps): Promise<Metadata> {
  const doc = getDocBySlug(params.slug)

  if (!doc) {
    return {
      title: 'Not Found',
    }
  }

  return {
    title: doc.meta.title,
    description: doc.meta.description || `Documentation: ${doc.meta.title}`,
    openGraph: {
      title: doc.meta.title,
      description: doc.meta.description,
      type: 'article',
    },
  }
}

export default function DocPage({ params }: DocPageProps) {
  const doc = getDocBySlug(params.slug)

  if (!doc) {
    notFound()
  }

  // Get all docs for prev/next navigation
  const allDocs = getAllDocsMeta()
  const currentIndex = allDocs.findIndex((d) => d.slug === params.slug)
  const prevDoc = currentIndex > 0 ? allDocs[currentIndex - 1] : null
  const nextDoc =
    currentIndex < allDocs.length - 1 ? allDocs[currentIndex + 1] : null

  return (
    <div className="mx-auto max-w-5xl px-4 py-12 sm:px-6 lg:px-8">
      {/* Breadcrumb */}
      <nav className="mb-8 flex items-center gap-2 text-sm">
        <Link
          href="/docs/"
          className="flex items-center gap-1 text-slate-500 transition-colors hover:text-primary-600 dark:text-slate-400 dark:hover:text-primary-400"
        >
          <FolderOpen className="h-4 w-4" />
          Documentation
        </Link>
        <ChevronRight className="h-4 w-4 text-slate-400" />
        <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-400">
          {doc.meta.category}
        </span>
        <ChevronRight className="h-4 w-4 text-slate-400" />
        <span className="font-medium text-slate-900 dark:text-white">
          {doc.meta.title}
        </span>
      </nav>

      {/* Article header */}
      <header className="mb-10">
        <div className="mb-4 flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary-100 text-primary-600 dark:bg-primary-950 dark:text-primary-400">
            <FileText className="h-6 w-6" />
          </div>
          <div>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {doc.meta.relativePath}
            </p>
          </div>
        </div>
        <h1 className="text-4xl font-bold tracking-tight text-slate-900 sm:text-5xl dark:text-white">
          {doc.meta.title}
        </h1>
        {doc.meta.description && (
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
            {doc.meta.description}
          </p>
        )}
      </header>

      {/* Article content */}
      <article
        className="prose-content"
        dangerouslySetInnerHTML={{ __html: doc.content }}
      />

      {/* Navigation footer */}
      <nav className="mt-16 flex items-center justify-between border-t border-slate-200 pt-8 dark:border-slate-800">
        {prevDoc ? (
          <Link
            href={`/docs/${prevDoc.slug}/`}
            className="group flex items-center gap-3"
          >
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-100 transition-colors group-hover:bg-primary-100 dark:bg-slate-800 dark:group-hover:bg-primary-950">
              <ArrowLeft className="h-5 w-5 text-slate-600 group-hover:text-primary-600 dark:text-slate-400 dark:group-hover:text-primary-400" />
            </div>
            <div className="text-left">
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Previous
              </p>
              <p className="text-sm font-medium text-slate-900 group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400">
                {prevDoc.title}
              </p>
            </div>
          </Link>
        ) : (
          <div />
        )}

        {nextDoc ? (
          <Link
            href={`/docs/${nextDoc.slug}/`}
            className="group flex items-center gap-3"
          >
            <div className="text-right">
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Next
              </p>
              <p className="text-sm font-medium text-slate-900 group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400">
                {nextDoc.title}
              </p>
            </div>
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-slate-100 transition-colors group-hover:bg-primary-100 dark:bg-slate-800 dark:group-hover:bg-primary-950">
              <ArrowRight className="h-5 w-5 text-slate-600 group-hover:text-primary-600 dark:text-slate-400 dark:group-hover:text-primary-400" />
            </div>
          </Link>
        ) : (
          <div />
        )}
      </nav>

      {/* Back to docs link */}
      <div className="mt-8 text-center">
        <Link
          href="/docs/"
          className="inline-flex items-center gap-2 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to all documentation
        </Link>
      </div>
    </div>
  )
}
