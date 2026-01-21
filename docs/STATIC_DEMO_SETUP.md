# Static Demo HTML served from Go Backend

## Overview
The static HTML demo (with improved modal UI) is now served directly from the Go backend at the `/demo/` route, while the main Vue.js SPA continues to run at the root path.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Browser Requests                            │
└────────────────────────┬─────────────────────┬──────────────────────┘
                         │                     │
                         │                     │
                    ┌────▼─────┐         ┌─────▼──────┐
                    │    /     │         │   /demo/   │
                    │ Vue.js   │         │  Static    │
                    │   SPA    │         │   HTML     │
                    └────┬─────┘         └─────┬──────┘
                         │                     │
                         │                     │
                    ┌────▼─────────────────────▼──────┐
                    │     Nginx (Port 80)             │
                    │   • Serves Vue.js from /dist    │
                    │   • Proxies /demo/ to Go        │
                    │   • Proxies /api/ to Go         │
                    │   • Proxies /ws to Go           │
                    └────┬────────────────────┬────────┘
                         │                    │
                         │                    │
                    ┌────▼────────────────────▼────────┐
                    │  Go MCP Server (Port 8080)       │
                    │   • REST API (/api/*)            │
                    │   • WebSocket (/ws)              │
                    │   • Static Demo (/demo/)         │
                    │   • File Server (web/static/)    │
                    └──────────────────────────────────┘
```

## Endpoints

### Production Vue.js App
- **URL**: `http://localhost/`
- **Description**: Modern Vue.js 3 Single Page Application
- **Features**: 
  - Tailwind CSS styling
  - Vue Router for navigation
  - Pinia state management
  - WebSocket integration
  - Responsive design

### Static HTML Demo
- **URL**: `http://localhost/demo/`
- **URL (Direct)**: `http://localhost:8080/demo/`
- **Description**: Static HTML with improved modal UI
- **Features**:
  - Modern modal design
  - Improved form styling
  - CSS animations
  - Vanilla JavaScript

### API Access
- **URL**: `http://localhost/api/*`
- **Description**: REST API endpoints
- **Proxied to**: `http://mcp-server:8080/api/*`

### WebSocket
- **URL**: `ws://localhost/ws?project_id=<uuid>`
- **Description**: Real-time communication
- **Proxied to**: `ws://mcp-server:8080/ws`

## File Structure

```
agent-shaker/
├── web/
│   ├── src/                    # Vue.js app source (Served at /)
│   │   ├── App.vue
│   │   ├── main.js
│   │   ├── components/
│   │   ├── views/
│   │   └── stores/
│   │
│   ├── static/                 # Static HTML (Served at /demo/)
│   │   ├── index.html         # Main demo page
│   │   ├── app.js             # Vanilla JavaScript
│   │   └── style.css          # CSS with modal improvements
│   │
│   ├── dist/                   # Built Vue.js (generated)
│   ├── Dockerfile             # Builds Vue.js app
│   └── nginx.conf             # Nginx configuration
│
└── cmd/server/main.go         # Go backend with static file server
```

## Implementation Details

### Go Backend (main.go)

```go
// Static demo HTML (for testing/demo purposes)
staticDir := http.Dir("web/static")
staticHandler := http.StripPrefix("/demo/", http.FileServer(staticDir))
r.PathPrefix("/demo/").Handler(staticHandler)
```

The Go server:
1. Creates a file server for the `web/static` directory
2. Strips the `/demo/` prefix from URLs
3. Serves files from `web/static/`

Example URL mapping:
- `http://localhost:8080/demo/` → `web/static/index.html`
- `http://localhost:8080/demo/app.js` → `web/static/app.js`
- `http://localhost:8080/demo/style.css` → `web/static/style.css`

### Nginx Configuration (nginx.conf)

```nginx
# Static demo HTML served from Go backend
location /demo/ {
    proxy_pass http://mcp-server:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Nginx:
1. Intercepts requests to `/demo/*`
2. Proxies them to the Go backend
3. Sets proper headers for logging and security

### Docker Integration

The `web/static` folder is copied into the Go container during build:

```dockerfile
# In Dockerfile (mcp-server stage)
COPY web ./web
```

This ensures the static files are available inside the container at `/app/web/static/`.

## Why Two Frontends?

### Vue.js App (Production)
- **Purpose**: Main production application
- **Benefits**: 
  - Component-based architecture
  - State management
  - TypeScript support
  - Hot module replacement in dev
  - Optimized production builds

### Static HTML (Demo/Testing)
- **Purpose**: Quick testing, demos, backward compatibility
- **Benefits**:
  - No build step required
  - Easy to modify and test
  - Vanilla JavaScript - no frameworks
  - Can serve as documentation/examples

## API Interaction

Both frontends interact with the same Go backend API:

```javascript
// In static/app.js
const API_BASE = window.location.origin + '/api';
const WS_BASE = 'ws://' + window.location.host + '/ws';

// Same endpoints work for both:
fetch(`${API_BASE}/projects`)    // Works from / and /demo/
```

## Development Workflow

### Modifying Vue.js App
1. Edit files in `web/src/`
2. Rebuild: `docker-compose up -d --build web`
3. Access at: `http://localhost/`

### Modifying Static Demo
1. Edit files in `web/static/`
2. Rebuild: `docker-compose up -d --build mcp-server`
3. Access at: `http://localhost/demo/`

**Note**: Static files are copied into the Go container, so changes require rebuilding the `mcp-server` service.

## Testing

### Test Vue.js App
```powershell
Invoke-WebRequest -Uri http://localhost/ -UseBasicParsing
```

### Test Static Demo
```powershell
Invoke-WebRequest -Uri http://localhost/demo/ -UseBasicParsing
```

### Test Direct Backend Access
```powershell
# Via Nginx (recommended)
Invoke-WebRequest -Uri http://localhost/demo/ -UseBasicParsing

# Direct to Go server (bypass Nginx)
Invoke-WebRequest -Uri http://localhost:8080/demo/ -UseBasicParsing
```

## Logs

### Check Go Server Logs
```powershell
docker-compose logs mcp-server
```

Expected output:
```
MCP API Server - Endpoints:
  API:         http://localhost:8080/api
  WebSocket:   ws://localhost:8080/ws
  Health:      http://localhost:8080/health
  Docs:        http://localhost:8080/api/docs
  Static Demo: http://localhost:8080/demo/
```

### Check Nginx Logs
```powershell
docker-compose logs web
```

## Troubleshooting

### 404 on /demo/ route

**Problem**: Getting 404 when accessing `/demo/`

**Solutions**:
1. Ensure Go server includes file server code
2. Verify `web/static/` folder exists with files
3. Check Nginx proxy configuration
4. Rebuild containers: `docker-compose up -d --build`

### Static files not updating

**Problem**: Changes to `web/static/` files not reflected

**Solution**: 
Static files are copied during Docker build. You must rebuild:
```powershell
docker-compose up -d --build mcp-server
```

### Modal not showing improvements

**Problem**: Modal UI improvements not visible

**Solution**:
1. Clear browser cache (Ctrl+Shift+Delete)
2. Hard refresh (Ctrl+F5)
3. Verify you're on `/demo/` not `/`

## Best Practices

1. **Use Vue.js for production**: The main app at `/` is optimized and production-ready
2. **Use static demo for**:
   - Quick UI mockups
   - Testing API changes
   - Demonstrating features
   - Documentation examples

3. **Keep both in sync**: If you add features to static demo, consider porting to Vue.js

4. **Cache management**: Static files served from Go may need cache headers for production

## Security Considerations

1. **Path Traversal**: The `http.FileServer` is restricted to `web/static/` directory
2. **Access Control**: No authentication on `/demo/` - add if needed for production
3. **CORS**: Already configured in Go server for API access
4. **Headers**: Security headers set in Nginx configuration

## Future Enhancements

1. **Option to disable demo**: Add environment variable to enable/disable `/demo/` route
2. **Hot reload**: Add file watching for static files during development
3. **CDN integration**: Serve static assets from CDN in production
4. **Authentication**: Add auth middleware if demo contains sensitive data

## Summary

✅ **Vue.js SPA**: `http://localhost/` - Production application
✅ **Static Demo**: `http://localhost/demo/` - Testing and demos
✅ **Go Backend**: Serves both static files and API
✅ **Nginx**: Proxies both routes appropriately
✅ **Clean Architecture**: Separation of concerns maintained
