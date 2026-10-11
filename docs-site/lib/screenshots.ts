import fs from 'fs'
import path from 'path'

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
        src: `/agent-shaker/screenshots/${file}`,
        name: file,
        title: meta.title,
        description: meta.description,
      }
    })

  return screenshots
}
