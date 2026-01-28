# VS 2026 MCP Setup - Implementation Guide

## Quick Start

### 1. Import the Composable
```javascript
import { useMcpSetup } from '@/composables/useMcpSetup'
```

### 2. Initialize in Your Component
```javascript
const mcpApiUrl = computed(() => {
  return `${window.location.protocol}//${window.location.host}/api`
})

const {
  mcpVS2026Json,      // New: VS 2026 MCP configuration
  mcpConfig,          // Bundle of all configs
  downloadAllMcpFiles,
  copyMcpFilesToProject // New: Direct copy to project
} = useMcpSetup(mcpSetupAgent, project, mcpApiUrl)
```

### 3. Add UI Buttons

#### Option A: Download ZIP
```vue
<button @click="downloadAllMcpFiles(mcpConfig, agent.name)" class="btn-primary">
  📥 Download MCP Setup (ZIP)
</button>
```

#### Option B: Direct Copy to Project
```vue
<button @click="copyToProjectDirectory" class="btn-success">
  📁 Apply to Project Directory
</button>

<script>
const copyToProjectDirectory = async () => {
  const result = await copyMcpFilesToProject(mcpConfig, project.id)
  
  if (result.success) {
    showNotification({
      type: 'success',
      title: '✅ Success',
      message: result.message,
      details: `Files created: ${result.files.join(', ')}`
    })
  } else {
    showNotification({
      type: 'error',
      title: '❌ Error',
      message: result.message
    })
  }
}
</script>
```

## File Structure Created

```
project-root/
├── .mcp.json                      ← VS 2026 main config
├── .vscode/
│   ├── settings.json              ← Environment variables
│   └── mcp.json                   ← VS Code MCP config
├── .github/
│   └── copilot-instructions.md    ← Copilot instructions
└── scripts/
    ├── mcp-agent.ps1              ← PowerShell helper
    └── mcp-agent.sh               ← Bash helper
```

## What Gets Generated

### `.mcp.json` (VS 2026)
- Minimal MCP server configuration
- Server type and URL with project/agent context
- Tools and capabilities exposed dynamically by server
- Extensible structure for future enhancements

### `.vscode/mcp.json` (VS Code)
- Minimal VS Code format
- Server URL with project/agent context
- Tools discovered dynamically through MCP protocol

### `.vscode/settings.json`
- Environment variables for terminal
- `MCP_AGENT_NAME`, `MCP_AGENT_ID`
- `MCP_PROJECT_ID`, `MCP_PROJECT_NAME`
- `MCP_API_URL`

### Helper Scripts
- **PowerShell**: Functions for task management
- **Bash**: Curl-based CLI commands

## Key Features

### 🚀 Auto-Detection
Visual Studio 2026 automatically detects `.mcp.json` in project root - no manual configuration needed.

### 🔄 Flexible Deployment
Choose between:
- **ZIP Download**: For local setup or sharing
- **Direct Copy**: One-click project integration

### 🛡️ Clean Configuration
- No hardcoded tools or capabilities in config
- Dynamic discovery through MCP protocol
- Minimal configuration for easy maintenance
- Extensible structure for custom needs

### 📦 Complete Setup
All necessary files included:
- MCP server config
- IDE configurations
- Helper scripts
- Instructions

## Usage Examples

### Example 1: Setup Modal Dialog
```vue
<template>
  <div class="mcp-setup-modal">
    <h2>Setup Agent MCP Configuration</h2>
    
    <div class="agent-info">
      <p><strong>Agent:</strong> {{ agent.name }} ({{ agent.role }})</p>
      <p><strong>Project:</strong> {{ project.name }}</p>
    </div>

    <div class="options">
      <button @click="downloadZip" class="option-card">
        <div class="icon">📥</div>
        <h3>Download ZIP</h3>
        <p>Get all files in a ZIP archive. Perfect for sharing or local setup.</p>
      </button>

      <button @click="applyDirectly" class="option-card">
        <div class="icon">📁</div>
        <h3>Apply to Project</h3>
        <p>Create files directly in your project. One-click setup.</p>
      </button>
    </div>

    <div class="preview">
      <h3>📄 Configuration Preview</h3>
      <pre><code>{{ JSON.parse(mcpVS2026Json) | json }}</code></pre>
    </div>
  </div>
</template>

<script setup>
const downloadZip = () => {
  downloadAllMcpFiles(mcpConfig, agent.value.name)
}

const applyDirectly = async () => {
  const result = await copyMcpFilesToProject(mcpConfig, project.value.id)
  // Handle result...
}
</script>
```

### Example 2: Direct Integration Button
```vue
<button 
  @click="setupMcp"
  :disabled="!agent || !project"
  class="btn-setup-mcp"
>
  <span v-if="!loading">⚡ Setup MCP for {{ agent?.name }}</span>
  <span v-else>
    <spinner /> Creating configuration files...
  </span>
</button>

<script setup>
const loading = ref(false)

const setupMcp = async () => {
  loading.value = true
  try {
    const result = await copyMcpFilesToProject(mcpConfig, project.id)
    
    if (result.success) {
      emit('success', result)
      // Show success toast
    } else {
      emit('error', result)
      // Show error toast
    }
  } finally {
    loading.value = false
  }
}
</script>
```

## Important Notes

### Backend Requirement
For `copyMcpFilesToProject()` to work, implement the endpoint:
```
POST /api/projects/{projectId}/mcp-files
```

### File Permissions
Ensure the project directory has write permissions for the following paths:
- `.mcp.json`
- `.vscode/`
- `.github/`
- `scripts/`

### VS 2026 Recognition
After applying files:
1. Restart Visual Studio 2026
2. Open project settings
3. MCP server should appear in configuration
4. Connection status should show "Connected"

## Verification Checklist

After setup, verify everything works:

- [ ] `.mcp.json` exists in project root
- [ ] `.vscode/settings.json` has environment variables
- [ ] `.vscode/mcp.json` has server configuration
- [ ] VS 2026 shows MCP server in settings
- [ ] Copilot prompt shows agent identity
- [ ] Can see agent name and project in Copilot chat
- [ ] Task management commands available
- [ ] Context sharing works
- [ ] API endpoints accessible

## Troubleshooting

### VS 2026 doesn't detect MCP
1. Check `.mcp.json` is in project root (not in subdirectory)
2. Verify file is valid JSON: `json-lint .mcp.json`
3. Restart VS 2026
4. Check VS 2026 MCP configuration panel

### Direct copy fails
1. Check browser console for errors
2. Verify API endpoint is running: `curl http://localhost:8080/api/health`
3. Check project ID is correct
4. Ensure you have project write permissions
5. Review network requests in DevTools

### Configuration shows but doesn't work
1. Check MCP server is running
2. Verify URL in `.mcp.json` is correct
3. Test API connectivity: `curl http://localhost:8080/api`
4. Check agent ID matches in configuration

## Next Steps

1. **Test with VS 2026**
   - Open project in VS 2026
   - Verify `.mcp.json` is recognized
   - Try Copilot prompts

2. **Customize Configuration**
   - Extend with additional fields if needed
   - Add explicit tool definitions (optional)
   - Configure custom capabilities (optional)

3. **Share with Team**
   - Download ZIP and share
   - Or commit `.mcp.json` to repository

4. **Monitor Usage**
   - Track API calls
   - Monitor agent activities
   - Review context sharing

## References

- [VS2026_MCP_SETUP.md](./VS2026_MCP_SETUP.md) - Detailed documentation
- [useMcpSetup.js](../web/src/composables/useMcpSetup.js) - Source code
- [MCP_SETUP_QUICK_REFERENCE.md](./MCP_SETUP_QUICK_REFERENCE.md) - Quick reference
