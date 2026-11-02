# Database

PostgreSQL integration uses [`pgx/v5`](https://github.com/jackc/pgx) (`pgxpool`), [`sqlc`](https://sqlc.dev/) (`v1.28.0`) for type-safe SQL code generation, and [`golang-migrate`](https://github.com/golang-migrate/migrate) (`v4.19.0`) compatible SQL migrations in `migrations/`.

## Enabling PostgreSQL

PostgreSQL is optional by default (`DATABASE_ENABLED=false`). When `DATABASE_ENABLED=true`:
- `DATABASE_URL` must be a valid `postgres://` or `postgresql://` connection URI.
- The server initializes a `pgxpool.Pool` at startup, verifies connectivity via a bounded ping (`DATABASE_CONNECT_TIMEOUT`), and closes the pool during graceful shutdown.
- `GET /api/v1/health/ready` checks database connectivity and returns `503 Service Unavailable` if PostgreSQL is unreachable.

## Local Development with Docker Compose

Start a local PostgreSQL 17 instance using the `postgres` Compose profile:

```bash
docker compose --profile postgres up -d postgres
```

## Database Migrations

Versioned SQL migration pairs (`NNNNNN_name.up.sql` and `NNNNNN_name.down.sql`) live in `migrations/` and are embedded via `migrations/fs.go`.

### CLI Migrations (`golang-migrate`)

For local development and deployment pipelines, use the `golang-migrate` CLI (`migrate`):

```bash
# Apply all pending migrations
make migrate-up DATABASE_URL="postgres://app:app_local_dev_password@localhost:5432/app?sslmode=disable"

# Roll back the single most recent migration
make migrate-down DATABASE_URL="postgres://app:app_local_dev_password@localhost:5432/app?sslmode=disable"
```

### Programmatic Migrations (`internal/platform/postgres`)

`postgres.ApplyMigrations` and `postgres.RollbackMigration` read the embedded `migrations.FS` files and provide `golang-migrate`-compatible semantics for isolated test schemas and programmatic bootstrapping:
- Acquires a PostgreSQL session advisory lock (`pg_advisory_lock`) to serialize concurrent migration callers.
- Tracks applied migration versions in `schema_migrations (version BIGINT PRIMARY KEY, dirty BOOLEAN NOT NULL DEFAULT FALSE)`.
- Applies only unapplied versions in ascending numerical order, recording each version inside its migration transaction (`ApplyMigrations` is idempotent).
- `RollbackMigration` rolls back only the single highest applied version and removes its row from `schema_migrations` (returning `postgres.ErrNoMigrations` when none remain).

Never run `migrate-down`, `RollbackMigration`, or destructive test migrations against a production database.

## Regenerating sqlc Code

SQL queries are checked into `internal/platform/postgres/queries/` and configured via `sqlc.yaml` (using `emit_pointers_for_null_types: true` with `pgx/v5`). Generated Go code lives in `internal/platform/postgres/sqlc/`.

After modifying schema migrations or `.sql` query files, regenerate the Go access layer using `sqlc` (`v1.28.0`) or the pinned Docker image `sqlc/sqlc:1.28.0`:

```bash
# Uses local sqlc binary if installed, or falls back to docker run sqlc/sqlc:1.28.0
make sqlc

# Or run the pinned container directly:
docker run --rm -v "${PWD}:/src" -w /src sqlc/sqlc:1.28.0 generate
```

## Integration Tests

Database integration tests in `internal/platform/postgres/integration_test.go` require `TEST_DATABASE_URL` (they never fall back to `DATABASE_URL` or a default local database). Each test run creates an isolated schema (`test_phase2_<timestamp>`), configures a dedicated pool with `search_path=<schema>,public`, tests idempotent migration application, CRUD queries, transaction rollback, and single-step migration rollback, and drops the temporary schema on cleanup.

To run against an ephemeral Docker PostgreSQL container on port `55432`:

```bash
docker run -d --rm --name go-backend-template-pg-test \
  -e POSTGRES_USER=testuser \
  -e POSTGRES_PASSWORD=testpass \
  -e POSTGRES_DB=testdb \
  -p 127.0.0.1:55432:5432 \
  postgres:17.4-alpine

TEST_DATABASE_URL="postgres://testuser:testpass@127.0.0.1:55432/testdb?sslmode=disable" \
  go test -v -count=1 ./internal/platform/postgres/...

docker stop go-backend-template-pg-test
```

## Design Rules

- Keep SQL explicit and parameterized.
- Pass `context.Context` through all database operations.
- Use `postgres.WithTx` for multi-statement transactions.
- Never log `DATABASE_URL` or include raw connection strings in error messages.
