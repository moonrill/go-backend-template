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
- `GET /api/v1/health/ready` (evaluates PostgreSQL, Redis, and MinIO readiness when their respective flags are enabled)
- `GET /api/v1/users`, `GET /api/v1/users/:id`, `POST /api/v1/users`, `PATCH /api/v1/users/:id`, `DELETE /api/v1/users/:id` (when `DATABASE_ENABLED=true`; see [`docs/openapi.yaml`](docs/openapi.yaml))
- `GET /metrics` when metrics are enabled
- OpenTelemetry distributed tracing with OTLP exporter, W3C context propagation, trace-log correlation, and client instrumentation (PostgreSQL, Redis, MinIO)

## Configuration

See [Configuration](docs/configuration.md), [Database](docs/database.md), [Redis](docs/redis.md), [MinIO](docs/minio.md), and [Observability](docs/observability.md).

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
