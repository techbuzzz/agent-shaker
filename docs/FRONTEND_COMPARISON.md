# Frontend Comparison: Vue.js vs Static HTML

## Overview
The agent-shaker project now has **two separate frontends** serving different purposes:

| Feature | Vue.js SPA (`/`) | Static HTML (`/demo/`) |
|---------|------------------|------------------------|
| **URL** | `http://localhost/` | `http://localhost/demo/` |
| **Technology** | Vue 3 + Vite + Tailwind | Vanilla HTML/CSS/JS |
| **Purpose** | Production application | Testing & demos |
| **Build Required** | Yes (npm run build) | No |
| **State Management** | Pinia stores | Global variables |
| **Routing** | Vue Router | Single page |
| **Styling** | Tailwind CSS classes | Custom CSS |
| **Bundle Size** | ~200KB (optimized) | ~50KB (uncompressed) |
| **Hot Reload** | Yes (dev mode) | No |
| **TypeScript** | Optional | No |
| **Components** | Reusable Vue components | Pure HTML |
| **Served By** | Nginx (from /dist) | Go backend (from /web/static) |

## Use Cases

### Vue.js SPA - When to Use
✅ **Production deployment**
- Full-featured application
- Complex user interactions
- Multiple views/pages
- Shared components
- State management needed
- TypeScript support required
- Modern development experience

### Static HTML - When to Use
✅ **Development & testing**
- Quick UI prototypes
- Testing API endpoints
- Demo for stakeholders
- Documentation examples
- Backward compatibility
- No build step needed
- Learning/training purposes

## Code Comparison

### Creating a Modal

#### Vue.js Approach (Component-based)
```vue
<!-- components/AgentModal.vue -->
<template>
  <div v-if="isOpen" class="modal">
    <div class="modal-content">
      <div class="modal-header">
        <h2 class="modal-title">{{ title }}</h2>
        <button @click="close" class="close">&times;</button>
      </div>
      <div class="modal-body">
        <slot></slot>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps(['isOpen', 'title'])
const emit = defineEmits(['close'])

const close = () => emit('close')
</script>

<style scoped>
/* Scoped styles here */
</style>
```

#### Static HTML Approach (Vanilla)
```html
<!-- index.html -->
<div id="agentModal" class="modal">
  <div class="modal-content">
    <div class="modal-header">
      <h2 class="modal-title">Add Agent</h2>
      <span class="close" onclick="closeModal('agentModal')">&times;</span>
    </div>
    <div class="modal-body">
      <!-- Form content -->
    </div>
  </div>
</div>

<script>
function openModal(id) {
  document.getElementById(id).classList.add('active');
}

function closeModal(id) {
  document.getElementById(id).classList.remove('active');
}
</script>
```

### API Calls

#### Vue.js (Composable)
```javascript
// services/api.js
import axios from 'axios'

const API_BASE = '/api'

export const projectApi = {
  async getAll() {
    const { data } = await axios.get(`${API_BASE}/projects`)
    return data
  },
  async create(project) {
    const { data } = await axios.post(`${API_BASE}/projects`, project)
    return data
  }
}
```

```vue
<!-- views/Projects.vue -->
<script setup>
import { onMounted } from 'vue'
import { useProjectStore } from '@/stores/projectStore'

const projectStore = useProjectStore()

onMounted(async () => {
  await projectStore.fetchProjects()
})
</script>
```

#### Static HTML (Fetch API)
```javascript
// app.js
const API_BASE = window.location.origin + '/api';

async function loadProjects() {
  try {
    const response = await fetch(`${API_BASE}/projects`);
    projects = await response.json() || [];
    renderProjects();
  } catch (error) {
    console.error('Failed to load projects:', error);
  }
}

async function createProject(event) {
  event.preventDefault();
  const data = {
    name: document.getElementById('projectName').value,
    description: document.getElementById('projectDescription').value
  };
  
  const response = await fetch(`${API_BASE}/projects`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  });
  
  if (response.ok) {
    await loadProjects();
    closeModal('projectModal');
  }
}
```

## File Structure Comparison

### Vue.js Structure
```
web/src/
├── App.vue                 # Root component
├── main.js                 # Entry point
├── assets/
│   └── styles.css         # Global styles
├── components/            # Reusable components
│   ├── ProjectCard.vue
│   ├── AgentCard.vue
│   └── TaskCard.vue
├── views/                 # Page components
│   ├── Dashboard.vue
│   ├── Projects.vue
│   ├── Agents.vue
│   └── Tasks.vue
├── stores/                # Pinia state
│   ├── projectStore.js
│   ├── agentStore.js
│   └── taskStore.js
├── services/              # API layer
│   └── api.js
├── router/                # Vue Router
│   └── index.js
└── composables/           # Reusable logic
    └── useWebSocket.js
```

### Static HTML Structure
```
web/static/
├── index.html             # Single HTML file (all-in-one)
├── app.js                 # All JavaScript logic
└── style.css              # All styles
```

## Performance Comparison

### Initial Load Time

| Metric | Vue.js (Production) | Static HTML |
|--------|---------------------|-------------|
| **HTML Size** | 0.5 KB | 10 KB |
| **CSS Size** | 15 KB (Tailwind purged) | 8 KB |
| **JS Size** | ~150 KB (minified + gzipped) | 30 KB |
| **Total Transfer** | ~165 KB | ~48 KB |
| **Parse Time** | ~100ms | ~20ms |
| **Hydration** | ~50ms | Instant |

### Subsequent Navigation

| Metric | Vue.js | Static HTML |
|--------|--------|-------------|
| **Page Changes** | Instant (client-side) | Full reload |
| **API Calls** | Only data | Full page + data |
| **State Preservation** | Yes | No |

## Development Experience

### Vue.js Development
```bash
# Development mode with hot reload
cd web
npm install
npm run dev

# Access at http://localhost:5173

# Build for production
npm run build

# Test production build
npm run preview
```

### Static HTML Development
```bash
# No build required!
# Just edit files in web/static/

# View changes:
docker-compose restart mcp-server

# Or use local file server
cd web/static
python -m http.server 8000
```

## Deployment

### Vue.js Deployment
1. Built files go to `web/dist/`
2. Nginx serves static files
3. Client-side routing handled by Vue Router
4. Optimized bundle with tree-shaking
5. Code splitting for lazy loading

**Nginx Config:**
```nginx
location / {
    try_files $uri $uri/ /index.html;  # SPA fallback
}
```

### Static HTML Deployment
1. Files served directly from `web/static/`
2. Go backend acts as file server
3. Simple HTTP file serving
4. No special routing needed

**Go Server:**
```go
staticDir := http.Dir("web/static")
staticHandler := http.StripPrefix("/demo/", http.FileServer(staticDir))
r.PathPrefix("/demo/").Handler(staticHandler)
```

## When to Switch Between Them

### Switch from Static to Vue.js when:
- ✅ Project grows beyond single page
- ✅ Need component reusability
- ✅ Want better developer tools
- ✅ Require state management
- ✅ Planning long-term maintenance
- ✅ Need TypeScript support
- ✅ Want automated testing

### Switch from Vue.js to Static when:
- ✅ Need quick prototype
- ✅ Build step is too slow
- ✅ Simple demo needed
- ✅ Testing API changes quickly
- ✅ Creating documentation examples
- ✅ Training non-JavaScript developers

## Migration Path

### Static → Vue.js
1. Create component for each modal
2. Move JavaScript logic to methods
3. Convert fetch calls to Pinia actions
4. Split CSS into component styles
5. Add routing for different views
6. Implement state management

### Vue.js → Static
1. Flatten component hierarchy
2. Inline all templates in single HTML
3. Combine all JavaScript into one file
4. Merge all CSS into one file
5. Replace store with global variables
6. Remove router, use hash navigation

## Maintenance Considerations

### Vue.js Maintenance
- ✅ **Pros**: Organized, scalable, testable
- ❌ **Cons**: Dependency updates, build complexity

### Static HTML Maintenance
- ✅ **Pros**: Simple, no dependencies, fast
- ❌ **Cons**: Code duplication, harder to refactor

## Recommendation

### For This Project
- **Primary**: Use Vue.js (`/`) for the main application
- **Secondary**: Keep static demo (`/demo/`) for:
  - Testing new features
  - API documentation
  - Quick demos
  - Training materials

### Best Practice
Keep both in sync:
1. Prototype in static HTML (fast iteration)
2. Once stable, port to Vue.js (production)
3. Keep static version as reference/demo

## Conclusion

Both frontends serve the **same API** from the Go backend, making them functionally equivalent. Choose based on your needs:

- **Need speed?** → Static HTML
- **Need scale?** → Vue.js
- **Need both?** → Keep both! 🚀

## Quick Access

- **Vue.js SPA**: http://localhost/
- **Static Demo**: http://localhost/demo/
- **API Docs**: http://localhost/api/docs
- **WebSocket**: ws://localhost/ws?project_id=<uuid>
