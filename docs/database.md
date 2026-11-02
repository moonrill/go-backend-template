# Database

PostgreSQL integration using pgx, sqlc, and versioned SQL migrations is planned for the database implementation phase.

Design rules:
- Keep SQL explicit and parameterized.
- Pass `context.Context` through repository operations.
- Use migrations for schema changes.
- Keep transactions explicit and scoped.
- Do not enable `DATABASE_ENABLED` until the integration is implemented.
