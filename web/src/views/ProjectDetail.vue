<template>
  <div class="project-detail">
    <div class="container">
      <div v-if="loading" class="text-center py-12 text-gray-500">Loading project...</div>
      <div v-else-if="error" class="p-4 bg-red-50 text-red-600 rounded-md mb-4">{{ error }}</div>
      
      <div v-else-if="project">
        <div class="flex justify-between items-start mb-8">
          <div class="flex-1">
            <div class="flex items-center gap-4 mb-2">
              <h2 class="text-3xl font-bold text-gray-900">{{ project.name }}</h2>
              <div class="flex items-center gap-2 px-3 py-1.5 bg-slate-100 rounded-full">
                <div :class="['w-2 h-2 rounded-full transition-colors duration-200', isConnected ? 'bg-green-500 animate-pulse' : 'bg-slate-400']"></div>
                <span class="text-xs font-medium text-slate-600">{{ isConnected ? 'Connected' : 'Disconnected' }}</span>
              </div>
            </div>
            <p class="text-gray-600 mt-2">{{ project.description }}</p>
          </div>
          <span :class=" [
            'px-3 py-1 rounded-full text-sm font-semibold uppercase',
            project.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
          ]">{{ project.status }}</span>
        </div>

        <div class="flex gap-0 mb-6 border-b border-gray-200">
          <button 
            :class=" [
              'px-4 py-2 text-gray-600 hover:text-gray-900 border-b-2 border-transparent hover:border-gray-300 transition-colors',
              activeTab === 'agents' ? 'text-blue-600 border-blue-600' : ''
            ]"
            @click="activeTab = 'agents'"
          >
            Agents ({{ agents.length }})
          </button>
          <button 
            :class=" [
              'px-4 py-2 text-gray-600 hover:text-gray-900 border-b-2 border-transparent hover:border-gray-300 transition-colors',
              activeTab === 'tasks' ? 'text-blue-600 border-blue-600' : ''
            ]"
            @click="activeTab = 'tasks'"
          >
            Tasks ({{ tasks.length }})
          </button>
          <button 
            :class=" [
              'px-4 py-2 text-gray-600 hover:text-gray-900 border-b-2 border-transparent hover:border-gray-300 transition-colors',
              activeTab === 'contexts' ? 'text-blue-600 border-blue-600' : ''
            ]"
            @click="activeTab = 'contexts'"
          >
            Contexts ({{ contexts.length }})
          </button>
        </div>

        <!-- Agents Tab -->
        <div v-if="activeTab === 'agents'" class="py-6">
          <div class="flex justify-between items-center mb-6">
            <h3 class="text-xl font-semibold text-gray-900">Project Agents</h3>
            <button @click="showAddAgentModal = true" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">
              + Add Agent
            </button>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div v-for="agent in agents" :key="agent.id" class="bg-white p-6 rounded-lg shadow-sm">
              <div class="flex justify-between items-start mb-4">
                <h4 class="text-lg font-semibold text-gray-900">{{ agent.name }}</h4>
                <div class="flex items-center gap-2">
                  <button @click="openMcpSetup(agent)" class="p-1.5 text-purple-500 hover:text-purple-700 hover:bg-purple-50 rounded transition-colors" title="Download MCP Setup">
                    ⚙️
                  </button>
                  <span :class=" [
                    'px-3 py-1 rounded-full text-sm font-semibold',
                    agent.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                  ]">{{ agent.status }}</span>
                </div>
              </div>
              <div class="space-y-2">
                <p class="text-gray-600"><strong class="font-medium text-gray-900">Role:</strong> <span :class=" [
                  'inline-block px-2 py-1 rounded text-xs font-semibold',
                  agent.role === 'frontend' ? 'bg-blue-100 text-blue-800' : 'bg-pink-100 text-pink-800'
                ]">{{ agent.role }}</span></p>
                <p class="text-gray-600"><strong class="font-medium text-gray-900">Team:</strong> {{ agent.team }}</p>
                <p class="text-gray-600"><strong class="font-medium text-gray-900">Last Seen:</strong> {{ formatDate(agent.last_seen) }}</p>
              </div>
            </div>
          </div>

          <div v-if="agents.length === 0" class="text-center py-12 text-gray-500">
            <p>No agents assigned to this project yet</p>
          </div>
        </div>

        <!-- Tasks Tab -->
        <div v-if="activeTab === 'tasks'" class="py-6">
          <div class="flex justify-between items-center mb-6">
            <h3 class="text-xl font-semibold text-gray-900">Project Tasks</h3>
            <button @click="showAddTaskModal = true" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">
              + Create Task
            </button>
          </div>

          <div class="space-y-4">
            <div v-for="task in tasks" :key="task.id" class="bg-white p-6 rounded-lg shadow-sm">
              <div class="flex justify-between items-start mb-3">
                <h4 class="text-lg font-semibold text-gray-900">{{ task.title }}</h4>
                <div class="flex gap-2">
                  <span :class=" [
                    'px-2 py-1 rounded text-xs font-semibold',
                    task.priority === 'high' ? 'bg-red-100 text-red-800' : 
                    task.priority === 'medium' ? 'bg-yellow-100 text-yellow-800' : 'bg-blue-100 text-blue-800'
                  ]">{{ task.priority }}</span>
                  <span :class=" [
                    'px-2 py-1 rounded text-xs font-semibold',
                    task.status === 'done' ? 'bg-green-100 text-green-800' : 
                    task.status === 'in_progress' ? 'bg-blue-100 text-blue-800' : 
                    task.status === 'pending' ? 'bg-gray-100 text-gray-800' : 'bg-red-100 text-red-800'
                  ]">{{ task.status }}</span>
                </div>
              </div>
              <p class="text-gray-600 mb-4">{{ task.description }}</p>
              <div class="flex justify-between text-sm text-gray-500">
                <span>Agent: {{ getAgentName(task.agent_id) }}</span>
                <span>Created {{ formatDate(task.created_at) }}</span>
              </div>
            </div>
          </div>

          <div v-if="tasks.length === 0" class="text-center py-12 text-gray-500">
            <p>No tasks in this project yet</p>
          </div>
        </div>

        <!-- Contexts Tab -->
        <div v-if="activeTab === 'contexts'" class="py-6">
          <div class="flex justify-between items-center mb-6">
            <h3 class="text-xl font-semibold text-gray-900">Project Documentation / Context</h3>
            <button @click="showAddContextModal = true" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">
              + Add Context
            </button>
          </div>

          <div class="flex gap-4 mb-6">
            <input 
              v-model="contextSearch" 
              type="text" 
              placeholder="Search contexts..." 
              class="px-4 py-2 border border-gray-300 rounded-md flex-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <select v-model="contextTagFilter" class="px-4 py-2 border border-gray-300 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option value="">All Tags</option>
              <option v-for="tag in uniqueTags" :key="tag" :value="tag">
                {{ tag }}
              </option>
            </select>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div v-for="context in filteredContexts" :key="context.id" class="bg-white p-6 rounded-lg shadow-sm">
              <div class="flex justify-between items-start mb-4">
                <h4 class="text-lg font-semibold text-gray-900">{{ context.title }}</h4>
                <div class="flex gap-2">
                  <button @click="viewContext(context)" class="p-2 text-gray-400 hover:text-gray-600 transition-colors" title="View">
                    👁️
                  </button>
                  <button @click="editContext(context)" class="p-2 text-gray-400 hover:text-gray-600 transition-colors" title="Edit">
                    ✏️
                  </button>
                  <button @click="confirmDeleteContext(context)" class="p-2 text-red-500 hover:text-red-700 transition-colors" title="Delete">
                    🗑️
                  </button>
                </div>
              </div>
              <div class="flex flex-wrap gap-2 mb-4">
                <span v-for="tag in context.tags" :key="tag" class="px-2 py-1 bg-gray-100 text-gray-800 rounded text-xs">{{ tag }}</span>
              </div>
              <div class="flex justify-between text-sm text-gray-500">
                <span>Agent: {{ getAgentName(context.agent_id) }}</span>
                <span v-if="context.task_id">Task: {{ getTaskTitle(context.task_id) }}</span>
                <span>{{ formatDate(context.created_at) }}</span>
              </div>
            </div>
          </div>

          <div v-if="filteredContexts.length === 0" class="text-center py-12 text-gray-500">
            <p>{{ contexts.length === 0 ? 'No contexts in this project yet' : 'No contexts match your search' }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Agent Modal -->
    <div v-if="showAddAgentModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showAddAgentModal = false">
      <div class="bg-white p-6 rounded-lg max-w-md w-full mx-4">
        <h3 class="text-xl font-semibold mb-6">Add Agent to Project</h3>
        <form @submit.prevent="handleAddAgent">
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Agent Name</label>
            <input v-model="newAgent.name" type="text" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" required />
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Role</label>
            <select v-model="newAgent.role" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" required>
              <option value="frontend">Frontend</option>
              <option value="backend">Backend</option>
            </select>
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Team</label>
            <input v-model="newAgent.team" type="text" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <button type="button" @click="showAddAgentModal = false" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-4 py-2 rounded-md font-medium transition-colors">
              Cancel
            </button>
            <button type="submit" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">Add Agent</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Add Task Modal -->
    <div v-if="showAddTaskModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showAddTaskModal = false">
      <div class="bg-white p-6 rounded-lg max-w-md w-full mx-4">
        <h3 class="text-xl font-semibold mb-6">Create New Task</h3>
        <form @submit.prevent="handleAddTask">
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Task Title</label>
            <input v-model="newTask.title" type="text" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" required />
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Description</label>
            <textarea v-model="newTask.description" rows="4" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"></textarea>
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Assign to Agent</label>
            <select v-model="newTask.agent_id" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" required>
              <option value="">Select an agent</option>
              <option v-for="agent in agents" :key="agent.id" :value="agent.id">
                {{ agent.name }}
              </option>
            </select>
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Priority</label>
            <select v-model="newTask.priority" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <button type="button" @click="showAddTaskModal = false" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-4 py-2 rounded-md font-medium transition-colors">
              Cancel
            </button>
            <button type="submit" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">Create Task</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Add Context Modal -->
    <div v-if="showAddContextModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeContextModal">
      <div class="bg-white p-6 rounded-lg max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
        <h3 class="text-xl font-semibold mb-6">{{ editingContext ? 'Edit Context' : 'Add Context / Documentation' }}</h3>
        <form @submit.prevent="handleSaveContext">
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Title</label>
            <input v-model="contextForm.title" type="text" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" required />
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Agent</label>
            <select v-model="contextForm.agent_id" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" required>
              <option value="">Select an agent</option>
              <option v-for="agent in agents" :key="agent.id" :value="agent.id">
                {{ agent.name }}
              </option>
            </select>
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Related Task (Optional)</label>
            <select v-model="contextForm.task_id" class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option value="">None</option>
              <option v-for="task in tasks" :key="task.id" :value="task.id">
                {{ task.title }}
              </option>
            </select>
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Content (Markdown)</label>
            <textarea 
              v-model="contextForm.content" 
              rows="12" 
              placeholder="Write your documentation in Markdown format..."
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            ></textarea>
            <small class="text-gray-500 text-sm">Supports Markdown: **bold**, *italic*, [link](url), etc.</small>
          </div>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Tags (comma-separated)</label>
            <input 
              v-model="contextForm.tagsString" 
              type="text" 
              placeholder="api, documentation, backend"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <button type="button" @click="closeContextModal" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-4 py-2 rounded-md font-medium transition-colors">
              Cancel
            </button>
            <button type="submit" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">
              {{ editingContext ? 'Update' : 'Create' }} Context
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- View Context Modal -->
    <div v-if="showViewContextModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showViewContextModal = false">
      <div class="bg-white p-6 rounded-lg max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-start mb-6">
          <h3 class="text-xl font-semibold text-gray-900">{{ viewingContext?.title }}</h3>
          <button @click="showViewContextModal = false" class="text-gray-400 hover:text-gray-600 text-2xl">×</button>
        </div>
        <div>
          <div class="flex justify-between items-center mb-6">
            <div class="flex flex-wrap gap-2">
              <span v-for="tag in viewingContext?.tags" :key="tag" class="px-2 py-1 bg-gray-100 text-gray-800 rounded text-xs">{{ tag }}</span>
            </div>
            <div class="flex gap-4 text-sm text-gray-500">
              <span>Agent: {{ getAgentName(viewingContext?.agent_id) }}</span>
              <span v-if="viewingContext?.task_id">Task: {{ getTaskTitle(viewingContext?.task_id) }}</span>
              <span>{{ formatDate(viewingContext?.created_at) }}</span>
            </div>
          </div>
          <div class="prose max-w-none" v-html="renderMarkdown(viewingContext?.content)"></div>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteConfirm" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showDeleteConfirm = false">
      <div class="bg-white p-6 rounded-lg max-w-sm w-full mx-4">
        <h3 class="text-xl font-semibold mb-4 text-red-600">⚠️ Confirm Delete</h3>
        <p class="text-gray-600 mb-6">Are you sure you want to delete the context "{{ deletingContext?.title }}"?</p>
        <div class="flex justify-end gap-3">
          <button @click="showDeleteConfirm = false" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-4 py-2 rounded-md font-medium transition-colors">
            Cancel
          </button>
          <button @click="handleDeleteContext" class="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-md font-medium transition-colors">
            Delete
          </button>
        </div>
      </div>
    </div>

    <!-- MCP Setup Modal -->
    <div v-if="showMcpSetupModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showMcpSetupModal = false">
      <div class="bg-white p-6 rounded-lg max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-start mb-6">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 bg-gradient-to-br from-purple-500 to-blue-600 rounded-xl flex items-center justify-center">
              <span class="text-white text-lg">⚙️</span>
            </div>
            <div>
              <h3 class="text-xl font-semibold text-gray-900">MCP Setup Files</h3>
              <p class="text-sm text-gray-500">Configure your IDE for agent: {{ mcpSetupAgent?.name }}</p>
            </div>
          </div>
          <button @click="showMcpSetupModal = false" class="text-gray-400 hover:text-gray-600 text-2xl">×</button>
        </div>
        
        <div class="space-y-6">
          <!-- Quick Setup -->
          <div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <h4 class="font-semibold text-blue-900 mb-2">🚀 Quick Setup</h4>
            <p class="text-sm text-blue-800 mb-3">Download all files and extract to your project's root folder:</p>
            <button @click="downloadAllMcpFiles" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors flex items-center gap-2">
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
              <button @click="downloadMcpFile('settings')" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
                <span>📥</span> Download
              </button>
            </div>
            <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-48"><code>{{ mcpSettingsJson }}</code></pre>
          </div>

          <!-- GitHub Copilot Instructions -->
          <div class="border border-gray-200 rounded-lg overflow-hidden">
            <div class="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
              <div>
                <h4 class="font-semibold text-gray-900">.github/copilot-instructions.md</h4>
                <p class="text-xs text-gray-500">Instructions for GitHub Copilot agent identity</p>
              </div>
              <button @click="downloadMcpFile('copilot')" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
                <span>📥</span> Download
              </button>
            </div>
            <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-48"><code>{{ mcpCopilotInstructions }}</code></pre>
          </div>

          <!-- PowerShell Helper Script -->
          <div class="border border-gray-200 rounded-lg overflow-hidden">
            <div class="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
              <div>
                <h4 class="font-semibold text-gray-900">scripts/mcp-agent.ps1</h4>
                <p class="text-xs text-gray-500">PowerShell helper script for API interactions</p>
              </div>
              <button @click="downloadMcpFile('powershell')" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
                <span>📥</span> Download
              </button>
            </div>
            <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-48"><code>{{ mcpPowerShellScript }}</code></pre>
          </div>

          <!-- Bash Helper Script -->
          <div class="border border-gray-200 rounded-lg overflow-hidden">
            <div class="bg-gray-50 px-4 py-3 flex justify-between items-center border-b border-gray-200">
              <div>
                <h4 class="font-semibold text-gray-900">scripts/mcp-agent.sh</h4>
                <p class="text-xs text-gray-500">Bash helper script for Linux/Mac</p>
              </div>
              <button @click="downloadMcpFile('bash')" class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-3 py-1.5 rounded-md text-sm font-medium transition-colors flex items-center gap-1">
                <span>📥</span> Download
              </button>
            </div>
            <pre class="p-4 bg-gray-900 text-green-400 text-sm overflow-x-auto max-h-48"><code>{{ mcpBashScript }}</code></pre>
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
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '../stores/projectStore'
import { useAgentStore } from '../stores/agentStore'
import { useTaskStore } from '../stores/taskStore'
import { useContextStore } from '../stores/contextStore'
import { useWebSocket } from '../composables/useWebSocket'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

export default {
  name: 'ProjectDetail',
  setup() {
    const route = useRoute()
    const projectStore = useProjectStore()
    const agentStore = useAgentStore()
    const taskStore = useTaskStore()
    const contextStore = useContextStore()
    const { connect, disconnect, on, isConnected } = useWebSocket(route.params.id)

    const activeTab = ref('agents')
    const showAddAgentModal = ref(false)
    const showAddTaskModal = ref(false)
    const showAddContextModal = ref(false)
    const showViewContextModal = ref(false)
    const showDeleteConfirm = ref(false)
    const showMcpSetupModal = ref(false)
    const mcpSetupAgent = ref(null)

    const newAgent = ref({ name: '', role: 'frontend', team: '' })
    const newTask = ref({ title: '', description: '', agent_id: '', priority: 'medium' })
    
    const contextForm = ref({
      title: '',
      agent_id: '',
      task_id: '',
      content: '',
      tagsString: ''
    })
    const editingContext = ref(null)
    const viewingContext = ref(null)
    const deletingContext = ref(null)
    
    const contextSearch = ref('')
    const contextTagFilter = ref('')

    const project = computed(() => projectStore.currentProject)
    const agents = computed(() => agentStore.agents)
    const tasks = computed(() => taskStore.tasks)
    const contexts = computed(() => contextStore.contexts)

    const uniqueTags = computed(() => {
      const tags = new Set()
      contexts.value.forEach(context => {
        if (context.tags && Array.isArray(context.tags)) {
          context.tags.forEach(tag => tags.add(tag))
        }
      })
      return Array.from(tags).sort()
    })

    const filteredContexts = computed(() => {
      let filtered = contexts.value

      // Filter by search
      if (contextSearch.value) {
        const search = contextSearch.value.toLowerCase()
        filtered = filtered.filter(context =>
          context.title.toLowerCase().includes(search) ||
          context.content.toLowerCase().includes(search)
        )
      }

      // Filter by tag
      if (contextTagFilter.value) {
        filtered = filtered.filter(context =>
          context.tags && context.tags.includes(contextTagFilter.value)
        )
      }

      return filtered
    })

    // MCP Setup computed properties
    const mcpApiUrl = computed(() => {
      return `${window.location.protocol}//${window.location.host}/api`
    })

    const mcpSettingsJson = computed(() => {
      if (!mcpSetupAgent.value || !project.value) return ''
      return JSON.stringify({
        "terminal.integrated.env.windows": {
          "MCP_AGENT_NAME": mcpSetupAgent.value.name,
          "MCP_AGENT_ID": mcpSetupAgent.value.id,
          "MCP_PROJECT_ID": project.value.id,
          "MCP_PROJECT_NAME": project.value.name,
          "MCP_API_URL": mcpApiUrl.value
        },
        "terminal.integrated.env.linux": {
          "MCP_AGENT_NAME": mcpSetupAgent.value.name,
          "MCP_AGENT_ID": mcpSetupAgent.value.id,
          "MCP_PROJECT_ID": project.value.id,
          "MCP_PROJECT_NAME": project.value.name,
          "MCP_API_URL": mcpApiUrl.value
        },
        "terminal.integrated.env.osx": {
          "MCP_AGENT_NAME": mcpSetupAgent.value.name,
          "MCP_AGENT_ID": mcpSetupAgent.value.id,
          "MCP_PROJECT_ID": project.value.id,
          "MCP_PROJECT_NAME": project.value.name,
          "MCP_API_URL": mcpApiUrl.value
        }
      }, null, 2)
    })

    const mcpCopilotInstructions = computed(() => {
      if (!mcpSetupAgent.value || !project.value) return ''
      return `# Agent Identity and MCP Integration

## Your Identity
- **Agent Name**: ${mcpSetupAgent.value.name}
- **Agent ID**: ${mcpSetupAgent.value.id}
- **Role**: ${mcpSetupAgent.value.role}
- **Team**: ${mcpSetupAgent.value.team || 'Not specified'}
- **Project**: ${project.value.name}
- **Project ID**: ${project.value.id}

## MCP API Configuration
- **API URL**: ${mcpApiUrl.value}

## Your Responsibilities
As the **${mcpSetupAgent.value.role}** agent, you should:
${mcpSetupAgent.value.role === 'frontend' ? `
- Focus on UI/UX implementation
- Work with Vue.js, React, or other frontend frameworks
- Implement responsive designs and accessibility
- Handle client-side state management
` : `
- Focus on API development and backend logic
- Work with databases and data models
- Implement business logic and validations
- Handle server-side security and authentication
`}

## Task Management
When working on tasks, use these API endpoints:

### Get Your Tasks
\`\`\`bash
curl "${mcpApiUrl.value}/agents/${mcpSetupAgent.value.id}/tasks"
\`\`\`

### Update Task Status
\`\`\`bash
curl -X PUT "${mcpApiUrl.value}/tasks/{task_id}/status" \\
  -H "Content-Type: application/json" \\
  -d '{"status": "in_progress"}'
\`\`\`

Status options: \`pending\`, \`in_progress\`, \`done\`, \`blocked\`

### Add Context/Documentation
\`\`\`bash
curl -X POST "${mcpApiUrl.value}/contexts" \\
  -H "Content-Type: application/json" \\
  -d '{
    "project_id": "${project.value.id}",
    "agent_id": "${mcpSetupAgent.value.id}",
    "title": "Implementation Notes",
    "content": "Your documentation here...",
    "tags": ["documentation", "${mcpSetupAgent.value.role}"]
  }'
\`\`\`

## Collaboration Guidelines
1. Always check for existing tasks before starting new work
2. Update task status when you begin and complete work
3. Document important decisions and implementation details
4. Check other agents' contexts to avoid conflicts
`
    })

    const mcpPowerShellScript = computed(() => {
      if (!mcpSetupAgent.value || !project.value) return ''
      return `# MCP Agent Helper Script for PowerShell
# Agent: ${mcpSetupAgent.value.name}
# Project: ${project.value.name}

$MCP_API_URL = "${mcpApiUrl.value}"
$MCP_AGENT_ID = "${mcpSetupAgent.value.id}"
$MCP_PROJECT_ID = "${project.value.id}"

function Get-MyTasks {
    Invoke-RestMethod -Uri "$MCP_API_URL/agents/$MCP_AGENT_ID/tasks" -Method GET
}

function Update-TaskStatus {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TaskId,
        [Parameter(Mandatory=$true)]
        [ValidateSet("pending", "in_progress", "done", "blocked")]
        [string]$Status
    )
    
    $body = @{ status = $Status } | ConvertTo-Json
    Invoke-RestMethod -Uri "$MCP_API_URL/tasks/$TaskId/status" -Method PUT -Body $body -ContentType "application/json"
}

function Add-Context {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Title,
        [Parameter(Mandatory=$true)]
        [string]$Content,
        [string[]]$Tags = @()
    )
    
    $body = @{
        project_id = $MCP_PROJECT_ID
        agent_id = $MCP_AGENT_ID
        title = $Title
        content = $Content
        tags = $Tags
    } | ConvertTo-Json
    
    Invoke-RestMethod -Uri "$MCP_API_URL/contexts" -Method POST -Body $body -ContentType "application/json"
}

function Get-ProjectContexts {
    Invoke-RestMethod -Uri "$MCP_API_URL/projects/$MCP_PROJECT_ID/contexts" -Method GET
}

# Usage examples:
# Get-MyTasks
# Update-TaskStatus -TaskId "task-uuid" -Status "in_progress"
# Add-Context -Title "API Design" -Content "Documentation content..." -Tags @("api", "design")
`
    })

    const mcpBashScript = computed(() => {
      if (!mcpSetupAgent.value || !project.value) return ''
      return `#!/bin/bash
# MCP Agent Helper Script for Bash
# Agent: ${mcpSetupAgent.value.name}
# Project: ${project.value.name}

MCP_API_URL="${mcpApiUrl.value}"
MCP_AGENT_ID="${mcpSetupAgent.value.id}"
MCP_PROJECT_ID="${project.value.id}"

# Get tasks assigned to this agent
get_my_tasks() {
    curl -s "$MCP_API_URL/agents/$MCP_AGENT_ID/tasks" | jq .
}

# Update task status
# Usage: update_task_status <task_id> <status>
# Status: pending, in_progress, done, blocked
update_task_status() {
    local task_id=$1
    local status=$2
    curl -s -X PUT "$MCP_API_URL/tasks/$task_id/status" \\
        -H "Content-Type: application/json" \\
        -d "{\\"status\\": \\"$status\\"}" | jq .
}

# Add context/documentation
# Usage: add_context "Title" "Content" "tag1,tag2"
add_context() {
    local title=$1
    local content=$2
    local tags=$3
    
    curl -s -X POST "$MCP_API_URL/contexts" \\
        -H "Content-Type: application/json" \\
        -d "{
            \\"project_id\\": \\"$MCP_PROJECT_ID\\",
            \\"agent_id\\": \\"$MCP_AGENT_ID\\",
            \\"title\\": \\"$title\\",
            \\"content\\": \\"$content\\",
            \\"tags\\": [\\"$tags\\"]
        }" | jq .
}

# Get project contexts
get_project_contexts() {
    curl -s "$MCP_API_URL/projects/$MCP_PROJECT_ID/contexts" | jq .
}

# Usage examples:
# get_my_tasks
# update_task_status "task-uuid" "in_progress"
# add_context "API Design" "Documentation content..." "api,design"
`
    })

    onMounted(() => {
      const projectId = route.params.id
      projectStore.fetchProject(projectId)
      agentStore.fetchProjectAgents(projectId)
      taskStore.fetchProjectTasks(projectId)
      contextStore.fetchProjectContexts(projectId)
      
      // Connect to WebSocket for real-time updates
      connect()
      
      // Add listeners for real-time updates
      on('task_update', (data) => {
        console.log('Task update received:', data)
        taskStore.fetchProjectTasks(projectId)
      })
      
      on('agent_update', (data) => {
        console.log('Agent update received:', data)
        agentStore.fetchProjectAgents(projectId)
      })
      
      on('context_added', (data) => {
        console.log('Context added:', data)
        contextStore.fetchProjectContexts(projectId)
      })
    })

    onUnmounted(() => {
      disconnect()
    })

    const handleAddAgent = async () => {
      try {
        await agentStore.createAgent({
          ...newAgent.value,
          project_id: route.params.id
        })
        showAddAgentModal.value = false
        newAgent.value = { name: '', role: 'frontend', team: '' }
      } catch (error) {
        console.error('Failed to add agent:', error)
      }
    }

    const handleAddTask = async () => {
      try {
        await taskStore.createTask({
          ...newTask.value,
          project_id: route.params.id,
          dependencies: []
        })
        showAddTaskModal.value = false
        newTask.value = { title: '', description: '', agent_id: '', priority: 'medium' }
      } catch (error) {
        console.error('Failed to create task:', error)
      }
    }

    const handleSaveContext = async () => {
      try {
        const tags = contextForm.value.tagsString
          .split(',')
          .map(tag => tag.trim())
          .filter(tag => tag.length > 0)

        const contextData = {
          project_id: route.params.id,
          agent_id: contextForm.value.agent_id,
          task_id: contextForm.value.task_id || null,
          title: contextForm.value.title,
          content: contextForm.value.content,
          tags: tags
        }

        if (editingContext.value) {
          await contextStore.updateContext(editingContext.value.id, contextData)
        } else {
          await contextStore.createContext(contextData)
        }

        closeContextModal()
      } catch (error) {
        console.error('Failed to save context:', error)
      }
    }

    const viewContext = (context) => {
      viewingContext.value = context
      showViewContextModal.value = true
    }

    const editContext = (context) => {
      editingContext.value = context
      contextForm.value = {
        title: context.title,
        agent_id: context.agent_id,
        task_id: context.task_id || '',
        content: context.content,
        tagsString: context.tags ? context.tags.join(', ') : ''
      }
      showAddContextModal.value = true
    }

    const confirmDeleteContext = (context) => {
      deletingContext.value = context
      showDeleteConfirm.value = true
    }

    const handleDeleteContext = async () => {
      try {
        await contextStore.deleteContext(deletingContext.value.id)
        showDeleteConfirm.value = false
        deletingContext.value = null
      } catch (error) {
        console.error('Failed to delete context:', error)
      }
    }

    const closeContextModal = () => {
      showAddContextModal.value = false
      editingContext.value = null
      contextForm.value = {
        title: '',
        agent_id: '',
        task_id: '',
        content: '',
        tagsString: ''
      }
    }

    const renderMarkdown = (content) => {
      if (!content) return ''
      const html = marked(content)
      return DOMPurify.sanitize(html)
    }

    const getAgentName = (agentId) => {
      const agent = agents.value.find(a => a.id === agentId)
      return agent ? agent.name : 'Unknown'
    }

    const getTaskTitle = (taskId) => {
      const task = tasks.value.find(t => t.id === taskId)
      return task ? task.title : 'Unknown'
    }

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleString()
    }

    // MCP Setup functions
    const openMcpSetup = (agent) => {
      mcpSetupAgent.value = agent
      showMcpSetupModal.value = true
    }

    const downloadFile = (filename, content, mimeType = 'text/plain') => {
      const blob = new Blob([content], { type: mimeType })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    }

    const downloadMcpFile = (fileType) => {
      switch (fileType) {
        case 'settings':
          downloadFile('settings.json', mcpSettingsJson.value, 'application/json')
          break
        case 'copilot':
          downloadFile('copilot-instructions.md', mcpCopilotInstructions.value, 'text/markdown')
          break
        case 'powershell':
          downloadFile('mcp-agent.ps1', mcpPowerShellScript.value, 'text/plain')
          break
        case 'bash':
          downloadFile('mcp-agent.sh', mcpBashScript.value, 'text/plain')
          break
      }
    }

    const downloadAllMcpFiles = async () => {
      // Using JSZip for creating zip files
      const { default: JSZip } = await import('jszip')
      const zip = new JSZip()
      
      // Add files to zip with proper folder structure
      zip.file('.vscode/settings.json', mcpSettingsJson.value)
      zip.file('.github/copilot-instructions.md', mcpCopilotInstructions.value)
      zip.file('scripts/mcp-agent.ps1', mcpPowerShellScript.value)
      zip.file('scripts/mcp-agent.sh', mcpBashScript.value)
      
      // Add a README
      const readmeContent = `# MCP Setup Files for ${mcpSetupAgent.value.name}

## Contents
- \`.vscode/settings.json\` - VS Code environment variables
- \`.github/copilot-instructions.md\` - GitHub Copilot agent instructions
- \`scripts/mcp-agent.ps1\` - PowerShell helper script
- \`scripts/mcp-agent.sh\` - Bash helper script

## Setup Instructions
1. Extract this zip to your project's root directory
2. Restart VS Code to apply environment variables
3. Start using Copilot with your agent identity!

## Agent Details
- **Name**: ${mcpSetupAgent.value.name}
- **ID**: ${mcpSetupAgent.value.id}
- **Role**: ${mcpSetupAgent.value.role}
- **Project**: ${project.value.name}
- **API URL**: ${mcpApiUrl.value}
`
      zip.file('MCP_SETUP_README.md', readmeContent)
      
      // Generate and download zip
      const content = await zip.generateAsync({ type: 'blob' })
      const agentSlug = mcpSetupAgent.value.name.toLowerCase().replace(/[^a-z0-9]+/g, '-')
      downloadFile(`mcp-setup-${agentSlug}.zip`, content, 'application/zip')
    }

    return {
      project,
      agents,
      tasks,
      contexts,
      loading: computed(() => projectStore.loading),
      error: computed(() => projectStore.error),
      isConnected,
      activeTab,
      showAddAgentModal,
      showAddTaskModal,
      showAddContextModal,
      showViewContextModal,
      showDeleteConfirm,
      showMcpSetupModal,
      mcpSetupAgent,
      newAgent,
      newTask,
      contextForm,
      editingContext,
      viewingContext,
      deletingContext,
      contextSearch,
      contextTagFilter,
      uniqueTags,
      filteredContexts,
      mcpSettingsJson,
      mcpCopilotInstructions,
      mcpPowerShellScript,
      mcpBashScript,
      handleAddAgent,
      handleAddTask,
      handleSaveContext,
      viewContext,
      editContext,
      confirmDeleteContext,
      handleDeleteContext,
      closeContextModal,
      renderMarkdown,
      getAgentName,
      getTaskTitle,
      formatDate,
      openMcpSetup,
      downloadMcpFile,
      downloadAllMcpFiles
    }
  }
}
</script>

<style scoped>
.project-detail {
  min-height: 100vh;
  background: #f5f7fa;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.loading, .error {
  text-align: center;
  padding: 2rem;
  font-size: 1.1rem;
}

.error {
  color: #e74c3c;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 2rem;
  background: white;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.page-header h2 {
  margin: 0 0 0.5rem 0;
  color: #2c3e50;
}

.subtitle {
  color: #7f8c8d;
  margin: 0;
}

.badge {
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.85rem;
  font-weight: 600;
  text-transform: uppercase;
}

.badge.active { background: #d4edda; color: #155724; }
.badge.completed { background: #cce5ff; color: #004085; }
.badge.archived { background: #d6d8db; color: #383d41; }

.tabs {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
  background: white;
  padding: 1rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.tabs button {
  padding: 0.75rem 1.5rem;
  border: none;
  background: transparent;
  color: #7f8c8d;
  font-weight: 500;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.3s;
}

.tabs button:hover {
  background: #ecf0f1;
  color: #2c3e50;
}

.tabs button.active {
  background: #3498db;
  color: white;
}

.tab-content {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.section-header h3 {
  margin: 0;
  color: #2c3e50;
}

.btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.3s;
}

.btn-primary {
  background: #3498db;
  color: white;
}

.btn-primary:hover {
  background: #2980b9;
}

.btn-secondary {
  background: #95a5a6;
  color: white;
}

.btn-secondary:hover {
  background: #7f8c8d;
}

.btn-danger {
  background: #e74c3c;
  color: white;
}

.btn-danger:hover {
  background: #c0392b;
}

.btn-icon {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1.2rem;
  padding: 0.25rem 0.5rem;
  opacity: 0.7;
  transition: opacity 0.3s;
}

.btn-icon:hover {
  opacity: 1;
}

.btn-close {
  background: none;
  border: none;
  font-size: 2rem;
  cursor: pointer;
  color: #95a5a6;
  line-height: 1;
  padding: 0;
}

.btn-close:hover {
  color: #7f8c8d;
}

.agents-grid, .contexts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
}

.agent-card, .context-card {
  background: #f8f9fa;
  padding: 1.5rem;
  border-radius: 8px;
  transition: transform 0.2s, box-shadow 0.2s;
}

.agent-card:hover, .context-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0,0,0,0.1);
}

.agent-header, .context-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.agent-header h4, .context-header h4 {
  margin: 0;
  color: #2c3e50;
}

.context-actions {
  display: flex;
  gap: 0.25rem;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-badge.active { background: #d4edda; color: #155724; }
.status-badge.idle { background: #fff3cd; color: #856404; }
.status-badge.offline { background: #f8d7da; color: #721c24; }

.agent-details p {
  margin: 0.5rem 0;
  color: #7f8c8c;
}

.badge.frontend { background: #e3f2fd; color: #1976d2; }
.badge.backend { background: #f3e5f5; color: #7b1fa2; }
.badge.devops { background: #e8f5e9; color: #388e3c; }

.tasks-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.task-card {
  background: #f8f9fa;
  padding: 1.5rem;
  border-radius: 8px;
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.task-header h4 {
  margin: 0;
  color: #2c3e50;
}

.task-badges {
  display: flex;
  gap: 0.5rem;
}

.priority, .status {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
}

.priority.high { background: #f8d7da; color: #721c24; }
.priority.medium { background: #fff3cd; color: #856404; }
.priority.low { background: #d1e7dd; color: #0f5132; }

.status.pending { background: #cfe2ff; color: #084298; }
.status.in_progress { background: #fff3cd; color: #856404; }
.status.done { background: #d1e7dd; color: #0f5132; }
.status.blocked { background: #f8d7da; color: #721c24; }

.task-footer {
  display: flex;
  justify-content: space-between;
  color: #7f8c8d;
  font-size: 0.85rem;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #dee2e6;
}

.context-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.tag {
  background: #e3f2fd;
  color: #1976d2;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 500;
}

.context-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  color: #7f8c8d;
  font-size: 0.85rem;
}

.search-filter {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.search-input, .filter-select {
  padding: 0.75rem 1rem;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  font-size: 0.95rem;
}

.search-input {
  flex: 1;
}

.filter-select {
  min-width: 200px;
}

.empty-state {
  text-align: center;
  padding: 3rem;
  color: #7f8c8d;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  max-width: 500px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-large {
  max-width: 800px;
}

.modal-small {
  max-width: 400px;
}

.modal h3 {
  margin: 0 0 1.5rem 0;
  color: #2c3e50;
}

.modal-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.modal-header-flex h3 {
  margin: 0;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: #2c3e50;
  font-weight: 500;
}

.form-group input,
.form-group textarea,
.form-group select {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  font-size: 0.95rem;
  font-family: inherit;
}

.form-group textarea {
  resize: vertical;
  font-family: 'Courier New', monospace;
}

.help-text {
  display: block;
  margin-top: 0.5rem;
  color: #7f8c8d;
}

.prose {
  max-width: 100%;
  line-height: 1.6;
}

.prose h1, .prose h2, .prose h3, .prose h4 {
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
  color: #2c3e50;
}

.prose h1 {
  font-size: 1.875rem;
  font-weight: 700;
}

.prose h2 {
  font-size: 1.5rem;
  font-weight: 600;
}

.prose h3 {
  font-size: 1.25rem;
  font-weight: 500;
}

.prose h4 {
  font-size: 1.125rem;
  font-weight: 500;
}

.prose p {
  margin-bottom: 1rem;
  color: #34495e;
}

.prose a {
  color: #3498db;
  text-decoration: underline;
}

.prose img {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
}

.prose pre {
  background: #f8f9fa;
  padding: 1rem;
  border-radius: 6px;
  overflow-x: auto;
}

.prose code {
  font-family: 'Courier New', monospace;
  background: #eef2f3;
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
}

.prose blockquote {
  border-left: 4px solid #3498db;
  padding-left: 1rem;
  margin: 0;
  color: #7f8c8d;
}

.prose ul, .prose ol {
  margin-left: 1.5rem;
  margin-bottom: 1rem;
}

.prose li {
  margin-bottom: 0.5rem;
}
</style>
