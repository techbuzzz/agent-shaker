import Link from 'next/link'
import Image from 'next/image'
import { Github, Heart } from 'lucide-react'
import { asset } from '@/lib/config'

export function Footer() {
  const currentYear = new Date().getFullYear()

  return (
    <footer className="border-t border-slate-200 bg-slate-50 dark:border-slate-800 dark:bg-slate-900/50">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-between gap-4 md:flex-row">
          {/* Left side */}
          <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
            <Image
              src={asset('/icon.png')}
              alt="Agent Shaker"
              width={28}
              height={28}
              className="h-7 w-7 rounded-md"
            />
            <span>Agent Shaker</span>
            <span className="text-slate-400">·</span>
            <span>© {currentYear}</span>
          </div>

          {/* Center - Navigation */}
          <nav className="flex items-center gap-6 text-sm">
            <Link
              href="/"
              className="text-slate-600 transition-colors hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            >
              Home
            </Link>
            <Link
              href="/docs/"
              className="text-slate-600 transition-colors hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            >
              Docs
            </Link>
            <Link
              href="/screenshots/"
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