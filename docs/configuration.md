# Configuration

The application reads environment variables and automatically loads a local `.env` file if present (existing shell environment variables take precedence over `.env`). Copy `.env.example` to `.env` for local development.

| Variable | Default | Description |
|---|---|---|
| `APP_NAME` | `go-backend-template` | Application identifier (must not be empty) |
| `APP_ENV` | `development` | `development` / `staging` (Gin `debug` mode), `production` (Gin `release` mode), or `test` (Gin `test` mode) |
| `APP_PORT` | `8080` | HTTP listen port (`1`–`65535`) |
| `APP_READ_TIMEOUT` | `10s` | Maximum request read duration (positive duration) |
| `APP_WRITE_TIMEOUT` | `15s` | Maximum response write duration (positive duration) |
| `APP_IDLE_TIMEOUT` | `60s` | Keep-alive idle timeout (positive duration) |
| `APP_SHUTDOWN_TIMEOUT` | `15s` | Graceful shutdown deadline (positive duration) |
| `APP_TRUSTED_PROXIES` | *(empty)* | Comma-separated trusted proxy IPs or CIDRs; when empty, forwarded IP headers are ignored |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `METRICS_ENABLED` | `true` | Exposes `/metrics` (`true` or `false`) |
| `DATABASE_ENABLED` | `false` | Enables PostgreSQL connection pool and readiness check |
| `DATABASE_URL` | *(empty)* | PostgreSQL connection URI (`postgres://...`); required when `DATABASE_ENABLED=true` |
| `DATABASE_MAX_CONNS` | `10` | Maximum connections in the pool (`>= 1`) |
| `DATABASE_MIN_CONNS` | `1` | Minimum idle connections in the pool (`0 <= min <= max`) |
| `DATABASE_CONNECT_TIMEOUT` | `3s` | Connection and health-check ping timeout |
| `DATABASE_MAX_CONN_LIFETIME` | `30m` | Maximum lifetime of a pooled connection |
| `DATABASE_MAX_CONN_IDLE_TIME` | `5m` | Maximum idle time of a pooled connection |
| `DATABASE_HEALTH_CHECK_PERIOD` | `1m` | Background pool health check interval |
| `REDIS_ENABLED` | `false` | Enables Redis client and readiness check (`true` or `false`) |
| `REDIS_URL` | *(empty)* | Redis connection URI (`redis://...` or `rediss://...`; e.g., `redis://127.0.0.1:6379/0` on host, `redis://redis:6379/0` in Docker Compose, `redis://redis-service.default.svc.cluster.local:6379/0` in Kubernetes); required when `REDIS_ENABLED=true` |
| `REDIS_PASSWORD` | *(empty)* | Optional password override (overrides password embedded in `REDIS_URL` if non-empty) |
| `REDIS_DB` | `0` | Redis logical database index (`>= 0`) |
| `REDIS_POOL_SIZE` | `10` | Maximum socket connections in client pool (`>= 1`) |
| `REDIS_DIAL_TIMEOUT` | `5s` | Connection dial timeout (positive duration; falls back to legacy `REDIS_CONNECT_TIMEOUT` if set) |
| `REDIS_READ_TIMEOUT` | `3s` | Socket read deadline (positive duration) |
| `REDIS_WRITE_TIMEOUT` | `3s` | Socket write deadline (positive duration) |
| `MINIO_ENABLED` | `false` | Enables MinIO/S3 object storage client and readiness check (`true` or `false`) |
| `MINIO_ENDPOINT` | *(empty)* | MinIO/S3 endpoint without scheme (e.g. `127.0.0.1:9000`, `minio:9000`); required when `MINIO_ENABLED=true` |
| `MINIO_REGION` | `us-east-1` | S3 region identifier |
| `MINIO_BUCKET` | *(empty)* | Target bucket name (3-63 chars, lowercase); required when `MINIO_ENABLED=true` |
| `MINIO_ACCESS_KEY` | *(empty)* | MinIO/S3 access key; required when `MINIO_ENABLED=true` |
| `MINIO_SECRET_KEY` | *(empty)* | MinIO/S3 secret key; required when `MINIO_ENABLED=true` |
| `MINIO_USE_SSL` | `true` | Enables TLS (`true` for HTTPS/production, `false` for local dev) |
| `MINIO_CONNECT_TIMEOUT` | `5s` | Connection and bucket health check timeout (positive duration) |
| `MINIO_AUTO_CREATE_BUCKET` | `true` | Automatically creates bucket on startup if it does not exist |
| `TRACING_ENABLED` | `false` | Enables OpenTelemetry distributed tracing (`true` or `false`) |
| `OTEL_SDK_DISABLED` | `true` | OpenTelemetry standard toggle (`true` disables SDK) |
| `OTEL_SERVICE_NAME` | `go-backend-template` | Service name attached to resource attributes |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `grpc` | OTLP exporter protocol (`grpc` or `http/protobuf`) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | *(empty)* | OTLP collector endpoint (e.g. `http://alloy:4318` or `alloy:4317`) |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | *(empty)* | OTLP traces collector endpoint override |
| `OTEL_TRACES_SAMPLER` | `parentbased_traceidratio` | Sampler type (`always_on`, `always_off`, `traceidratio`, `parentbased_always_on`, `parentbased_always_off`, `parentbased_traceidratio`) |
| `OTEL_TRACES_SAMPLER_ARG` | `1.0` | Sampler ratio argument between `0.0` and `1.0` |
| `OTEL_RESOURCE_ATTRIBUTES` | *(empty)* | Custom resource attributes (comma-separated `key=value`) |

Invalid values for booleans, integers, durations, ports, environment names, log levels, database URLs, Redis URLs, MinIO endpoints/buckets, tracing protocols/samplers, or trusted proxy entries return an explicit startup error rather than silently falling back to defaults.

Never commit real credentials. Prefer a secret manager or deployment platform secrets for production.
