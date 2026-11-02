# Configuration

The application reads environment variables. Copy `.env.example` to `.env` for local development. The application does not currently load `.env` files automatically; export variables in the shell or use Compose's `env_file`.

| Variable | Default | Description |
|---|---|---|
| `APP_NAME` | `go-backend-template` | Application identifier (must not be empty) |
| `APP_ENV` | `development` | `development`, `staging`, `production`, or `test` (`production` enables Gin release mode) |
| `APP_PORT` | `8080` | HTTP listen port (`1`–`65535`) |
| `APP_READ_TIMEOUT` | `10s` | Maximum request read duration (positive duration) |
| `APP_WRITE_TIMEOUT` | `15s` | Maximum response write duration (positive duration) |
| `APP_IDLE_TIMEOUT` | `60s` | Keep-alive idle timeout (positive duration) |
| `APP_SHUTDOWN_TIMEOUT` | `15s` | Graceful shutdown deadline (positive duration) |
| `APP_TRUSTED_PROXIES` | *(empty)* | Comma-separated trusted proxy IPs or CIDRs; when empty, forwarded IP headers are ignored |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `METRICS_ENABLED` | `true` | Exposes `/metrics` (`true` or `false`) |
| `TRACING_ENABLED` | `false` | Reserved for tracing integration phase |
| `DATABASE_ENABLED` | `false` | Reserved for PostgreSQL integration phase |
| `REDIS_ENABLED` | `false` | Reserved for Redis integration phase |
| `MINIO_ENABLED` | `false` | Reserved for MinIO / object storage integration phase |

Invalid values for booleans, durations, ports, environment names, log levels, or trusted proxy entries return an explicit startup error rather than silently falling back to defaults. Enabling `DATABASE_ENABLED`, `REDIS_ENABLED`, `MINIO_ENABLED`, or `TRACING_ENABLED` before their implementation phase also returns a startup error.

Never commit real credentials. Prefer a secret manager or deployment platform secrets for production.
