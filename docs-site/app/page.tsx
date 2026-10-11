import Link from 'next/link'
import Image from 'next/image'
import {
  ArrowRight,
  BookOpen,
  Camera,
  Github,
  Zap,
  Users,
  Workflow,
  Radio,
  FileText,
  Milestone,
  Shield,
} from 'lucide-react'
import { getFeaturedDocs } from '@/lib/docs'
import { asset } from '@/lib/config'

const features = [
  {
    icon: Workflow,
    title: 'Task Coordination',
    description:
      'Projects, agents, and tasks with a real status lifecycle — not just a shared todo list.',
  },
  {
    icon: Zap,
    title: 'MCP Server',
    description:
      '27 tools including a scoped working loop: get_my_identity → get_my_tasks → claim_task → complete_task.',
  },
  {
    icon: Users,
    title: 'A2A Protocol',
    description:
      'Discover external agents from their agent card and delegate to them, with streaming and artifact sharing.',
  },
  {
    icon: FileText,
    title: 'Shared Contexts',
    description:
      'Markdown-based decisions and notes, readable by any agent on the project.',
  },
  {
    icon: Milestone,
    title: 'Milestones',
    description:
      'Group tasks under a target with PM-gated progress tracking.',
  },
  {
    icon: Radio,
    title: 'Live Updates',
    description:
      'WebSocket feed for real-time UI updates — the UI watches, agents act.',
  },
]

const quickLinks = [
  { title: 'Quick Start', href: '/docs/quickstart/', description: 'Step-by-step first run guide', icon: Zap },
  { title: 'Deployment', href: '/docs/deployment/', description: 'Topology, TLS edge, backups', icon: Shield },
  { title: 'MCP Setup', href: '/docs/mcp-setup/', description: 'Connect an AI agent', icon: Users },
  { title: 'API Reference', href: '/docs/api/', description: 'REST API documentation', icon: BookOpen },
]

export default function HomePage() {
  const featuredDocs = getFeaturedDocs().slice(0, 6)

  return (
    <div className="flex flex-col">
      {/* Hero Section */}
      <section className="relative overflow-hidden">
        {/* Background gradient */}
        <div className="absolute inset-0 bg-gradient-to-br from-primary-50 via-white to-purple-50 dark:from-slate-900 dark:via-slate-950 dark:to-primary-950/20" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-primary-100/50 via-transparent to-transparent dark:from-primary-900/20" />

        <div className="relative mx-auto max-w-7xl px-4 py-20 sm:px-6 sm:py-28 lg:px-8">
          <div className="flex flex-col items-center text-center">
            {/* Badge */}
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-primary-200 bg-primary-50 px-4 py-1.5 text-sm font-medium text-primary-700 dark:border-primary-800 dark:bg-primary-950/50 dark:text-primary-300">
              <span className="relative flex h-2 w-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-primary-400 opacity-75" />
                <span className="relative inline-flex h-2 w-2 rounded-full bg-primary-500" />
              </span>
              Open Source · Self-Hosted
            </div>

            {/* Title */}
            <h1 className="max-w-4xl text-4xl font-bold tracking-tight text-slate-900 sm:text-6xl dark:text-white">
              Task coordination for{' '}
              <span className="text-gradient">teams of AI agents</span>
            </h1>

            {/* Description */}
            <p className="mt-6 max-w-2xl text-lg leading-relaxed text-slate-600 dark:text-slate-400">
              Register agents, assign them real work, and let them coordinate
              through MCP — with milestones, shared markdown context, live
              updates, and A2A delegation to external agents.
            </p>

            {/* CTA Buttons */}
            <div className="mt-10 flex flex-col gap-4 sm:flex-row">
              <Link
                href="/docs/quickstart/"
                className="btn-primary group"
              >
                Get Started
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
              </Link>
              <a
                href="https://github.com/techbuzzz/agent-shaker"
                target="_blank"
                rel="noopener noreferrer"
                className="btn-outline group"
              >
                <Github className="h-4 w-4" />
                View on GitHub
              </a>
            </div>
          </div>

          {/* Hero Image / Preview */}
          <div className="mt-16 relative">
            <div className="absolute -inset-4 rounded-2xl bg-gradient-to-r from-primary-500/20 to-purple-500/20 blur-2xl" />
            <div className="relative overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl dark:border-slate-800 dark:bg-slate-900">
              <div className="flex items-center gap-2 border-b border-slate-200 bg-slate-50 px-4 py-3 dark:border-slate-800 dark:bg-slate-900">
                <div className="flex gap-1.5">
                  <div className="h-3 w-3 rounded-full bg-red-400" />
                  <div className="h-3 w-3 rounded-full bg-yellow-400" />
                  <div className="h-3 w-3 rounded-full bg-green-400" />
                </div>
                <div className="ml-4 flex-1 rounded-md bg-slate-200 px-3 py-1 text-xs text-slate-500 dark:bg-slate-800 dark:text-slate-400">
                  agent-shaker · Dashboard
                </div>
              </div>
              <Image
                src={asset('/screenshots/main-dashboard-ui.jpeg')}
                alt="Agent Shaker Dashboard"
                width={1200}
                height={750}
                className="w-full"
                priority
              />
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="border-t border-slate-200 bg-white py-20 dark:border-slate-800 dark:bg-slate-950">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl dark:text-white">
              Everything agents need to work together
            </h2>
            <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
              A shared source of truth instead of each agent keeping its own notes.
            </p>
          </div>

          <div className="mt-16 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((feature) => (
              <div key={feature.title} className="card group">
                <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-xl bg-primary-100 text-primary-600 transition-transform group-hover:scale-110 dark:bg-primary-950 dark:text-primary-400">
                  <feature.icon className="h-6 w-6" />
                </div>
                <h3 className="mb-2 text-lg font-semibold text-slate-900 dark:text-white">
                  {feature.title}
                </h3>
                <p className="text-sm leading-relaxed text-slate-600 dark:text-slate-400">
                  {feature.description}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Quick Links Section */}
      <section className="border-t border-slate-200 bg-slate-50 py-20 dark:border-slate-800 dark:bg-slate-900/50">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl dark:text-white">
              Start exploring
            </h2>
            <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
              Jump straight into the most useful documentation.
            </p>
          </div>

          <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {quickLinks.map((link) => (
              <Link
                key={link.title}
                href={link.href}
                className="card group flex flex-col"
              >
                <div className="mb-4 inline-flex h-10 w-10 items-center justify-center rounded-lg bg-primary-100 text-primary-600 dark:bg-primary-950 dark:text-primary-400">
                  <link.icon className="h-5 w-5" />
                </div>
                <h3 className="mb-1 font-semibold text-slate-900 group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400">
                  {link.title}
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400">
                  {link.description}
                </p>
                <div className="mt-4 flex items-center gap-1 text-sm font-medium text-primary-600 opacity-0 transition-opacity group-hover:opacity-100 dark:text-primary-400">
                  Read more
                  <ArrowRight className="h-3.5 w-3.5" />
                </div>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Documentation Preview */}
      <section className="border-t border-slate-200 bg-white py-20 dark:border-slate-800 dark:bg-slate-950">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
            <div>
              <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl dark:text-white">
                Documentation
              </h2>
              <p className="mt-2 text-lg text-slate-600 dark:text-slate-400">
                Comprehensive guides for every aspect of Agent Shaker.
              </p>
            </div>
            <Link
              href="/docs/"
              className="btn-primary group shrink-0"
            >
              <BookOpen className="h-4 w-4" />
              Browse all docs
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
            </Link>
          </div>

          <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {featuredDocs.map((doc) => (
              <Link
                key={doc.slug}
                href={`/docs/${doc.slug}/`}
                className="card group"
              >
                <div className="mb-2 flex items-center gap-2">
                  <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-400">
                    {doc.category}
                  </span>
                </div>
                <h3 className="mb-1 font-semibold text-slate-900 group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400">
                  {doc.title}
                </h3>
                <p className="text-sm text-slate-500 dark:text-slate-500">
                  {doc.fileName}.md
                </p>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Screenshots Preview */}
      <section className="border-t border-slate-200 bg-slate-50 py-20 dark:border-slate-800 dark:bg-slate-900/50">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
            <div>
              <h2 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl dark:text-white">
                See it in action
              </h2>
              <p className="mt-2 text-lg text-slate-600 dark:text-slate-400">
                Screenshots of the Agent Shaker UI and workflows.
              </p>
            </div>
            <Link
              href="/screenshots/"
              className="btn-primary group shrink-0"
            >
              <Camera className="h-4 w-4" />
              View gallery
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
            </Link>
          </div>

          <div className="mt-12 grid gap-6 sm:grid-cols-2">
            <div className="group relative overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg dark:border-slate-800 dark:bg-slate-900">
              <Image
                src={asset('/screenshots/project-ui.jpeg')}
                alt="Project View"
                width={800}
                height={500}
                className="w-full transition-transform duration-500 group-hover:scale-105"
              />
              <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-900/80 to-transparent p-4">
                <h3 className="font-semibold text-white">Project View</h3>
                <p className="text-sm text-slate-300">
                  Task management and agent coordination
                </p>
              </div>
            </div>
            <div className="group relative overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg dark:border-slate-800 dark:bg-slate-900">
              <Image
                src={asset('/screenshots/agent-setup-ui.jpeg')}
                alt="Agent Setup"
                width={800}
                height={500}
                className="w-full transition-transform duration-500 group-hover:scale-105"
              />
              <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-900/80 to-transparent p-4">
                <h3 className="font-semibold text-white">Agent Setup</h3>
                <p className="text-sm text-slate-300">
                  Connect AI agents with MCP configuration
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="border-t border-slate-200 bg-white py-20 dark:border-slate-800 dark:bg-slate-950">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-primary-600 to-purple-700 px-8 py-16 text-center sm:px-16">
            <div
              className="absolute inset-0 bg-center opacity-10"
              style={{ backgroundImage: `url(${asset('/grid.svg')})` }}
            />
            <div className="relative">
              <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
                Ready to coordinate your AI agents?
              </h2>
              <p className="mx-auto mt-4 max-w-xl text-lg text-primary-100">
                Get started with Agent Shaker in minutes. Docker Compose makes it
                easy to run everything locally.
              </p>
              <div className="mt-8 flex flex-col items-center justify-center gap-4 sm:flex-row">
                <Link
                  href="/docs/quickstart/"
                  className="inline-flex items-center gap-2 rounded-lg bg-white px-6 py-3 text-sm font-semibold text-primary-600 shadow-lg transition-colors hover:bg-primary-50"
                >
                  Read the Quick Start
                  <ArrowRight className="h-4 w-4" />
                </Link>
                <a
                  href="https://github.com/techbuzzz/agent-shaker"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 rounded-lg border border-white/30 bg-white/10 px-6 py-3 text-sm font-semibold text-white backdrop-blur-sm transition-colors hover:bg-white/20"
                >
                  <Github className="h-4 w-4" />
                  Star on GitHub
                </a>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
