<template>
  <div v-if="show" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="bg-white p-6 rounded-lg max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
      <div class="flex justify-between items-start mb-6">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 bg-gradient-to-br from-purple-500 to-blue-600 rounded-xl flex items-center justify-center">
            <span class="text-white text-lg">⚙️</span>
          </div>
          <div>
            <h3 class="text-xl font-semibold text-gray-900">MCP Setup Files</h3>
            <p class="text-sm text-gray-500">Configure your IDE for agent: {{ agent?.name }}</p>
          </div>
        </div>
        <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600 text-2xl">×</button>
      </div>
      
      <div class="space-y-6">
        <!-- Quick Setup -->
        <div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <h4 class="font-semibold text-blue-900 mb-2">🚀 Quick Setup</h4>
          <p class="text-sm text-blue-800 mb-3">Download all files and extract to your project's root folder:</p>
          <button @click="$emit('download-all')" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors flex items-center gap-2">
            <span>📦</span> Download All Setup Files (.zip)
          </button>
        </div>

        <!-- VS Code Settings -->
        <div class="border border-gray-200 rounded-lg overflow-hidden">
          <div class="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
            <div>
              <h4 class="font-semibold text-gray-900">.vscode/settings.json</h4>
              <p class="text-xs text-gray-500">Environment variables for your workspace</p>
            </div>
            <button @click="$emit('download-file', 'settings')" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
              <span>📥</span> Download
            </button>
          </div>
          <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-48"><code>{{ mcpConfig.mcpSettingsJson }}</code></pre>
        </div>

        <!-- MCP Configuration for VS Code -->
        <div class="border border-blue-200 bg-blue-50/30 rounded-lg overflow-hidden">
          <div class="bg-blue-100 px-4 py-3 flex justify-between items-center border-b border-blue-200">
            <div>
              <h4 class="font-semibold text-blue-900 flex items-center gap-2">
                <span>🔗</span> .vscode/mcp.json
                <span class="text-xs bg-blue-600 text-white px-2 py-0.5 rounded-full">VS Code</span>
              </h4>
              <p class="text-xs text-blue-700">MCP server configuration for VS Code</p>
            </div>
            <button @click="$emit('download-file', 'mcp')" class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
              <span>📥</span> Download
            </button>
          </div>
          <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-64"><code>{{ mcpConfig.mcpVSCodeJson }}</code></pre>
        </div>

        <!-- Visual Studio 2026 Configuration -->
        <div class="border border-purple-200 bg-purple-50/30 rounded-lg overflow-hidden">
          <div class="bg-purple-100 px-4 py-3 flex justify-between items-center border-b border-purple-200">
            <div>
              <h4 class="font-semibold text-purple-900 flex items-center gap-2">
                <span>🔗</span> .mcp.json
                <span class="text-xs bg-purple-600 text-white px-2 py-0.5 rounded-full">Visual Studio 2026</span>
              </h4>
              <p class="text-xs text-purple-700">Complete MCP server configuration - place in project root directory</p>
            </div>
            <button @click="$emit('download-file', '.mcp')" class="bg-purple-600 hover:bg-purple-700 text-white px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
              <span>📥</span> Download
            </button>
          </div>
          <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-64"><code>{{ mcpConfig.mcpVS2026Json }}</code></pre>
        </div>

        <!-- GitHub Copilot Instructions -->
        <div class="border border-gray-200 rounded-lg overflow-hidden">
          <div class="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
            <div>
              <h4 class="font-semibold text-gray-900">.github/copilot-instructions.md</h4>
              <p class="text-xs text-gray-500">Instructions for GitHub Copilot agent identity</p>
            </div>
            <button @click="$emit('download-file', 'copilot')" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
              <span>📥</span> Download
            </button>
          </div>
          <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-48"><code>{{ mcpConfig.mcpCopilotInstructions }}</code></pre>
        </div>

        <!-- PowerShell Helper Script -->
        <div class="border border-gray-200 rounded-lg overflow-hidden">
          <div class="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
            <div>
              <h4 class="font-semibold text-gray-900">scripts/mcp-agent.ps1</h4>
              <p class="text-xs text-gray-500">PowerShell helper script for API interactions</p>
            </div>
            <button @click="$emit('download-file', 'powershell')" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
              <span>📥</span> Download
            </button>
          </div>
          <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-48"><code>{{ mcpConfig.mcpPowerShellScript }}</code></pre>
        </div>

        <!-- Bash Helper Script -->
        <div class="border border-gray-200 rounded-lg overflow-hidden">
          <div class="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
            <div>
              <h4 class="font-semibold text-gray-900">scripts/mcp-agent.sh</h4>
              <p class="text-xs text-gray-500">Bash helper script for Linux/Mac</p>
            </div>
            <button @click="$emit('download-file', 'bash')" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
              <span>📥</span> Download
            </button>
          </div>
          <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-48"><code>{{ mcpConfig.mcpBashScript }}</code></pre>
        </div>
      </div>

      <div class="mt-6 pt-4 border-t border-gray-200">
        <h4 class="font-semibold text-gray-900 mb-2">📖 Setup Instructions</h4>
        <ol class="list-decimal list-inside text-sm text-gray-600 space-y-2">
          <li>Download the setup files using the buttons above or the "Download All" option</li>
          <li>Extract/copy the files to your project's root directory</li>
          <li>Restart VS Code to apply the environment variables</li>
          <li>Start using Copilot with your agent identity!</li>
        </ol>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'McpSetupModal',
  props: {
    show: {
      type: Boolean,
      required: true
    },
    agent: {
      type: Object,
      default: null
    },
    mcpConfig: {
      type: Object,
      required: true
    }
  },
  emits: ['close', 'download-file', 'download-all']
}
</script>
