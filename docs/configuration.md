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
| `TRACING_ENABLED` | `false` | Reserved for tracing integration phase |
| `REDIS_ENABLED` | `false` | Reserved for Redis integration phase |
| `MINIO_ENABLED` | `false` | Reserved for MinIO / object storage integration phase |

Invalid values for booleans, integers, durations, ports, environment names, log levels, database URLs, or trusted proxy entries return an explicit startup error rather than silently falling back to defaults. Enabling `REDIS_ENABLED`, `MINIO_ENABLED`, or `TRACING_ENABLED` before their implementation phase also returns a startup error.

Never commit real credentials. Prefer a secret manager or deployment platform secrets for production.
