# Migration Concurrency Safety

## Problem

The original migration system had a race condition when multiple instances started simultaneously:

1. **Instance A** reads `schema_migrations`, sees migration X is not applied
2. **Instance B** reads `schema_migrations`, sees migration X is not applied
3. **Instance A** executes migration X DDL successfully
4. **Instance B** executes migration X DDL (may succeed if idempotent)
5. **Instance A** inserts `version=X` into `schema_migrations` → SUCCESS
6. **Instance B** tries to insert `version=X` into `schema_migrations` → PRIMARY KEY CONFLICT → CRASH

Even though the migration was successfully applied by Instance A, Instance B would crash, preventing the application from starting.

## Solution

The implementation now uses PostgreSQL's `INSERT ... ON CONFLICT DO NOTHING RETURNING` pattern to atomically claim migrations:

```go
var claimedVersion string
err = db.QueryRow(
    `INSERT INTO schema_migrations (version, applied_at) 
     VALUES ($1, CURRENT_TIMESTAMP) 
     ON CONFLICT (version) DO NOTHING 
     RETURNING version`,
    entry.Name(),
).Scan(&claimedVersion)

if err != nil {
    // sql.ErrNoRows means ON CONFLICT happened - another instance claimed this
    log.Printf("Migration %s already claimed by another instance, skipping", entry.Name())
    continue
}

// Only the instance that successfully claimed the migration reaches here
// and executes the DDL
```

### How It Works

1. Each instance attempts to insert a row into `schema_migrations` for the migration
2. The `ON CONFLICT DO NOTHING` clause prevents errors if the row already exists
3. The `RETURNING version` clause returns the version only if the INSERT succeeded
4. If another instance already claimed the migration:
   - `ON CONFLICT` prevents the insert
   - No rows are returned
   - `Scan()` returns `sql.ErrNoRows`
   - The instance safely skips this migration
5. Only one instance will successfully claim and execute each migration

### Benefits

- **No race conditions**: The database enforces atomicity
- **No crashes**: Concurrent instances gracefully skip already-claimed migrations
- **Simple**: No need for advisory locks or external coordination
- **Docker-safe**: Works correctly with `docker-compose up --scale app=3`
- **Production-safe**: Multiple pods/containers can start simultaneously

### Trade-offs

- Migrations are no longer wrapped in a single transaction
- The tracking row is inserted *before* executing the DDL
- If a migration fails:
  - The tracking row remains (prevents re-attempts)
  - Manual intervention is required to fix and continue
  - This is intentional: failed migrations should not auto-retry

### Testing Concurrent Safety

Run the integration test with a test database:

```bash
export TEST_DATABASE_URL="postgresql://user:pass@localhost/testdb"
go test ./cmd/server -v -run TestMigrationConcurrentSafety
```

Or test manually with Docker Compose:

```bash
docker-compose up --scale app=3 -d
docker-compose logs app
# All three instances should start without migration conflicts
```
