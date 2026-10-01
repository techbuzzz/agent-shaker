# Migrations

Schema changes are applied by [`golang-migrate`](https://github.com/golang-migrate/migrate)
through the thin wrapper in [`cmd/migrate`](../cmd/migrate/main.go). This document
describes what that code actually does; an earlier version of this file described
a custom runner that no longer exists, and following it would have led to
migrations that silently never applied.

## How migrations run

Migrations are a **one-shot job**, not a startup step:

```
postgres (healthy) ──► migrate (runs, exits 0) ──► mcp-server (healthy) ──► web
```

`docker-compose.yml` gates `mcp-server` on `migrate` completing successfully, so
the API never starts against a stale schema. `mcp-server` sets
`RUN_MIGRATIONS_ON_START=false` so the two cannot race on the same tables.

```bash
make migrate-up         # ./cmd/migrate -cmd up -dir migrations
make migrate-version    # version=<n> dirty=<bool>
```

Outside Docker, `DATABASE_URL` must be set and the schema must already exist.

## File naming is strict

The driver parses filenames with:

```
^([0-9]+)_(.*)\.(down|up)\.(.*)$
```

so **both infixes are required**:

```
001_init.up.sql          ✓
008_remove_sample_data.up.sql   ✓
001_init.sql             ✗ silently rejected
```

A file that does not match makes the driver report `first .: file does not exist`,
which reads like a missing directory rather than a naming mistake. `cmd/migrate`
therefore scans the directory first and fails by filename, before handing
anything to the driver:

```
migrations dir "migrations" contains 1 file(s) golang-migrate cannot parse: notes.sql
every file must be named <version>_<title>.sql (e.g. 008_add_widget.sql)
```

One-off and manual SQL does not belong in `migrations/` — it goes in
`scripts/`, as `bootstrap_existing_db.sql` and `seed_demo_data.sql` do.

## Migrations are forward-only

`-cmd down` is refused outright:

```
-cmd down is not supported: this project's migrations are forward-only
(no .down.sql files are shipped on purpose). Use -cmd force -version N to
resynchronise the recorded version after a failed run.
```

Shipping untested `.down.sql` files would be worse than refusing: a wrong or
no-op down migration either destroys data, or marks a version as reverted while
leaving the schema in place. Neither is recoverable by the operator. To undo
something, write a new forward migration.

## The state table

golang-migrate owns one table, and it does **not** look like what a hand-rolled
runner would use:

```sql
CREATE TABLE schema_migrations (
    version bigint  NOT NULL PRIMARY KEY,
    dirty   boolean NOT NULL
);
```

A **single row** holding one integer. There is no `applied_at`, no `checksum`,
and no row per migration file. The highest number in `migrations/` is the
target; the row says how far the database got.

`dirty = true` means a migration half-applied. golang-migrate refuses to run
anything until the version is forced, which is the correct behaviour — the
schema is in an unknown state and guessing is worse than stopping.

```sql
SELECT version, dirty FROM schema_migrations;
```

## Recovering from a failed migration

```bash
make migrate-version
# version=8 dirty=true

# inspect what actually landed, then decide the true version:
docker compose exec -T postgres psql -U mcp -d mcp_tracker -c '\dt'

make migrate-force MIGRATION_VERSION=8
```

`MIGRATION_VERSION` is a plain integer. The target deliberately does not reuse
`VERSION`, which is build metadata and expands to something like
`v0.3.5-68-g85cbbf7-dirty` — a value the `-version` int flag cannot parse. That
collision made the target fail on every invocation, including the one case it
exists for.

Force records a version; it does not run anything. Forcing past a migration that
did not apply leaves the schema short, so inspect first.

## Adopting an existing database

For a database that already has the schema but was never tracked:

```bash
docker compose cp scripts/bootstrap_existing_db.sql postgres:/tmp/boot.sql
docker compose exec -T postgres psql -U mcp -d mcp_tracker \
  -v ON_ERROR_STOP=1 -f /tmp/boot.sql

make migrate-force MIGRATION_VERSION=<highest already applied>
```

`ON_ERROR_STOP=1` is required. Without it psql reports each error, continues,
and exits 0 as long as the last statement succeeded — so a failed adoption
reports success.

Set the version to the highest migration **already present**, not the next one.
Too high skips migrations and leaves the schema short; too low replays one.
Both fail quietly, which is why the script stops at creating the table and
leaves the choice to you.

## Demo data is not a migration

`002_sample_data.up.sql` inserted 3 projects, 9 agents, 9 tasks and 4 contexts
into every database. It was a development affordance that ended up in the
forward-only chain, so a fresh **production** database came up pre-loaded with
fiction, rendered by the UI exactly like real work.

`008_remove_sample_data.up.sql` undoes it, keyed on the exact UUIDs 002 inserted
— never on name or on "everything in the table" — so a real project that happens
to be called "E-Commerce Platform" is untouched. 002 itself is left alone: it
has been applied in existing environments, and editing an applied migration
makes environments diverge.

The dataset is preserved, opt-in:

```bash
docker compose cp scripts/seed_demo_data.sql postgres:/tmp/seed.sql
docker compose exec -T postgres psql -U mcp -d mcp_tracker \
  -v ON_ERROR_STOP=1 -f /tmp/seed.sql
```

Every statement is `ON CONFLICT DO NOTHING`, so it is safe to run twice and
will not clobber real data.

## Writing a migration

1. Find the highest number in `migrations/`.
2. Create `<next>_<descriptive_name>.up.sql`.
3. Write idempotent SQL where you can:

   ```sql
   CREATE TABLE IF NOT EXISTS notifications (...);
   ALTER TABLE tasks ADD COLUMN IF NOT EXISTS priority TEXT;
   CREATE INDEX IF NOT EXISTS idx_notifications_project ON notifications(project_id);
   ```

4. Do not wrap it in `BEGIN`/`COMMIT` — the driver handles that.
5. Do not include `DROP DATABASE` or other global destructive statements.
6. Run it: `make migrate-up`.

`./scripts/create-migration.ps1 "<Title>"` will name the file for you.

## Verification

A fresh database is the case worth checking, because it is the one a new
deployer hits:

```bash
docker compose down -v
docker compose up -d
docker compose exec -T postgres psql -U mcp -d mcp_tracker -c "SELECT version, dirty FROM schema_migrations;"
curl -s http://127.0.0.1:3000/api/projects   # []
```

An empty array is the expected result. If it lists projects, a migration is
seeding data again.
