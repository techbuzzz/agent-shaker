import Link from 'next/link'
import { Github, Heart } from 'lucide-react'

export function Footer() {
  const currentYear = new Date().getFullYear()

  return (
    <footer className="border-t border-slate-200 bg-slate-50 dark:border-slate-800 dark:bg-slate-900/50">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-between gap-4 md:flex-row">
          {/* Left side */}
          <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
            <div className="flex h-7 w-7 items-center justify-center rounded-md bg-gradient-to-br from-primary-500 to-purple-600 text-white">
              <span className="text-xs font-black">AS</span>
            </div>
            <span>Agent Shaker</span>
            <span className="text-slate-400">·</span>
            <span>© {currentYear}</span>
          </div>

          {/* Center - Navigation */}
          <nav className="flex items-center gap-6 text-sm">
            <Link
              href="/agent-shaker/"
              className="text-slate-600 transition-colors hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            >
              Home
            </Link>
            <Link
              href="/agent-shaker/docs/"
              className="text-slate-600 transition-colors hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            >
              Docs
            </Link>
            <Link
              href="/agent-shaker/screenshots/"
              className="text-slate-600 transition-colors hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            >
              Screenshots
            </Link>
          </nav>

          {/* Right side */}
          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1 text-sm text-slate-500 dark:text-slate-500">
              Made with <Heart className="h-3.5 w-3.5 fill-red-500 text-red-500" /> for AI agents
            </span>
            <a
              href="https://github.com/techbuzzz/agent-shaker"
              target="_blank"
              rel="noopener noreferrer"
              className="text-slate-500 transition-colors hover:text-slate-900 dark:text-slate-500 dark:hover:text-white"
              aria-label="GitHub"
            >
              <Github className="h-5 w-5" />
            </a>
          </div>
        </div>
      </div>
    </footer>
  )
}
