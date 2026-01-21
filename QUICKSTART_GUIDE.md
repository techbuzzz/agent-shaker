# Quick Start Guide - MCP Task Tracker

## Prerequisites
- Go 1.21+ installed
- PostgreSQL running
- Node.js 18+ installed
- Git

## Step 1: Database Setup

### Start PostgreSQL
Make sure PostgreSQL is running on port 5432.

### Create Database (if not exists)
```sql
CREATE DATABASE mcp_tracker;
CREATE USER mcp WITH PASSWORD 'secret';
GRANT ALL PRIVILEGES ON DATABASE mcp_tracker TO mcp;
```

## Step 2: Build and Run Backend

### From Project Root

```bash
# Build the backend
go build -o mcp-tracker ./cmd/server

# Run migrations (automatic on first start)
./mcp-tracker
```

The backend will:
- Connect to database
- Run migrations automatically
- Start on port 8080
- Create WebSocket hub

### Environment Variables (Optional)
```bash
# Custom database URL
export DATABASE_URL="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

# Custom port
export PORT="8080"
```

## Step 3: Run Frontend

### Navigate to Web Directory
```bash
cd web
```

### Install Dependencies (if not done)
```bash
npm install
```

### Start Development Server
```bash
npm run dev
```

Frontend runs on: http://localhost:3000

## Step 4: Add Sample Data

### Option 1: SQL Script
```bash
psql -U mcp -d mcp_tracker -f migrations/002_sample_data.sql
```

### Option 2: Via UI
1. Open http://localhost:3000
2. Go to Projects
3. Click "+ Create Project"
4. Fill in details and submit
5. Click on the project
6. Go to "Agents" tab
7. Click "+ Add Agent"
8. Fill in details and submit

## Accessing the Application

### Main URLs
- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080/api
- **Health Check**: http://localhost:8080/health
- **WebSocket**: ws://localhost:8080/ws

### Pages
- **Dashboard**: http://localhost:3000/
- **Projects**: http://localhost:3000/projects
- **Agents**: http://localhost:3000/agents
- **Tasks**: http://localhost:3000/tasks
- **Documentation**: http://localhost:3000/docs

## Common Commands

### Backend

```bash
# Build
go build -o mcp-tracker ./cmd/server

# Run
./mcp-tracker

# Run with custom database
DATABASE_URL="postgres://..." ./mcp-tracker

# Run tests
go test ./...

# Run with race detector
go run -race ./cmd/server
```

### Frontend

```bash
# Install dependencies
npm install

# Development server
npm run dev

# Production build
npm run build

# Preview production build
npm run preview

# Lint
npm run lint
```

### Database

```bash
# Connect to database
psql -U mcp -d mcp_tracker

# Run migrations
psql -U mcp -d mcp_tracker -f migrations/001_init.sql

# Add sample data
psql -U mcp -d mcp_tracker -f migrations/002_sample_data.sql

# Backup database
pg_dump -U mcp mcp_tracker > backup.sql

# Restore database
psql -U mcp -d mcp_tracker < backup.sql

# Reset database
psql -U mcp -d mcp_tracker -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
psql -U mcp -d mcp_tracker -f migrations/001_init.sql
```

## Troubleshooting

### Backend won't start
**Check:**
1. PostgreSQL is running: `psql -U mcp -d mcp_tracker -c "SELECT 1"`
2. Port 8080 is available: `netstat -ano | findstr :8080`
3. Database URL is correct
4. Migrations ran successfully

### Frontend can't connect to backend
**Check:**
1. Backend is running on port 8080
2. CORS is properly configured
3. Proxy settings in `vite.config.js`
4. Browser console for errors

### WebSocket connection fails
**Check:**
1. Backend is running
2. WebSocket endpoint is accessible
3. Browser console for connection errors
4. Firewall settings

### No data showing
**Check:**
1. Database has data: `psql -U mcp -d mcp_tracker -c "SELECT COUNT(*) FROM projects"`
2. API endpoints return data: `curl http://localhost:8080/api/projects`
3. Browser console for API errors
4. Network tab in dev tools

## Development Workflow

### Making Backend Changes
1. Edit Go files
2. `go build -o mcp-tracker ./cmd/server`
3. Stop and restart server
4. Test changes

### Making Frontend Changes
1. Edit Vue files
2. Changes hot-reload automatically
3. Check browser for updates
4. Test functionality

### Database Changes
1. Create new migration file
2. Apply migration: `psql -U mcp -d mcp_tracker -f migrations/003_your_migration.sql`
3. Update models if needed
4. Rebuild and restart backend

## API Testing

### Using curl

```bash
# Create project
curl -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","description":"Test project"}'

# Get all projects
curl http://localhost:8080/api/projects

# Create agent
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{"project_id":"uuid","name":"Test Agent","role":"frontend","team":"Dev"}'

# Get all agents
curl http://localhost:8080/api/agents

# Get project agents
curl "http://localhost:8080/api/agents?project_id=uuid"
```

### Using PowerShell

```powershell
# Create project
$project = Invoke-RestMethod -Uri "http://localhost:8080/api/projects" `
  -Method Post `
  -ContentType "application/json" `
  -Body '{"name":"Test","description":"Test project"}'

# Get all projects
Invoke-RestMethod -Uri "http://localhost:8080/api/projects"

# Create agent
Invoke-RestMethod -Uri "http://localhost:8080/api/agents" `
  -Method Post `
  -ContentType "application/json" `
  -Body "{`"project_id`":`"$($project.id)`",`"name`":`"Test Agent`",`"role`":`"frontend`",`"team`":`"Dev`"}"

# Get all agents
Invoke-RestMethod -Uri "http://localhost:8080/api/agents"
```

## Production Build

### Backend
```bash
# Build optimized binary
go build -ldflags="-s -w" -o mcp-tracker ./cmd/server

# Or with compression
go build -ldflags="-s -w" -o mcp-tracker ./cmd/server
upx mcp-tracker
```

### Frontend
```bash
cd web
npm run build
# Output in web/dist/
```

### Deploy
1. Build both backend and frontend
2. Upload backend binary and dist/ folder
3. Set environment variables
4. Configure reverse proxy (nginx, etc.)
5. Setup systemd service (Linux) or Windows Service
6. Configure SSL/TLS

## Monitoring

### Check Backend Health
```bash
curl http://localhost:8080/health
```

### Check Database Connections
```sql
SELECT COUNT(*) FROM pg_stat_activity WHERE datname = 'mcp_tracker';
```

### View Logs
Backend logs to stdout, so redirect to file:
```bash
./mcp-tracker > app.log 2>&1 &
```

## Security Checklist

- [ ] Change default database password
- [ ] Use environment variables for secrets
- [ ] Enable SSL/TLS in production
- [ ] Configure CORS properly
- [ ] Use strong JWT secrets
- [ ] Enable rate limiting
- [ ] Set up firewall rules
- [ ] Regular security updates
- [ ] Database backups configured
- [ ] HTTPS for frontend

---

**Happy Coding! 🚀**

For more information, see:
- `AGENTS_PAGE_IMPLEMENTATION.md` - Agents page details
- `UI_MODERNIZATION.md` - UI documentation
- `docs/API.md` - API documentation
