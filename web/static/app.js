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
    } else if (data.type === 'context_added' || data.type === 'context_updated' || data.type === 'context_deleted') {
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
    const loadingSpinner = document.getElementById('contextsLoading');
    const contextsList = document.getElementById('contextsList');

    loadingSpinner.style.display = 'block';
    contextsList.style.opacity = '0.5';

    try {
        const response = await fetch(`${API_BASE}/contexts?project_id=${projectId}`);
        contexts = await response.json() || [];
        renderContexts();
    } catch (error) {
        console.error('Failed to load contexts:', error);
        showNotification('Failed to load documentation', 'error');
        contextsList.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">❌</div>
                <h3>Failed to Load Documentation</h3>
                <p>There was an error loading the documentation. Please try again.</p>
                <button class="btn btn-primary" onclick="loadContexts('${projectId}')">
                    <span class="btn-icon">🔄</span>
                    Retry
                </button>
            </div>
        `;
    } finally {
        loadingSpinner.style.display = 'none';
        contextsList.style.opacity = '1';
    }
}

function renderContexts() {
    const container = document.getElementById('contextsList');
    if (!contexts || contexts.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">📚</div>
                <h3>No Documentation Yet</h3>
                <p>Start building your knowledge base by adding your first documentation entry.</p>
                <button class="btn btn-primary" onclick="openContextModal()">
                    <span class="btn-icon">➕</span>
                    Add Documentation
                </button>
            </div>
        `;
        return;
    }

    // Update tag filter options
    updateTagFilter();

    container.innerHTML = contexts.map(c => `
        <div class="card" data-context-id="${c.id}">
            <div class="card-content">
                <div class="card-header">
                    <div class="card-title">${escapeHtml(c.title)}</div>
                    <div class="card-meta-extended">
                        <span class="card-author">By ${getAgentName(c.agent_id)}</span>
                        <span class="card-updated">Updated ${formatDate(c.updated_at)}</span>
                    </div>
                </div>
                <div class="card-body card-body-expanded">
                    <p>${escapeHtml(c.content.substring(0, 300))}${c.content.length > 300 ? '...' : ''}</p>
                </div>
                <div class="card-tags">
                    ${c.tags ? c.tags.map(tag => `<span class="badge">${escapeHtml(tag)}</span>`).join(' ') : '<span class="badge" style="background: #e9ecef; color: #6c757d;">No tags</span>'}
                </div>
                <div class="card-actions-extended">
                    <div class="card-actions">
                        <button class="btn btn-sm btn-secondary" onclick="viewContext('${c.id}')" title="View full documentation">
                            <span class="btn-icon">👁️</span>
                            View
                        </button>
                        <button class="btn btn-sm btn-secondary" onclick="editContext('${c.id}')" title="Edit documentation">
                            <span class="btn-icon">✏️</span>
                            Edit
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="confirmDeleteContext('${c.id}', '${escapeHtml(c.title)}')" title="Delete documentation">
                            <span class="btn-icon">🗑️</span>
                            Delete
                        </button>
                    </div>
                    ${c.task_id ? `<div class="task-link">📋 Linked to task</div>` : ''}
                </div>
            </div>
        </div>
    `).join('');
}

// Helper functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function getAgentName(agentId) {
    const agent = agents.find(a => a.id === agentId);
    return agent ? agent.name : 'Unknown Agent';
}

function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diffTime = Math.abs(now - date);
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays === 1) {
        return 'today';
    } else if (diffDays === 2) {
        return 'yesterday';
    } else if (diffDays < 7) {
        return `${diffDays - 1} days ago`;
    } else {
        return date.toLocaleDateString();
    }
}

function updateTagFilter() {
    const tagFilter = document.getElementById('tagFilter');
    const allTags = new Set();

    contexts.forEach(context => {
        if (context.tags) {
            context.tags.forEach(tag => allTags.add(tag));
        }
    });

    const currentValue = tagFilter.value;
    tagFilter.innerHTML = '<option value="">All Tags</option>';

    Array.from(allTags).sort().forEach(tag => {
        const option = document.createElement('option');
        option.value = tag;
        option.textContent = tag;
        if (tag === currentValue) {
            option.selected = true;
        }
        tagFilter.appendChild(option);
    });
}

function filterContexts() {
    const searchTerm = document.getElementById('contextSearch').value.toLowerCase();
    const tagFilter = document.getElementById('tagFilter').value;
    const cards = document.querySelectorAll('#contextsList .card');

    cards.forEach(card => {
        const title = card.querySelector('.card-title').textContent.toLowerCase();
        const content = card.querySelector('.card-body').textContent.toLowerCase();
        const tags = Array.from(card.querySelectorAll('.badge')).map(badge => badge.textContent.toLowerCase());

        const matchesSearch = title.includes(searchTerm) || content.includes(searchTerm);
        const matchesTag = !tagFilter || tags.includes(tagFilter.toLowerCase());

        card.style.display = matchesSearch && matchesTag ? 'block' : 'none';
    });
}

function viewContext(id) {
    const context = contexts.find(c => c.id === id);
    if (!context) return;

    // Create a view modal with full content
    const viewModal = document.createElement('div');
    viewModal.className = 'modal active';
    viewModal.id = 'viewContextModal';
    viewModal.innerHTML = `
        <div class="modal-content" style="max-width: 800px;">
            <div class="modal-header">
                <span class="close" onclick="closeModal('viewContextModal')">&times;</span>
                <h2 class="modal-title">${escapeHtml(context.title)}</h2>
            </div>
            <div class="modal-body" style="max-height: 60vh; overflow-y: auto;">
                <div style="margin-bottom: 15px;">
                    <strong>Author:</strong> ${getAgentName(context.agent_id)} |
                    <strong>Updated:</strong> ${new Date(context.updated_at).toLocaleString()}
                </div>
                <div style="line-height: 1.6; white-space: pre-wrap;">${escapeHtml(context.content)}</div>
                ${context.tags ? `
                    <div style="margin-top: 20px; padding-top: 15px; border-top: 1px solid #e9ecef;">
                        <strong>Tags:</strong> ${context.tags.map(tag => `<span class="badge">${escapeHtml(tag)}</span>`).join(' ')}
                    </div>
                ` : ''}
            </div>
            <div style="padding: 20px; border-top: 1px solid #e9ecef; display: flex; gap: 10px;">
                <button class="btn btn-secondary" onclick="editContext('${context.id}'); closeModal('viewContextModal');">
                    <span class="btn-icon">✏️</span>
                    Edit
                </button>
                <button class="btn btn-danger" onclick="confirmDeleteContext('${context.id}', '${escapeHtml(context.title)}'); closeModal('viewContextModal');">
                    <span class="btn-icon">🗑️</span>
                    Delete
                </button>
            </div>
        </div>
    `;

    document.body.appendChild(viewModal);
}

async function editContext(id) {
    try {
        const response = await fetch(`${API_BASE}/contexts/${id}`);
        const context = await response.json();

        // Populate the modal with existing data
        document.getElementById('contextProjectId').value = context.project_id;
        document.getElementById('contextTitle').value = context.title;
        document.getElementById('contextContent').value = context.content;
        document.getElementById('contextTags').value = context.tags ? context.tags.join(', ') : '';

        // Load agents and tasks for the project
        await loadAgentsAndTasksForContext();

        // Set agent and task if they exist
        if (context.agent_id) {
            document.getElementById('contextAgentId').value = context.agent_id;
        }
        if (context.task_id) {
            document.getElementById('contextTaskId').value = context.task_id;
        }

        // Change modal title and button
        document.querySelector('#contextModal .modal-title').textContent = 'Edit Documentation';
        const submitBtn = document.querySelector('#contextModal .btn-primary');
        submitBtn.textContent = 'Update Documentation';
        
        // Change form submit handler
        const form = document.querySelector('#contextModal form');
        form.onsubmit = (e) => {
            e.preventDefault();
            updateContext(id);
        };

        document.getElementById('contextModal').classList.add('active');
    } catch (error) {
        console.error('Failed to load context for editing:', error);
        showNotification('Failed to load documentation for editing', 'error');
    }
}

async function createContext(event) {
    event.preventDefault();

    // Clear previous validation
    clearFormValidation();

    const formData = getContextFormData();
    if (!validateContextForm(formData)) return;

    const submitBtn = event.target.querySelector('.btn-primary');
    const originalText = submitBtn.innerHTML;
    submitBtn.disabled = true;
    submitBtn.innerHTML = '<span class="btn-icon">⏳</span> Creating...';

    try {
        const response = await fetch(`${API_BASE}/contexts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(formData)
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        loadContexts(formData.project_id);
        closeModal('contextModal');
        showNotification('Documentation added successfully');
    } catch (error) {
        console.error('Failed to create context:', error);
        showFormValidation('contextTitle', 'Failed to create documentation. Please try again.', 'error');
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalText;
    }
}

async function updateContext(id) {
    // Clear previous validation
    clearFormValidation();

    const formData = getContextFormData();
    if (!validateContextForm(formData)) return;

    const submitBtn = document.querySelector('#contextModal .btn-primary');
    const originalText = submitBtn.textContent;
    submitBtn.disabled = true;
    submitBtn.innerHTML = '<span class="btn-icon">⏳</span> Updating...';

    try {
        const response = await fetch(`${API_BASE}/contexts/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                task_id: formData.task_id,
                title: formData.title,
                content: formData.content,
                tags: formData.tags
            })
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        loadContexts(currentProject);
        closeModal('contextModal');
        showNotification('Documentation updated successfully');
    } catch (error) {
        console.error('Failed to update context:', error);
        showNotification('Failed to update documentation. Please try again.', 'error');
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = originalText;
    }
}

async function deleteContext(id) {
    try {
        const loadingBtn = document.querySelector(`[onclick*="confirmDeleteContext('${id}"]`);
        if (loadingBtn) {
            loadingBtn.disabled = true;
            loadingBtn.innerHTML = '<span class="btn-icon">⏳</span> Deleting...';
        }

        await fetch(`${API_BASE}/contexts/${id}`, {
            method: 'DELETE'
        });

        loadContexts(currentProject);
        showNotification('Documentation deleted successfully');
        closeModal('confirmationModal');
    } catch (error) {
        console.error('Failed to delete context:', error);
        showNotification('Failed to delete documentation', 'error');
    } finally {
        // Re-enable all delete buttons
        document.querySelectorAll('.btn-danger').forEach(btn => {
            if (btn.onclick && btn.onclick.toString().includes('confirmDeleteContext')) {
                btn.disabled = false;
                btn.innerHTML = '<span class="btn-icon">🗑️</span> Delete';
            }
        });
    }
}

function confirmDeleteContext(id, title) {
    const confirmationMessage = document.getElementById('confirmationMessage');
    const confirmBtn = document.getElementById('confirmActionBtn');

    confirmationMessage.innerHTML = `
        Are you sure you want to delete the documentation "<strong>${title}</strong>"?<br>
        <small style="color: #6c757d;">This action cannot be undone.</small>
    `;

    confirmBtn.onclick = () => deleteContext(id);
    document.getElementById('confirmationModal').classList.add('active');
}

function getContextFormData() {
    return {
        project_id: document.getElementById('contextProjectId').value,
        agent_id: document.getElementById('contextAgentId').value,
        task_id: document.getElementById('contextTaskId').value || null,
        title: document.getElementById('contextTitle').value.trim(),
        content: document.getElementById('contextContent').value.trim(),
        tags: document.getElementById('contextTags').value ?
            document.getElementById('contextTags').value.split(',').map(t => t.trim()).filter(t => t) : []
    };
}

function validateContextForm(data) {
    let isValid = true;

    if (!data.title) {
        showFormValidation('contextTitle', 'Title is required', 'error');
        isValid = false;
    } else if (data.title.length > 255) {
        showFormValidation('contextTitle', 'Title must be less than 255 characters', 'error');
        isValid = false;
    }

    if (!data.content) {
        showFormValidation('contextContent', 'Content is required', 'error');
        isValid = false;
    }

    if (!data.project_id) {
        showFormValidation('contextProjectId', 'Project selection is required', 'error');
        isValid = false;
    }

    if (!data.agent_id) {
        showFormValidation('contextAgentId', 'Agent selection is required', 'error');
        isValid = false;
    }

    return isValid;
}

function showFormValidation(fieldId, message, type) {
    const field = document.getElementById(fieldId);
    const existingValidation = field.parentNode.querySelector('.form-validation');

    if (existingValidation) {
        existingValidation.remove();
    }

    const validationDiv = document.createElement('div');
    validationDiv.className = `form-validation ${type}`;
    validationDiv.textContent = message;

    field.parentNode.appendChild(validationDiv);
    field.style.borderColor = type === 'error' ? '#dc3545' : '#28a745';
}

function clearFormValidation() {
    document.querySelectorAll('.form-validation').forEach(el => el.remove());
    document.querySelectorAll('#contextModal input, #contextModal textarea, #contextModal select').forEach(el => {
        el.style.borderColor = '#ced4da';
    });
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

    // Reset modal to create mode
    document.querySelector('#contextModal .modal-title').textContent = 'Add Documentation';
    const submitBtn = document.querySelector('#contextModal .btn-primary');
    submitBtn.textContent = 'Add Documentation';
    
    // Reset form submit handler
    const form = document.querySelector('#contextModal form');
    form.onsubmit = createContext;

    // Clear form fields and validation
    clearFormValidation();
    document.getElementById('contextTitle').value = '';
    document.getElementById('contextContent').value = '';
    document.getElementById('contextTags').value = '';

    populateProjectSelect('contextProjectId');
    loadAgentsAndTasksForContext();
    document.getElementById('contextModal').classList.add('active');
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('active');
        // Remove view modal from DOM if it exists
        if (modalId === 'viewContextModal') {
            setTimeout(() => modal.remove(), 300);
        }
    }
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
    notification.className = `notification ${type} show`;

    setTimeout(() => {
        notification.classList.remove('show');
    }, 4000);
}
