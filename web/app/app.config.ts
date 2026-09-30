/**
 * Brand-aligned Nuxt UI theme.
 * - primary: indigo (matches the old "blue → violet" gradient)
 * - neutral: slate (matches the old gray/slate palette)
 * - both light & dark mode via @nuxtjs/color-mode (auto-enabled by @nuxt/ui)
 */
export default defineAppConfig({
  ui: {
    colors: {
      primary: 'indigo',
      neutral: 'slate',
      success: 'emerald',
      info: 'sky',
      warning: 'amber',
      error: 'rose'
    },
    theme: {
      transitions: true,
      defaultVariants: {
        color: 'primary',
        size: 'md'
      }
    }
  }
})
