# Troubleshooting

## Configuration error

Check duration values such as `APP_READ_TIMEOUT` and ensure `LOG_LEVEL` is one of `debug`, `info`, `warn`, or `error`.

## Port already in use

Change `APP_PORT` or stop the process already listening on the configured port.

## Metrics endpoint unavailable

Confirm `METRICS_ENABLED=true` and that the server has restarted after configuration changes.

## Dependency integrations

PostgreSQL, Redis, MinIO, tracing, and CRUD are not yet implemented in this initial scaffold. Do not enable their flags until the relevant phase is complete.
