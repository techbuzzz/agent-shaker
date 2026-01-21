#!/usr/bin/env node

/**
 * Agent Shaker MCP Bridge
 * A simple bridge between GitHub Copilot and the Agent Shaker API
 */

const axios = require('axios');
const readline = require('readline');

const API_BASE = process.env.AGENT_SHAKER_URL || 'http://localhost:8080/api';

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    prompt: 'agent-shaker> '
});

// Color codes for pretty output
const colors = {
    reset: '\x1b[0m',
    bright: '\x1b[1m',
    green: '\x1b[32m',
    blue: '\x1b[34m',
    yellow: '\x1b[33m',
    red: '\x1b[31m'
};

function log(message, color = colors.reset) {
    console.log(`${color}${message}${colors.reset}`);
}

function formatAgent(agent) {
    const statusColor = agent.status === 'active' ? colors.green : colors.red;
    return `${colors.bright}${agent.name}${colors.reset} (${colors.blue}${agent.role}${colors.reset}) - ${statusColor}${agent.status}${colors.reset}\n  Team: ${agent.team}\n  Last seen: ${agent.last_seen}`;
}

function formatProject(project) {
    return `${colors.bright}${project.name}${colors.reset}\n  ${project.description}\n  Status: ${project.status}`;
}

function formatTask(task) {
    const priorityColor = task.priority === 'high' ? colors.red : task.priority === 'medium' ? colors.yellow : colors.reset;
    return `${colors.bright}${task.title}${colors.reset}\n  Priority: ${priorityColor}${task.priority}${colors.reset} | Status: ${task.status}\n  ${task.description || 'No description'}`;
}

async function executeCommand(command) {
    const parts = command.trim().split(' ');
    const action = parts[0];
    const resource = parts[1];

    try {
        switch (`${action} ${resource}`) {
            case 'list agents': {
                const projectId = parts.find(p => p.startsWith('project:'))?.split(':')[1];
                const url = projectId 
                    ? `${API_BASE}/agents?project_id=${projectId}`
                    : `${API_BASE}/agents`;
                
                const response = await axios.get(url);
                log(`\nFound ${response.data.length} agents:`, colors.green);
                response.data.forEach(agent => console.log(formatAgent(agent)));
                break;
            }

            case 'list projects': {
                const response = await axios.get(`${API_BASE}/projects`);
                log(`\nFound ${response.data.length} projects:`, colors.green);
                response.data.forEach(project => console.log(formatProject(project)));
                break;
            }

            case 'list tasks': {
                const projectId = parts.find(p => p.startsWith('project:'))?.split(':')[1];
                if (!projectId) {
                    log('Error: project_id required. Usage: list tasks project:PROJECT_ID', colors.red);
                    break;
                }
                const response = await axios.get(`${API_BASE}/tasks?project_id=${projectId}`);
                log(`\nFound ${response.data.length} tasks:`, colors.green);
                response.data.forEach(task => console.log(formatTask(task)));
                break;
            }

            case 'get project': {
                const projectId = parts[2];
                if (!projectId) {
                    log('Error: project_id required. Usage: get project PROJECT_ID', colors.red);
                    break;
                }
                const response = await axios.get(`${API_BASE}/projects/${projectId}`);
                log('\nProject details:', colors.green);
                console.log(JSON.stringify(response.data, null, 2));
                break;
            }

            case 'create task': {
                log('Enter task details:', colors.blue);
                const title = await askQuestion('Title: ');
                const description = await askQuestion('Description: ');
                const projectId = await askQuestion('Project ID: ');
                const priority = await askQuestion('Priority (low/medium/high): ') || 'medium';

                const response = await axios.post(`${API_BASE}/tasks`, {
                    title,
                    description,
                    project_id: projectId,
                    priority,
                    status: 'pending'
                });
                log('Task created successfully!', colors.green);
                console.log(JSON.stringify(response.data, null, 2));
                break;
            }

            case 'help':
            case 'undefined undefined': {
                showHelp();
                break;
            }

            default: {
                log(`Unknown command: ${command}`, colors.red);
                log('Type "help" for available commands', colors.yellow);
            }
        }
    } catch (error) {
        if (error.response) {
            log(`API Error: ${error.response.status} - ${error.response.statusText}`, colors.red);
            if (error.response.data) {
                console.log(error.response.data);
            }
        } else {
            log(`Error: ${error.message}`, colors.red);
        }
    }
}

function askQuestion(question) {
    return new Promise((resolve) => {
        rl.question(question, (answer) => {
            resolve(answer);
        });
    });
}

function showHelp() {
    console.log(`
${colors.bright}Agent Shaker MCP Bridge - Available Commands:${colors.reset}

${colors.blue}Query Commands:${colors.reset}
  list agents [project:ID]     - List all agents or filter by project
  list projects                - List all projects
  list tasks project:ID        - List tasks for a project
  get project PROJECT_ID       - Get details of a specific project

${colors.blue}Action Commands:${colors.reset}
  create task                  - Create a new task (interactive)

${colors.blue}System Commands:${colors.reset}
  help                         - Show this help message
  exit                         - Exit the bridge

${colors.blue}Examples:${colors.reset}
  ${colors.yellow}list agents${colors.reset}
  ${colors.yellow}list agents project:550e8400-e29b-41d4-a716-446655440001${colors.reset}
  ${colors.yellow}list projects${colors.reset}
  ${colors.yellow}create task${colors.reset}
`);
}

// Main
console.log(`${colors.bright}${colors.green}
╔═══════════════════════════════════════════╗
║    Agent Shaker MCP Bridge v1.0.0         ║
╚═══════════════════════════════════════════╝
${colors.reset}`);
console.log(`API Base: ${API_BASE}`);
console.log(`Type ${colors.yellow}help${colors.reset} for available commands\n`);

rl.on('line', async (line) => {
    if (line.trim() === 'exit') {
        log('Goodbye!', colors.green);
        process.exit(0);
    }
    
    await executeCommand(line.trim());
    console.log(); // Empty line for readability
    rl.prompt();
});

rl.on('close', () => {
    log('\nGoodbye!', colors.green);
    process.exit(0);
});

rl.prompt();
