const path = require('path')

/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export for GitHub Pages
  output: 'export',

  // Base path for GitHub Pages (repository name)
  basePath: '/agent-shaker',

  // Disable image optimization for static export
  images: {
    unoptimized: true,
  },

  // Trailing slash for better compatibility with static hosting
  trailingSlash: true,

  // Disable server-side features that don't work with static export
  experimental: {
    // appDir is stable in Next.js 14, no need to enable
  },

  // Webpack config to handle markdown files
  webpack: (config) => {
    config.resolve.alias['@'] = path.resolve(__dirname)
    return config
  },
}

module.exports = nextConfig
