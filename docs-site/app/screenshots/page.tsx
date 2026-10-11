import type { Metadata } from 'next'
import Image from 'next/image'
import { getScreenshots } from '@/lib/screenshots'

export const metadata: Metadata = {
  title: 'Screenshots',
  description:
    'Screenshots of the Agent Shaker UI — dashboard, project management, agent setup, and shared contexts.',
}

export default function ScreenshotsPage() {
  const screenshots = getScreenshots()

  return (
    <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
      {/* Header */}
      <div className="mb-12 text-center">
        <h1 className="text-4xl font-bold tracking-tight text-slate-900 sm:text-5xl dark:text-white">
          Screenshots
        </h1>
        <p className="mx-auto mt-4 max-w-2xl text-lg text-slate-600 dark:text-slate-400">
          Explore the Agent Shaker interface — from the main dashboard to agent
          setup and shared context management.
        </p>
      </div>

      {/* Gallery */}
      {screenshots.length > 0 ? (
        <div className="grid gap-8 md:grid-cols-2">
          {screenshots.map((screenshot, index) => (
            <div
              key={screenshot.name}
              className="group relative overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-lg transition-shadow hover:shadow-xl dark:border-slate-800 dark:bg-slate-900"
            >
              {/* Image */}
              <div className="relative aspect-video overflow-hidden bg-slate-100 dark:bg-slate-800">
                <Image
                  src={screenshot.src}
                  alt={screenshot.title}
                  fill
                  className="object-cover transition-transform duration-500 group-hover:scale-105"
                  sizes="(max-width: 768px) 100vw, 50vw"
                  priority={index < 2}
                />
              </div>

              {/* Caption */}
              <div className="p-6">
                <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                  {screenshot.title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-400">
                  {screenshot.description}
                </p>
              </div>

              {/* Number badge */}
              <div className="absolute right-4 top-4 flex h-8 w-8 items-center justify-center rounded-full bg-slate-900/70 text-sm font-medium text-white backdrop-blur-sm">
                {index + 1}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed border-slate-300 bg-slate-50 p-12 text-center dark:border-slate-700 dark:bg-slate-900/50">
          <p className="text-slate-500 dark:text-slate-400">
            No screenshots available yet.
          </p>
        </div>
      )}

      {/* Info box */}
      <div className="mt-16 rounded-2xl border border-primary-200 bg-primary-50 p-8 dark:border-primary-900 dark:bg-primary-950/30">
        <h2 className="text-xl font-semibold text-primary-900 dark:text-primary-100">
          Want to see more?
        </h2>
        <p className="mt-2 text-primary-700 dark:text-primary-300">
          Run Agent Shaker locally with Docker Compose to explore the full
          interface. Check out the{' '}
          <a
            href="/agent-shaker/docs/quickstart/"
            className="font-medium underline hover:no-underline"
          >
            Quick Start guide
          </a>{' '}
          for setup instructions.
        </p>
        <div className="mt-4">
          <code className="block rounded-lg bg-slate-900 px-4 py-3 text-sm text-slate-100">
            <span className="text-green-400">docker</span>{' '}
            <span className="text-blue-400">compose</span> up -d
          </code>
        </div>
      </div>
    </div>
  )
}
