import type { Metadata, Viewport } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Header } from '@/components/Header'
import { Footer } from '@/components/Footer'
import { asset, SITE_URL } from '@/lib/config'

const inter = Inter({ subsets: ['latin', 'cyrillic'] })

// Next.js 14+: themeColor moved out of `metadata` into its own `viewport` export.
export const viewport: Viewport = {
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#0f172a' },
    { media: '(prefers-color-scheme: dark)', color: '#020617' },
  ],
}

export const metadata: Metadata = {
  // Required so canonical/OG URLs on a subpath deployment resolve absolutely.
  metadataBase: new URL(SITE_URL),
  title: {
    default: 'Agent Shaker — Documentation',
    template: '%s | Agent Shaker',
  },
  description:
    'Task coordination for teams of AI agents. Register agents, assign them real work, and let them coordinate through MCP — with milestones, shared markdown context, live updates, and A2A delegation.',
  keywords: [
    'Agent Shaker',
    'AI agents',
    'MCP',
    'Model Context Protocol',
    'A2A',
    'task coordination',
    'documentation',
  ],
  authors: [{ name: 'Agent Shaker Team' }],
  icons: {
    // SVG is the preferred pick on most browsers; PNG/ICO are emitted alongside
    // so legacy browsers (and the pinned-tab shortcut) get a sized fallback.
    icon: [
      { url: asset('/icon.svg'), type: 'image/svg+xml' },
      { url: asset('/favicon-32x32.png'), sizes: '32x32', type: 'image/png' },
      { url: asset('/favicon-16x16.png'), sizes: '16x16', type: 'image/png' },
      { url: asset('/favicon.ico'), sizes: 'any', type: 'image/x-icon' },
    ],
    apple: [{ url: asset('/apple-touch-icon.png'), sizes: '180x180', type: 'image/png' }],
  },
  openGraph: {
    title: 'Agent Shaker — Documentation',
    description:
      'Task coordination for teams of AI agents. Register agents, assign them real work, and let them coordinate through MCP.',
    url: SITE_URL,
    siteName: 'Agent Shaker',
    type: 'website',
    locale: 'en_US',
    images: [
      {
        // 1280x640 — the size rendered as the GitHub repository social preview
        // when the repo is shared, and the Open Graph fallback for crawlers.
        // Passed as a deployment-relative path; `metadataBase` resolves it
        // against SITE_URL — running `asset()` here would double the basePath.
        url: '/social-preview.png',
        width: 1280,
        height: 640,
        alt: 'Agent Shaker — AI coordination platform for teams of AI agents',
        type: 'image/png',
      },
    ],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Agent Shaker — AI coordination platform',
    description:
      'Task coordination for teams of AI agents. Register agents, assign them real work, and let them coordinate through MCP.',
    images: ['/social-preview.png'],
  },
  robots: {
    index: true,
    follow: true,
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" className="scroll-smooth">
      <body
        className={`${inter.className} bg-white text-slate-900 antialiased dark:bg-slate-950 dark:text-slate-100`}
      >
        <div className="flex min-h-screen flex-col">
          <Header />
          <main className="flex-1">{children}</main>
          <Footer />
        </div>
      </body>
    </html>
  )
}
