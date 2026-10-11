import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Header } from '@/components/Header'
import { Footer } from '@/components/Footer'

const inter = Inter({ subsets: ['latin', 'cyrillic'] })

export const metadata: Metadata = {
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
  openGraph: {
    title: 'Agent Shaker — Documentation',
    description:
      'Task coordination for teams of AI agents. Register agents, assign them real work, and let them coordinate through MCP.',
    type: 'website',
    locale: 'en_US',
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
