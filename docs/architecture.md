# Architecture

The template uses a modular monolith with a small layered architecture.

## Request flow

`HTTP -> middleware -> router -> handler -> service -> repository -> dependency`

The template includes health endpoints and an example `users` CRUD module (`internal/modules/users`) backed by PostgreSQL and `sqlc`. Optional Redis, MinIO, and OpenTelemetry tracing integrations are planned implementation phases; do not assume they are already implemented.

## Folder responsibilities

- `cmd/api`: process entry point and signal handling.
- `internal/config`: environment parsing and validation.
- `internal/server`: Gin router and HTTP server.
- `internal/middleware`: request-level cross-cutting concerns.
- `internal/platform`: external integrations and observability adapters.
- `internal/modules`: business features.
- `internal/shared`: cross-module primitives that are genuinely shared.
- `migrations`: versioned SQL schema changes.
- `tests`: integration tests and test fixtures.

## Dependency rules

- Handlers do not issue SQL directly.
- Services own business rules.
- Repositories own persistence queries.
- DTOs define API contracts and should not expose database records directly.
- Create interfaces where they help isolate a dependency or test a boundary; do not create interfaces for every struct by default.
