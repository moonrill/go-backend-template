# Observability

## Logging

The initial scaffold uses `log/slog` with JSON output. Request logs include request ID, method, path, status, latency, and client IP. Review client IP trust configuration before deploying behind proxies.

## Health

- `GET /api/v1/health/live`: process liveness.
- `GET /api/v1/health/ready`: readiness. Add checks for only the dependencies required by the active application configuration.

## Metrics

When `METRICS_ENABLED=true`, `GET /metrics` serves Prometheus metrics. Restrict access to this endpoint at the network or proxy layer.

## Tracing

OpenTelemetry tracing is planned and disabled by default. It should be added without making an exporter mandatory for local development.
