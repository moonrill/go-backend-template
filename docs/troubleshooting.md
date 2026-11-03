# Troubleshooting

## Configuration error

Check duration values such as `APP_READ_TIMEOUT` and ensure `LOG_LEVEL` is one of `debug`, `info`, `warn`, or `error`.

## Port already in use

Change `APP_PORT` or stop the process already listening on the configured port.

## Metrics endpoint unavailable

Confirm `METRICS_ENABLED=true` and that the server has restarted after configuration changes.

## Dependency integrations

PostgreSQL (with Users CRUD), Redis, and MinIO are implemented. OpenTelemetry tracing is reserved for Phase 6. Do not enable flags for uncompleted phases until those phases are implemented.
