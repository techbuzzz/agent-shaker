import fs from 'fs'
import path from 'path'
import { asset } from './config'

// Path to screenshots in public folder
const SCREENSHOTS_DIR = path.join(process.cwd(), 'public', 'screenshots')

export interface Screenshot {
  src: string
  name: string
  title: string
  description: string
  width?: number
  height?: number
}

// Metadata for known screenshots
const SCREENSHOT_META: Record<string, { title: string; description: string }> = {
  'main-dashboard-ui.jpeg': {
    title: 'Main Dashboard',
    description:
      'The main dashboard view showing projects, agents, and tasks overview with real-time status updates.',
  },
  'project-ui.jpeg': {
    title: 'Project View',
    description:
      'Detailed project view with task list, agent assignments, milestones, and progress tracking.',
  },
  'project-context-from-agent-ui.jpeg': {
    title: 'Agent Context View',
    description:
      'Shared markdown context visible from an AI agent — decisions, notes, and project knowledge base.',
  },
  'agent-setup-ui.jpeg': {
    title: 'Agent Setup',
    description:
      'Connect an AI agent configuration screen — generate MCP config, VS Code settings, and setup instructions.',
  },
  'projects-list-ui.jpeg': {
    title: 'Projects List',
    description:
      'Browse every project, search by name, and jump into the detail view from one card per project.',
  },
  'project-overview-ui.jpeg': {
    title: 'Project Overview',
    description:
      'Single-project snapshot — agent and task counts, plus a one-click path to the MCP agent config bundle.',
  },
  'project-tasks-ui.jpeg': {
    title: 'Project Tasks',
    description:
      'Per-project task list with priority, status, and assignee — change any field inline without leaving the page.',
  },
  'agents-list-ui.jpeg': {
    title: 'Agents List',
    description:
      'Every agent registered to the server with role, team, project, and online status — search and filter by status.',
  },
  'tasks-list-ui.jpeg': {
    title: 'Tasks List',
    description:
      'Global task backlog grouped by status, filterable by priority and text — the home of every work item in flight.',
  },
  'documentation-ui.jpeg': {
    title: 'Documentation Search',
    description:
      'Full-text search across every project doc — surfaces decisions, notes, and APIs by title, content, and tags.',
  },
  'global-context-ui.jpeg': {
    title: 'Global Context',
    description:
      'Server-wide playbooks and notes that every agent reads via read_resource("global://<title>").',
  },
}

/**
 * Get all screenshots from the public/screenshots directory
 */
export function getScreenshots(): Screenshot[] {
  if (!fs.existsSync(SCREENSHOTS_DIR)) {
    return []
  }

  const files = fs.readdirSync(SCREENSHOTS_DIR)
  const imageExtensions = ['.jpeg', '.jpg', '.png', '.gif', '.webp', '.avif']

  const screenshots: Screenshot[] = files
    .filter((file) =>
      imageExtensions.some((ext) => file.toLowerCase().endsWith(ext))
    )
    .map((file) => {
      const nameWithoutExt = file.replace(/\.[^.]+$/, '')
      const meta = SCREENSHOT_META[file] || {
        title: nameWithoutExt
          .split('-')
          .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
          .join(' '),
        description: `Screenshot: ${nameWithoutExt}`,
      }

      return {
        src: asset(`/screenshots/${file}`),
        name: file,
        title: meta.title,
        description: meta.description,
      }
    })

  return screenshots
}
