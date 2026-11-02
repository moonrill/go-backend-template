# Go Backend Template

A lightweight, modular REST API starter built with Go and Gin.

## Goals

- Consistent handler/service/repository boundaries.
- PostgreSQL via `pgx/v5`, `sqlc`, and versioned SQL migrations.
- Optional PostgreSQL, Redis, and S3-compatible object storage.
- Structured logging, health checks, optional Prometheus metrics and tracing.
- Secure defaults, graceful shutdown, Docker-based development.
- Documentation and an example CRUD module.

## Requirements

- A supported Go release compatible with `go.mod`.
- Docker Engine and Docker Compose (optional, for local dependencies).

## Quick start

1. Copy `.env.example` to `.env` and review the values.
2. Download dependencies: `go mod tidy`
3. Run tests: `go test ./...`
4. Start the API: `go run ./cmd/api`

The scaffold currently exposes:
- `GET /api/v1/health/live`
- `GET /api/v1/health/ready` (includes PostgreSQL readiness when `DATABASE_ENABLED=true`)
- `GET /api/v1/users`, `GET /api/v1/users/:id`, `POST /api/v1/users`, `PATCH /api/v1/users/:id`, `DELETE /api/v1/users/:id` (when `DATABASE_ENABLED=true`; see [`docs/openapi.yaml`](docs/openapi.yaml))
- `GET /metrics` when metrics are enabled

Optional Redis, MinIO, and tracing integrations are added in subsequent implementation phases. Until then, do not enable a dependency that has not yet been implemented.

## Configuration

See [Configuration](docs/configuration.md) and [Database](docs/database.md).

## Architecture

See [Architecture](docs/architecture.md).

## Security

See [Security](docs/security.md).

## Development

```bash
make fmt
make test
make vet
make build
make run
```

## License

MIT. Review and update `LICENSE` before distributing this template.
