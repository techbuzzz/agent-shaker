# Visual Studio 2026 MCP Configuration Setup

## Overview

Enhanced MCP setup composable with dedicated Visual Studio 2026 support, including automatic `.mcp.json` generation and direct project directory integration.

## Features

### 1. **VS 2026 Dedicated Configuration** (`mcpVS2026Json`)
- Full-featured MCP server configuration optimized for Visual Studio 2026
- Includes schema reference for VS 2026 compatibility
- Comprehensive tool definitions and endpoints
- Enhanced security and logging configuration

### 2. **Root Directory `.mcp.json`**
- Auto-generated `.mcp.json` file for project root
- Recognized automatically by Visual Studio 2026
- No manual configuration required after extraction
- Includes full environment context (project, agent, capabilities)

### 3. **Direct Project Integration**
- New `copyMcpFilesToProject()` function writes files directly to project
- No need to manually copy or extract files
- Automatic directory structure creation
- API-based file management

### 4. **Multi-IDE Support**
Generates configuration for:
- **Visual Studio 2026** - `.mcp.json` in project root
- **VS Code** - `.vscode/settings.json` and `.vscode/mcp.json`
- **Command Line** - PowerShell and Bash scripts
- **GitHub Copilot** - `.github/copilot-instructions.md`

## Configuration Files Generated

### `.mcp.json` (Visual Studio 2026 - Root Directory)
```json
{
  "$schema": "https://aka.ms/mcp-server-schema",
  "version": "1.0.0",
  "servers": {
    "agent-shaker": {
      "type": "http",
      "url": "http://localhost:8080?project_id=...&agent_id=...",
      "name": "Agent Shaker MCP Server",
      "capabilities": ["resources", "tools", "prompts", "context-sharing"],
      "tools": [...],
      "resources": {...},
      "security": {...},
      "logging": {...}
    }
  }
}
```

### Additional Files
- `.vscode/settings.json` - VS Code environment variables
- `.vscode/mcp.json` - VS Code MCP server configuration
- `.github/copilot-instructions.md` - Copilot instructions
- `scripts/mcp-agent.ps1` - PowerShell helper
- `scripts/mcp-agent.sh` - Bash helper

## Usage

### Option 1: Download as ZIP
```javascript
import { downloadAllMcpFiles } from '@/composables/useMcpSetup'

// In your component
downloadAllMcpFiles(mcpConfig, agentName)
// Downloads: mcp-setup-{agent-name}.zip
```

### Option 2: Copy Directly to Project
```javascript
import { copyMcpFilesToProject } from '@/composables/useMcpSetup'

// In your component
const result = await copyMcpFilesToProject(mcpConfig, projectId)

if (result.success) {
  console.log('Files created:', result.files)
} else {
  console.error('Error:', result.message)
}
```

## Implementation in Components

### Using the Composable
```vue
<script setup>
import { computed } from 'vue'
import { useMcpSetup } from '@/composables/useMcpSetup'

const mcpApiUrl = computed(() => {
  return `${window.location.protocol}//${window.location.host}/api`
})

const {
  mcpSettingsJson,
  mcpCopilotInstructions,
  mcpPowerShellScript,
  mcpBashScript,
  mcpVSCodeJson,
  mcpVS2026Json,      // New: VS 2026 configuration
  mcpConfig,
  downloadFile,
  downloadAllMcpFiles,
  copyMcpFilesToProject // New: Direct copy function
} = useMcpSetup(mcpSetupAgent, project, mcpApiUrl)
</script>
```

### Download All Files Button
```vue
<button @click="downloadAll">
  📥 Download All MCP Files
</button>

<script>
const downloadAll = async () => {
  await downloadAllMcpFiles(mcpConfig, agentName)
}
</script>
```

### Direct Copy Button
```vue
<button @click="copyToProject">
  📁 Apply to Project Directory
</button>

<script>
const copyToProject = async () => {
  const result = await copyMcpFilesToProject(mcpConfig, projectId)
  if (result.success) {
    showSuccess(`✅ ${result.message}`)
  } else {
    showError(`❌ ${result.message}`)
  }
}
</script>
```

## Visual Studio 2026 Setup Flow

1. **Generate Configuration**
   - User selects agent and project
   - System generates `.mcp.json`

2. **Two Options for Installation**
   - **Option A**: Download ZIP and extract to project root
   - **Option B**: Click "Apply to Project" button for direct copy

3. **Auto-Recognition**
   - VS 2026 automatically detects `.mcp.json`
   - Loads MCP server configuration
   - Establishes connection to Agent Shaker

4. **Ready to Use**
   - Copilot features enabled
   - Agent identity assigned
   - Task management active
   - Context sharing available

## Configuration Structure

### VS 2026 Config Includes
```
✓ Server metadata and schema
✓ HTTP connection details
✓ Project information
✓ Agent identity and capabilities
✓ 10+ task management tools
✓ Resources and API endpoints
✓ Security configuration
✓ Logging setup
✓ Auto-reconnect behavior
✓ Health check configuration
```

## API Endpoint Required

For `copyMcpFilesToProject()` to work, backend should implement:

```
POST /api/projects/{projectId}/mcp-files

Request Body:
{
  "files": {
    ".mcp.json": "...",
    ".vscode/settings.json": "...",
    ".vscode/mcp.json": "...",
    ...
  }
}

Response:
{
  "success": true,
  "files": [...]
}
```

## Benefits

✨ **No Manual Configuration** - Automatic setup
🚀 **IDE Native** - Works with VS 2026 natively
📦 **Complete Setup** - All files generated automatically
🔄 **Flexible Deployment** - Download or direct copy options
🛡️ **Secure** - Includes security and logging config
📝 **Well Documented** - Comprehensive comments and JSDoc
✅ **Error Handling** - Proper error messages and recovery
🔌 **Future Proof** - Extensible architecture

## Version Info

- **Composable Version**: 2.0.0
- **VS 2026 Support**: ✅ Full
- **VS Code Support**: ✅ Full
- **CLI Support**: ✅ Full

## Related Files

- [useMcpSetup.js](../web/src/composables/useMcpSetup.js) - Main composable
- [ProjectDetail.vue](../web/src/views/ProjectDetail.vue) - Component usage
- [MCP_SETUP_QUICK_REFERENCE.md](./MCP_SETUP_QUICK_REFERENCE.md) - Quick guide
- [QUICKSTART.md](./QUICKSTART.md) - Getting started

## Troubleshooting

### Files not appearing in VS 2026
- Restart Visual Studio 2026 after file creation
- Check if `.mcp.json` is in project root
- Verify project has MCP server feature enabled

### Direct copy fails
- Check API endpoint availability
- Verify project permissions
- Check network connectivity
- Review browser console for errors

### Configuration not applied
- Clear VS 2026 cache
- Restart IDE
- Verify `.mcp.json` file is valid JSON
- Check MCP server status

## Support

For issues or questions:
1. Check MCP_SETUP_QUICK_REFERENCE.md
2. Review error messages in browser console
3. Verify API endpoint is accessible
4. Check project permissions and configuration
