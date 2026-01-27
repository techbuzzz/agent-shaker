# Database Migration System

## Overview

Agent Shaker uses a custom, lightweight migration system that tracks and applies database schema changes automatically. Migrations are executed on server startup and are idempotent (safe to run multiple times).

## Features

✅ **Automatic Migration Tracking** - Maintains `schema_migrations` table  
✅ **Transactional Execution** - Each migration runs in a transaction  
✅ **Ordered Execution** - Migrations run in alphabetical order  
✅ **Skip Applied Migrations** - Won't re-run already applied migrations  
✅ **Docker Compatible** - Works seamlessly in containerized environments  
✅ **Zero Dependencies** - No external migration tools required  

## How It Works

1. On startup, server creates `schema_migrations` table (if not exists)
2. Reads all `.sql` files from `migrations/` directory
3. Compares with already-applied migrations
4. Executes pending migrations in order within transactions
5. Records each successful migration

## Migration File Format

Migrations follow this naming convention:
```
NNN_descriptive_name.sql
```

- **NNN**: 3-digit number (001, 002, 003, etc.)
- **descriptive_name**: Lowercase with underscores
- **.sql**: Standard SQL file extension

### Examples
```
001_init.sql
002_sample_data.sql
003_daily_standups.sql
004_add_user_roles.sql
```

## Creating a New Migration

### Using PowerShell Script (Recommended)

```powershell
./scripts/create-migration.ps1 "Add User Roles"
```

This creates: `migrations/004_add_user_roles.sql` with a template.

### Manual Creation

1. Find the highest numbered migration in `migrations/`
2. Create new file with next number: `00X_your_change.sql`
3. Write your SQL statements

## Migration Best Practices

### ✅ DO

- **Use transactions implicitly** - Each migration file runs in a transaction
- **Make migrations idempotent** when possible:
  ```sql
  CREATE TABLE IF NOT EXISTS users (...);
  ALTER TABLE tasks ADD COLUMN IF NOT EXISTS priority TEXT;
  ```
- **Test migrations locally first** before deploying
- **Keep migrations small and focused** - One logical change per file
- **Add comments** explaining complex changes:
  ```sql
  -- Migration: Add user authentication
  -- Created: 2026-01-27
  -- Description: Adds users table and authentication columns
  ```
- **Use semantic naming** - `add_user_roles.sql` not `migration_4.sql`

### ❌ DON'T

- **Don't modify existing migrations** that have been deployed
- **Don't delete data** without backup/confirmation
- **Don't use database-specific syntax** unless necessary (prefer standard SQL)
- **Don't include DROP DATABASE** or other destructive global commands
- **Don't include transaction statements** (BEGIN/COMMIT) - handled automatically

## Migration Workflow

### Development

```bash
# 1. Create migration
./scripts/create-migration.ps1 "Your Change"

# 2. Edit the generated SQL file
code migrations/00X_your_change.sql

# 3. Test locally
go run cmd/server/main.go

# 4. Verify migration applied
# Check logs for: "✓ Applied migration: 00X_your_change.sql"
```

### Docker Deployment

Migrations run automatically on container startup:

```bash
docker-compose up -d
docker-compose logs -f agent-shaker
# Watch for migration messages
```

The system ensures:
- Existing databases won't be affected (only new migrations run)
- Multiple containers won't conflict (each checks applied migrations)
- Failed migrations won't leave partial changes (transaction rollback)

## Troubleshooting

### Migration Failed

```
Error: pq: duplicate column "status"
```

**Solution:** Check if migration was partially applied. Either:
1. Fix the SQL to be idempotent (use `IF NOT EXISTS`)
2. Manually rollback the partial change
3. Remove the migration from `schema_migrations` table

### Migration Skipped

```
No pending migrations
```

**Check:**
```sql
SELECT * FROM schema_migrations ORDER BY applied_at DESC;
```

If wrongly marked as applied, delete the record:
```sql
DELETE FROM schema_migrations WHERE version = '00X_filename.sql';
```

### Migration Won't Run

**Possible causes:**
- File not in `migrations/` directory
- File doesn't have `.sql` extension
- Permissions issue reading the file
- Database connection failed

**Debug:**
```bash
# Check files
ls migrations/

# Check database connection
psql $DATABASE_URL -c "SELECT version FROM schema_migrations;"
```

## Database Schema Tracking

The `schema_migrations` table structure:

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,      -- Filename (e.g., "001_init.sql")
    applied_at TIMESTAMP DEFAULT NOW(),    -- When migration was applied
    checksum VARCHAR(64)                   -- Reserved for future checksum validation
);
```

### Viewing Applied Migrations

```sql
-- List all applied migrations
SELECT version, applied_at 
FROM schema_migrations 
ORDER BY version;

-- Check if specific migration applied
SELECT EXISTS(
    SELECT 1 FROM schema_migrations 
    WHERE version = '003_daily_standups.sql'
);
```

## Advanced Usage

### Manual Migration Execution (if needed)

```bash
# Connect to database
psql $DATABASE_URL

# Run migration manually
\i migrations/00X_your_migration.sql

# Mark as applied
INSERT INTO schema_migrations (version) VALUES ('00X_your_migration.sql');
```

### Rollback (Manual Process)

Since migrations don't have automatic rollback:

1. Write reverse SQL manually
2. Execute in database
3. Remove from `schema_migrations`

```sql
-- Example rollback for "ADD COLUMN"
ALTER TABLE tasks DROP COLUMN IF EXISTS priority;
DELETE FROM schema_migrations WHERE version = '00X_add_priority.sql';
```

## Future Enhancements

Planned improvements:
- [ ] Checksum validation to detect modified migrations
- [ ] Down migrations (rollback SQL files)
- [ ] Dry-run mode (`--dry-run` flag)
- [ ] Migration status CLI command
- [ ] Migration locking for distributed systems

## Examples

### Example 1: Add New Table

`004_add_notifications.sql`:
```sql
-- Migration: Add notifications system
-- Created: 2026-01-27

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'info',
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CHECK (type IN ('info', 'warning', 'error', 'success'))
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read) WHERE NOT read;
```

### Example 2: Alter Existing Table

`005_add_task_priority.sql`:
```sql
-- Migration: Add priority field to tasks
-- Created: 2026-01-27

-- Add column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'tasks' AND column_name = 'priority'
    ) THEN
        ALTER TABLE tasks ADD COLUMN priority TEXT DEFAULT 'medium';
        ALTER TABLE tasks ADD CONSTRAINT tasks_priority_check 
            CHECK (priority IN ('low', 'medium', 'high', 'urgent'));
    END IF;
END $$;

-- Set default priority for existing tasks
UPDATE tasks SET priority = 'medium' WHERE priority IS NULL;
```

### Example 3: Data Migration

`006_migrate_agent_status.sql`:
```sql
-- Migration: Standardize agent status values
-- Created: 2026-01-27

-- Normalize status values
UPDATE agents 
SET status = 'active' 
WHERE LOWER(status) IN ('online', 'available', 'ready');

UPDATE agents 
SET status = 'inactive' 
WHERE LOWER(status) IN ('offline', 'unavailable', 'away');

UPDATE agents 
SET status = 'busy' 
WHERE LOWER(status) IN ('working', 'occupied', 'in-progress');
```

## Configuration

### Environment Variables

```bash
# Database connection
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable

# Optional: Custom migrations directory (default: ./migrations)
MIGRATIONS_DIR=./db/migrations
```

### Docker Compose

```yaml
services:
  agent-shaker:
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/dbname
    volumes:
      - ./migrations:/app/migrations:ro  # Mount as read-only
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/techbuzzz/agent-shaker/issues
- Documentation: https://github.com/techbuzzz/agent-shaker/docs
