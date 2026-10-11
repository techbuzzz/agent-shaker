import type { Metadata } from 'next'
import Link from 'next/link'
import { BookOpen, FileText, FolderOpen, ArrowRight } from 'lucide-react'
import { getDocsByCategory, getFeaturedDocs } from '@/lib/docs'

export const metadata: Metadata = {
  title: 'Documentation',
  description:
    'Complete documentation for Agent Shaker — deployment, API reference, MCP setup, architecture, and more.',
}

export default function DocsIndexPage() {
  const docsByCategory = getDocsByCategory()
  const featuredDocs = getFeaturedDocs()
  const categories = Object.keys(docsByCategory).sort()
  const totalDocs = Object.values(docsByCategory).flat().length

  return (
    <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
      {/* Header */}
      <div className="mb-12">
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary-100 text-primary-600 dark:bg-primary-950 dark:text-primary-400">
            <BookOpen className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-4xl font-bold tracking-tight text-slate-900 dark:text-white">
              Documentation
            </h1>
            <p className="mt-1 text-slate-600 dark:text-slate-400">
              {totalDocs} documents across {categories.length} categories
            </p>
          </div>
        </div>
      </div>

      {/* Featured Docs */}
      <section className="mb-16">
        <h2 className="mb-6 text-2xl font-bold text-slate-900 dark:text-white">
          ⭐ Featured Documentation
        </h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {featuredDocs.map((doc) => (
            <Link
              key={doc.slug}
              href={`/docs/${doc.slug}/`}
              className="card group relative overflow-hidden"
            >
              <div className="absolute right-0 top-0 h-24 w-24 translate-x-8 -translate-y-8 rounded-full bg-primary-500/10 transition-transform group-hover:scale-150" />
              <div className="relative">
                <div className="mb-3 flex items-center gap-2">
                  <FileText className="h-5 w-5 text-primary-500" />
                  <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-400">
                    {doc.category}
                  </span>
                </div>
                <h3 className="mb-2 font-semibold text-slate-900 group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400">
                  {doc.title}
                </h3>
                <p className="text-sm text-slate-500 dark:text-slate-500">
                  {doc.fileName}.md
                </p>
                <div className="mt-4 flex items-center gap-1 text-sm font-medium text-primary-600 opacity-0 transition-opacity group-hover:opacity-100 dark:text-primary-400">
                  Read documentation
                  <ArrowRight className="h-3.5 w-3.5" />
                </div>
              </div>
            </Link>
          ))}
        </div>
      </section>

      {/* All Docs by Category */}
      <section>
        <h2 className="mb-6 text-2xl font-bold text-slate-900 dark:text-white">
          📚 All Documentation
        </h2>
        <div className="space-y-8">
          {categories.map((category) => (
            <div
              key={category}
              className="rounded-2xl border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900"
            >
              {/* Category header */}
              <div className="flex items-center gap-3 border-b border-slate-200 px-6 py-4 dark:border-slate-800">
                <FolderOpen className="h-5 w-5 text-primary-500" />
                <h3 className="font-semibold text-slate-900 dark:text-white">
                  {category}
                </h3>
                <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-400">
                  {docsByCategory[category].length}
                </span>
              </div>

              {/* Category docs */}
              <div className="divide-y divide-slate-100 dark:divide-slate-800/50">
                {docsByCategory[category].map((doc) => (
                  <Link
                    key={doc.slug}
                    href={`/docs/${doc.slug}/`}
                    className="group flex items-center justify-between px-6 py-4 transition-colors hover:bg-slate-50 dark:hover:bg-slate-800/50"
                  >
                    <div className="flex items-center gap-4">
                      <FileText className="h-5 w-5 shrink-0 text-slate-400 group-hover:text-primary-500" />
                      <div>
                        <h4 className="font-medium text-slate-900 group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400">
                          {doc.title}
                        </h4>
                        <p className="text-sm text-slate-500 dark:text-slate-500">
                          {doc.relativePath}
                        </p>
                      </div>
                    </div>
                    <ArrowRight className="h-4 w-4 shrink-0 text-slate-400 opacity-0 transition-opacity group-hover:opacity-100" />
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
