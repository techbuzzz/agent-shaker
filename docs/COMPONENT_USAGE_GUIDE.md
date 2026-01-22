# Component Usage Guide

Quick reference for using the refactored components in other parts of the application.

## AgentModal

**Purpose:** Add or edit agent information

### Props
- `show` (Boolean, required) - Controls modal visibility
- `agent` (Object, optional) - Agent to edit (null for new agent)

### Events
- `@close` - Emitted when modal should close
- `@save` - Emitted with agent data when form is submitted

### Usage Example
```vue
<template>
  <AgentModal 
    :show="showModal"
    :agent="selectedAgent"
    @close="showModal = false"
    @save="handleSave"
  />
</template>

<script setup>
import { ref } from 'vue'
import AgentModal from '@/components/AgentModal.vue'

const showModal = ref(false)
const selectedAgent = ref(null)

const handleSave = (agentData) => {
  console.log('Received agent data:', agentData)
  // agentData contains: { name, role, team, status }
  showModal.value = false
}
</script>
```

---

## TaskModal

**Purpose:** Create or edit tasks

### Props
- `show` (Boolean, required) - Controls modal visibility
- `task` (Object, optional) - Task to edit (null for new task)
- `agents` (Array, required) - List of available agents

### Events
- `@close` - Emitted when modal should close
- `@save` - Emitted with task data when form is submitted

### Usage Example
```vue
<template>
  <TaskModal 
    :show="showModal"
    :task="selectedTask"
    :agents="availableAgents"
    @close="showModal = false"
    @save="handleSave"
  />
</template>

<script setup>
import { ref } from 'vue'
import TaskModal from '@/components/TaskModal.vue'

const showModal = ref(false)
const selectedTask = ref(null)
const availableAgents = ref([
  { id: '1', name: 'John Doe' },
  { id: '2', name: 'Jane Smith' }
])

const handleSave = (taskData) => {
  console.log('Received task data:', taskData)
  // taskData contains: { title, description, agent_id, priority, status? }
  showModal.value = false
}
</script>
```

---

## ContextModal

**Purpose:** Add or edit context/documentation

### Props
- `show` (Boolean, required) - Controls modal visibility
- `context` (Object, optional) - Context to edit (null for new)
- `agents` (Array, required) - List of available agents
- `tasks` (Array, required) - List of available tasks

### Events
- `@close` - Emitted when modal should close
- `@save` - Emitted with context data when form is submitted

### Usage Example
```vue
<template>
  <ContextModal 
    :show="showModal"
    :context="selectedContext"
    :agents="agents"
    :tasks="tasks"
    @close="showModal = false"
    @save="handleSave"
  />
</template>

<script setup>
import { ref } from 'vue'
import ContextModal from '@/components/ContextModal.vue'

const showModal = ref(false)
const selectedContext = ref(null)
const agents = ref([...])
const tasks = ref([...])

const handleSave = (contextData) => {
  console.log('Received context data:', contextData)
  // contextData contains: { title, agent_id, task_id, content, tags[] }
  showModal.value = false
}
</script>
```

---

## ContextViewer

**Purpose:** Display context with rendered markdown

### Props
- `show` (Boolean, required) - Controls modal visibility
- `context` (Object, required) - Context to display
- `agentName` (String, optional) - Name of the agent
- `taskName` (String, optional) - Name of the related task

### Events
- `@close` - Emitted when modal should close

### Usage Example
```vue
<template>
  <ContextViewer
    :show="showViewer"
    :context="contextToView"
    :agent-name="getAgentName(contextToView?.agent_id)"
    :task-name="getTaskName(contextToView?.task_id)"
    @close="showViewer = false"
  />
</template>

<script setup>
import { ref } from 'vue'
import ContextViewer from '@/components/ContextViewer.vue'
import { getAgentName } from '@/utils/dataHelpers'

const showViewer = ref(false)
const contextToView = ref({
  id: '1',
  title: 'API Documentation',
  content: '## Overview\n\nThis is **markdown** content',
  tags: ['api', 'docs'],
  agent_id: '1',
  task_id: '2',
  created_at: '2026-01-22T10:00:00Z'
})
</script>
```

---

## ConfirmModal

**Purpose:** Reusable confirmation dialog

### Props
- `show` (Boolean, required) - Controls modal visibility
- `title` (String, default: 'Confirm Delete') - Dialog title
- `message` (String, required) - Main message to display
- `warning` (String, optional) - Additional warning text
- `confirmText` (String, default: 'Delete') - Confirm button text

### Events
- `@close` - Emitted when cancelled
- `@confirm` - Emitted when confirmed

### Usage Example
```vue
<template>
  <ConfirmModal
    :show="showConfirm"
    title="Delete User"
    :message="`Are you sure you want to delete ${user.name}?`"
    warning="This action cannot be undone."
    confirm-text="Delete User"
    @close="showConfirm = false"
    @confirm="handleDelete"
  />
</template>

<script setup>
import { ref } from 'vue'
import ConfirmModal from '@/components/ConfirmModal.vue'

const showConfirm = ref(false)
const user = ref({ name: 'John Doe' })

const handleDelete = () => {
  // Perform delete action
  console.log('User deleted')
  showConfirm.value = false
}
</script>
```

---

## McpSetupModal

**Purpose:** Display and download MCP setup files

### Props
- `show` (Boolean, required) - Controls modal visibility
- `agent` (Object, required) - Agent for MCP setup
- `mcpConfig` (Object, required) - MCP configuration object

### Events
- `@close` - Emitted when modal should close
- `@download-file` - Emitted with file type to download
- `@download-all` - Emitted to download all files as ZIP

### Usage Example
```vue
<template>
  <McpSetupModal
    :show="showSetup"
    :agent="selectedAgent"
    :mcp-config="mcpConfiguration"
    @close="showSetup = false"
    @download-file="handleDownloadFile"
    @download-all="handleDownloadAll"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
import McpSetupModal from '@/components/McpSetupModal.vue'
import { useMcpSetup, downloadFile, downloadAllMcpFiles } from '@/composables/useMcpSetup'

const showSetup = ref(false)
const selectedAgent = ref({ id: '1', name: 'Agent Smith', role: 'frontend' })
const project = ref({ id: '1', name: 'My Project' })
const apiUrl = 'http://localhost:8080/api'

const mcpConfiguration = computed(() => 
  useMcpSetup(selectedAgent, project, apiUrl)
)

const handleDownloadFile = (fileType) => {
  const config = mcpConfiguration.value
  switch (fileType) {
    case 'settings':
      downloadFile('.vscode-settings.json', config.mcpSettingsJson, 'application/json')
      break
    // ... other cases
  }
}

const handleDownloadAll = () => {
  downloadAllMcpFiles(mcpConfiguration.value, selectedAgent.value.name)
}
</script>
```

---

## Utility Functions

### formatters.js

```javascript
import { formatDate, parseTags, tagsToString, getUniqueTags } from '@/utils/formatters'

// Format date
const formatted = formatDate('2026-01-22T10:00:00Z')
// Output: "1/22/2026, 10:00:00 AM"

// Parse tags
const tags = parseTags('api, documentation, backend')
// Output: ['api', 'documentation', 'backend']

// Tags to string
const tagString = tagsToString(['api', 'docs'])
// Output: "api, docs"

// Get unique tags
const contexts = [
  { tags: ['api', 'docs'] },
  { tags: ['api', 'testing'] }
]
const unique = getUniqueTags(contexts)
// Output: ['api', 'docs', 'testing']
```

### dataHelpers.js

```javascript
import { getAgentName, getTaskTitle, filterContexts } from '@/utils/dataHelpers'

const agents = [{ id: '1', name: 'John Doe' }]
const tasks = [{ id: '1', title: 'Build API' }]

// Get agent name
const name = getAgentName(agents, '1')
// Output: "John Doe"

// Get task title
const title = getTaskTitle(tasks, '1')
// Output: "Build API"

// Filter contexts
const contexts = [
  { title: 'API Docs', content: 'About APIs', tags: ['api'] },
  { title: 'UI Guide', content: 'About UI', tags: ['ui'] }
]
const filtered = filterContexts(contexts, 'api', 'api')
// Output: [{ title: 'API Docs', ... }]
```

---

## MCP Setup Composable

### useMcpSetup.js

```javascript
import { ref } from 'vue'
import { useMcpSetup, downloadFile, downloadAllMcpFiles } from '@/composables/useMcpSetup'

const agent = ref({ id: '1', name: 'Agent', role: 'frontend' })
const project = ref({ id: '1', name: 'Project' })
const apiUrl = 'http://localhost:8080/api'

// Generate all MCP configs
const mcpConfig = useMcpSetup(agent, project, apiUrl)

// Access individual configs
console.log(mcpConfig.mcpSettingsJson.value)
console.log(mcpConfig.mcpVSCodeJson.value)
console.log(mcpConfig.mcpCopilotInstructions.value)
console.log(mcpConfig.mcpPowerShellScript.value)
console.log(mcpConfig.mcpBashScript.value)

// Download single file
downloadFile('settings.json', mcpConfig.mcpSettingsJson.value, 'application/json')

// Download all as ZIP
await downloadAllMcpFiles(mcpConfig, agent.value.name)
```

---

## Tips

1. **Always validate props** - Components expect certain data structures
2. **Handle loading states** - Show loading indicators during async operations
3. **Error handling** - Use try-catch blocks and show user-friendly errors
4. **Clean up** - Close modals and reset state after operations
5. **Reusability** - These components can be used across different views

## Testing

```javascript
import { mount } from '@vue/test-utils'
import AgentModal from '@/components/AgentModal.vue'

describe('AgentModal', () => {
  it('emits save event with agent data', async () => {
    const wrapper = mount(AgentModal, {
      props: { show: true, agent: null }
    })
    
    await wrapper.find('input[type="text"]').setValue('John Doe')
    await wrapper.find('form').trigger('submit')
    
    expect(wrapper.emitted('save')).toBeTruthy()
    expect(wrapper.emitted('save')[0][0]).toEqual({
      name: 'John Doe',
      role: 'frontend',
      team: '',
      status: 'active'
    })
  })
})
```

---

**Happy coding!** 🎉
