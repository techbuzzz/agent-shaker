const API_BASE = window.location.origin + '/api';
const WS_BASE = (window.location.protocol === 'https:' ? 'wss:' : 'ws:') + '//' + window.location.host + '/ws';

let ws = null;
let currentProject = null;
let projects = [];
let agents = [];
let tasks = [];
let contexts = [];

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    setupTabs();
    loadProjects();
    setupWebSocket();
});

// Tab management
function setupTabs() {
    const tabs = document.querySelectorAll('.tab');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const tabName = tab.getAttribute('data-tab');
            switchTab(tabName);
        });
    });
}

function switchTab(tabName) {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
    
    document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');
    document.getElementById(tabName).classList.add('active');
    
    // Load data for the tab
    if (tabName === 'agents' && currentProject) loadAgents(currentProject);
    if (tabName === 'tasks' && currentProject) loadTasks(currentProject);
    if (tabName === 'contexts' && currentProject) loadContexts(currentProject);
}

// WebSocket
function setupWebSocket() {
    if (!currentProject) return;
    
    const wsUrl = `${WS_BASE}?project_id=${currentProject}`;
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
        document.getElementById('wsStatus').textContent = '● Connected';
        document.getElementById('wsStatus').classList.add('online');
        document.getElementById('wsStatus').classList.remove('offline');
    };
    
    ws.onclose = () => {
        document.getElementById('wsStatus').textContent = '● Disconnected';
        document.getElementById('wsStatus').classList.add('offline');
        document.getElementById('wsStatus').classList.remove('online');
        setTimeout(setupWebSocket, 3000);
    };
    
    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        handleWebSocketMessage(data);
    };
}

function handleWebSocketMessage(data) {
    showNotification(`Update: ${data.type}`);
    
    if (data.type === 'task_update') {
        loadTasks(currentProject);
    } else if (data.type === 'agent_update') {
        loadAgents(currentProject);
    } else if (data.type === 'context_added') {
        loadContexts(currentProject);
    }
}

// Projects
async function loadProjects() {
    try {
        const response = await fetch(`${API_BASE}/projects`);
        projects = await response.json() || [];
        renderProjects();
        
        if (projects.length > 0 && !currentProject) {
            currentProject = projects[0].id;
            setupWebSocket();
        }
    } catch (error) {
        console.error('Failed to load projects:', error);
    }
}

function renderProjects() {
    const container = document.getElementById('projectsList');
    if (!projects || projects.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📦</div><p>No projects yet. Create your first project!</p></div>';
        return;
    }
    
    container.innerHTML = projects.map(p => `
        <div class="card" onclick="selectProject('${p.id}')">
            <div class="card-header">
                <div class="card-title">${p.name}</div>
                <div class="card-meta">${new Date(p.created_at).toLocaleDateString()}</div>
            </div>
            <div class="card-body">
                <p>${p.description || 'No description'}</p>
            </div>
            <div class="card-footer">
                <span class="badge ${p.status}">${p.status}</span>
                ${currentProject === p.id ? '<span style="color: #007bff;">● Selected</span>' : ''}
            </div>
        </div>
    `).join('');
}

function selectProject(projectId) {
    currentProject = projectId;
    renderProjects();
    setupWebSocket();
    loadAgents(projectId);
    loadTasks(projectId);
    loadContexts(projectId);
    showNotification('Project selected');
}

async function createProject(event) {
    event.preventDefault();
    
    const name = document.getElementById('projectName').value;
    const description = document.getElementById('projectDescription').value;
    
    try {
        const response = await fetch(`${API_BASE}/projects`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, description })
        });
        
        const project = await response.json();
        projects.push(project);
        currentProject = project.id;
        renderProjects();
        closeModal('projectModal');
        showNotification('Project created successfully');
        setupWebSocket();
    } catch (error) {
        console.error('Failed to create project:', error);
        showNotification('Failed to create project', 'error');
    }
}

// Agents
async function loadAgents(projectId) {
    try {
        const response = await fetch(`${API_BASE}/agents?project_id=${projectId}`);
        agents = await response.json() || [];
        renderAgents();
    } catch (error) {
        console.error('Failed to load agents:', error);
    }
}

function renderAgents() {
    const container = document.getElementById('agentsList');
    if (!agents || agents.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">🤖</div><p>No agents registered. Register your first agent!</p></div>';
        return;
    }
    
    container.innerHTML = agents.map(a => `
        <div class="card">
            <div class="card-header">
                <div class="card-title">${a.name}</div>
                <div class="card-meta">${a.role || 'No role'}</div>
            </div>
            <div class="card-body">
                <p><strong>Team:</strong> ${a.team || 'No team'}</p>
                <p><strong>Last seen:</strong> ${new Date(a.last_seen).toLocaleString()}</p>
            </div>
            <div class="card-footer">
                <span class="badge ${a.status}">${a.status}</span>
            </div>
        </div>
    `).join('');
}

async function createAgent(event) {
    event.preventDefault();
    
    const project_id = document.getElementById('agentProjectId').value;
    const name = document.getElementById('agentName').value;
    const role = document.getElementById('agentRole').value;
    const team = document.getElementById('agentTeam').value;
    
    try {
        await fetch(`${API_BASE}/agents`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ project_id, name, role, team })
        });
        
        loadAgents(project_id);
        closeModal('agentModal');
        showNotification('Agent registered successfully');
    } catch (error) {
        console.error('Failed to create agent:', error);
        showNotification('Failed to register agent', 'error');
    }
}

// Tasks
async function loadTasks(projectId) {
    try {
        const response = await fetch(`${API_BASE}/tasks?project_id=${projectId}`);
        tasks = await response.json() || [];
        renderTasks();
    } catch (error) {
        console.error('Failed to load tasks:', error);
    }
}

function renderTasks() {
    const container = document.getElementById('tasksList');
    if (!tasks || tasks.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📋</div><p>No tasks yet. Create your first task!</p></div>';
        return;
    }
    
    container.innerHTML = tasks.map(t => `
        <div class="card">
            <div class="card-header">
                <div class="card-title">${t.title}</div>
                <div class="card-meta">${new Date(t.created_at).toLocaleDateString()}</div>
            </div>
            <div class="card-body">
                <p>${t.description || 'No description'}</p>
                ${t.output ? `<p><strong>Output:</strong> ${t.output}</p>` : ''}
            </div>
            <div class="card-footer">
                <span class="badge ${t.status}">${t.status}</span>
                <span class="badge ${t.priority}">${t.priority}</span>
            </div>
        </div>
    `).join('');
}

async function createTask(event) {
    event.preventDefault();
    
    const project_id = document.getElementById('taskProjectId').value;
    const title = document.getElementById('taskTitle').value;
    const description = document.getElementById('taskDescription').value;
    const priority = document.getElementById('taskPriority').value;
    const created_by = document.getElementById('taskCreatedBy').value;
    const assigned_to = document.getElementById('taskAssignedTo').value || null;
    
    try {
        await fetch(`${API_BASE}/tasks`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                project_id, 
                title, 
                description, 
                priority, 
                created_by, 
                assigned_to 
            })
        });
        
        loadTasks(project_id);
        closeModal('taskModal');
        showNotification('Task created successfully');
    } catch (error) {
        console.error('Failed to create task:', error);
        showNotification('Failed to create task', 'error');
    }
}

// Contexts
async function loadContexts(projectId) {
    try {
        const response = await fetch(`${API_BASE}/contexts?project_id=${projectId}`);
        contexts = await response.json() || [];
        renderContexts();
    } catch (error) {
        console.error('Failed to load contexts:', error);
    }
}

function renderContexts() {
    const container = document.getElementById('contextsList');
    if (!contexts || contexts.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📚</div><p>No documentation yet. Add your first documentation!</p></div>';
        return;
    }
    
    container.innerHTML = contexts.map(c => `
        <div class="card">
            <div class="card-header">
                <div class="card-title">${c.title}</div>
                <div class="card-meta">${new Date(c.created_at).toLocaleDateString()}</div>
            </div>
            <div class="card-body">
                <p>${c.content.substring(0, 200)}${c.content.length > 200 ? '...' : ''}</p>
            </div>
            <div class="card-footer">
                ${c.tags ? c.tags.map(tag => `<span class="badge">${tag}</span>`).join(' ') : ''}
            </div>
        </div>
    `).join('');
}

async function createContext(event) {
    event.preventDefault();
    
    const project_id = document.getElementById('contextProjectId').value;
    const agent_id = document.getElementById('contextAgentId').value;
    const task_id = document.getElementById('contextTaskId').value || null;
    const title = document.getElementById('contextTitle').value;
    const content = document.getElementById('contextContent').value;
    const tagsStr = document.getElementById('contextTags').value;
    const tags = tagsStr ? tagsStr.split(',').map(t => t.trim()) : [];
    
    try {
        await fetch(`${API_BASE}/contexts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                project_id, 
                agent_id, 
                task_id, 
                title, 
                content, 
                tags 
            })
        });
        
        loadContexts(project_id);
        closeModal('contextModal');
        showNotification('Documentation added successfully');
    } catch (error) {
        console.error('Failed to create context:', error);
        showNotification('Failed to add documentation', 'error');
    }
}

// Modal management
function openProjectModal() {
    document.getElementById('projectModal').classList.add('active');
}

function openAgentModal() {
    if (!currentProject) {
        showNotification('Please select a project first', 'error');
        return;
    }
    populateProjectSelect('agentProjectId');
    document.getElementById('agentModal').classList.add('active');
}

function openTaskModal() {
    if (!currentProject) {
        showNotification('Please select a project first', 'error');
        return;
    }
    populateProjectSelect('taskProjectId');
    loadAgentsForTask();
    document.getElementById('taskModal').classList.add('active');
}

function openContextModal() {
    if (!currentProject) {
        showNotification('Please select a project first', 'error');
        return;
    }
    populateProjectSelect('contextProjectId');
    loadAgentsAndTasksForContext();
    document.getElementById('contextModal').classList.add('active');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('active');
}

// Helper functions
function populateProjectSelect(selectId) {
    const select = document.getElementById(selectId);
    select.innerHTML = projects.map(p => 
        `<option value="${p.id}" ${p.id === currentProject ? 'selected' : ''}>${p.name}</option>`
    ).join('');
}

async function loadAgentsForTask() {
    const projectId = document.getElementById('taskProjectId').value;
    if (!projectId) return;
    
    try {
        const response = await fetch(`${API_BASE}/agents?project_id=${projectId}`);
        const agents = await response.json() || [];
        
        const createdBy = document.getElementById('taskCreatedBy');
        const assignedTo = document.getElementById('taskAssignedTo');
        
        const options = agents.map(a => 
            `<option value="${a.id}">${a.name} (${a.role})</option>`
        ).join('');
        
        createdBy.innerHTML = options;
        assignedTo.innerHTML = '<option value="">Unassigned</option>' + options;
    } catch (error) {
        console.error('Failed to load agents:', error);
    }
}

async function loadAgentsAndTasksForContext() {
    const projectId = document.getElementById('contextProjectId').value;
    if (!projectId) return;
    
    try {
        const [agentsRes, tasksRes] = await Promise.all([
            fetch(`${API_BASE}/agents?project_id=${projectId}`),
            fetch(`${API_BASE}/tasks?project_id=${projectId}`)
        ]);
        
        const agents = await agentsRes.json() || [];
        const tasks = await tasksRes.json() || [];
        
        const agentSelect = document.getElementById('contextAgentId');
        const taskSelect = document.getElementById('contextTaskId');
        
        agentSelect.innerHTML = agents.map(a => 
            `<option value="${a.id}">${a.name} (${a.role})</option>`
        ).join('');
        
        taskSelect.innerHTML = '<option value="">None</option>' + tasks.map(t => 
            `<option value="${t.id}">${t.title}</option>`
        ).join('');
    } catch (error) {
        console.error('Failed to load data:', error);
    }
}

function showNotification(message, type = 'success') {
    const notification = document.getElementById('notification');
    notification.textContent = message;
    notification.className = 'notification show';
    
    setTimeout(() => {
        notification.classList.remove('show');
    }, 3000);
}
