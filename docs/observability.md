# Observability

This template provides an end-to-end observability architecture covering the three pillars: **Metrics**, **Logs**, and **Distributed Tracing**, modeled after modern cloud-native standards (Prometheus, OpenTelemetry, Grafana Tempo, Loki, and Grafana Alloy).

---

## 1. Architecture Overview

```
                      +-------------------+
                      |   HTTP Request    |
                      +---------+---------+
                                |
                                v
               +---------------------------------+
               |        Gin HTTP Server          |
               |  (otelgin + Prometheus + slog)  |
               +---+------------+------------+---+
                   |            |            |
       Trace Context (W3C)      |         Metrics
       (trace_id, span_id)      |      (/metrics)
                   |            |            |
                   v            v            v
          +------------+   +---------+  +------------+
          | PostgreSQL |   |  Redis  |  |   MinIO    |
          | (otelpgx)  |   |(redisotel)|(otelhttp)  |
          +------------+   +---------+  +------------+
                   |            |            |
                   +------------+------------+
                                |
                    OTLP (gRPC / HTTP Protobuf)
                                |
                                v
                   +-------------------------+
                   |      Grafana Alloy      |
                   +------------+------------+
                                |
             +------------------+------------------+
             | Traces                              | Logs (Docker)
             v                                     v
     +---------------+                     +---------------+
     | Grafana Tempo |                     | Grafana Loki  |
     +-------+-------+                     +-------+-------+
             |                                     |
             +------------------+------------------+
                                |
                                v
                    +-----------------------+
                    |    Grafana Dashboards |
                    |      (Port 3000)      |
                    +-----------------------+
```

---

## 2. Distributed Tracing (OpenTelemetry)

Distributed tracing is managed by the `internal/otel` package and Gin middleware (`otelgin`).

### Key Capabilities
- **W3C Standard Propagation**: Supports standard `traceparent`, `tracestate`, and `baggage` headers across distributed boundaries.
- **Protocol Flexibility**: Supports both `grpc` (port 4317) and `http/protobuf` (port 4318) OTLP protocols.
- **Configurable Samplers**:
  - `parentbased_traceidratio` (default, recommended)
  - `parentbased_always_on`
  - `parentbased_always_off`
  - `always_on`
  - `always_off`
  - `traceidratio`
- **Graceful Degradation**: If the OTLP collector is unreachable or fails to initialize, the application logs a warning and falls back to a no-op tracer provider without crashing or interrupting traffic.
- **Automatic Client Instrumentation**:
  - **PostgreSQL**: Queries, transactions, and connection pools instrumented via `github.com/exaring/otelpgx`.
  - **Redis**: Commands and pipelining instrumented via `github.com/redis/go-redis/extra/redisotel/v9`.
  - **MinIO / S3**: HTTP transport instrumented via `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`.

### Configuration Variables
| Variable | Description | Default |
| --- | --- | --- |
| `TRACING_ENABLED` | Toggles distributed tracing (`true` / `false`) | `false` |
| `OTEL_SDK_DISABLED` | OpenTelemetry standard flag (`true` disables SDK) | `true` |
| `OTEL_SERVICE_NAME` | Service identifier reported in traces | `go-backend-template` |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | Exporter transport: `grpc` or `http/protobuf` | `grpc` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector base endpoint URL | `http://alloy:4318` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Direct trace exporter endpoint override | *empty* |
| `OTEL_TRACES_SAMPLER` | Head sampling strategy | `parentbased_traceidratio` |
| `OTEL_TRACES_SAMPLER_ARG` | Sampling ratio between `0.0` and `1.0` | `1.0` |
| `OTEL_RESOURCE_ATTRIBUTES` | Comma-separated resource attributes (`k=v`) | `deployment.environment=local` |

---

## 3. Structured Logging & Trace Correlation

Request logs use standard library `log/slog` in JSON format.

### Trace-Log Correlation
When tracing is active, the access logging middleware automatically extracts the active trace and span IDs and injects them into both the structured log record and the response headers:

```json
{
  "time": "2026-11-03T12:00:00.000Z",
  "level": "INFO",
  "msg": "http request",
  "request_id": "req_8a2d1f7e-1234-4567-89ab-cdef01234567",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "method": "GET",
  "path": "/api/v1/health/live",
  "status": 200,
  "latency_ms": 1,
  "client_ip": "127.0.0.1"
}
```

Response header `X-Trace-ID` is returned to clients for end-to-end request tracing.

---

## 4. Prometheus Metrics

When `METRICS_ENABLED=true`, Prometheus metrics are exposed at `GET /metrics`.

### Core HTTP RED Metrics
| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `http_requests_total` | Counter | `method`, `route`, `status` | Total HTTP requests handled |
| `http_request_duration_seconds` | Histogram | `method`, `route`, `status` | Request latency in seconds |

> **Cardinality & DoS Protection**: To prevent memory exhaustion attacks, metric labels are strictly bounded. The `route` label uses matched Gin route templates (e.g. `/api/v1/users/:id`), falling back to `UNMATCHED` for unrouted paths. HTTP methods are normalized against a bounded allowlist (`GET`, `POST`, `PUT`, `DELETE`, etc.), with unrecognized methods collapsed to `UNKNOWN`.

In addition, standard Go runtime metrics (`go_goroutines`, `go_memstats_heap_alloc_bytes`) and process metrics (`process_cpu_seconds_total`, `process_resident_memory_bytes`) are automatically exported.

---

## 5. Pre-configured Observability Stack (Docker Compose)

The repository provides a complete local observability stack in `compose.yaml` under the `monitoring` profile:

| Service | Container Port | Host Port | Purpose |
| --- | --- | --- | --- |
| **Prometheus** | `9090` | `9090` | Metrics scraping and storage |
| **Grafana** | `3000` | `127.0.0.1:3000` | Dashboards and visualization (Admin login required) |
| **Loki** | `3100` | `3100` | Log aggregation and indexing |
| **Tempo** | `3200`, `4317`, `4318` | `3200` | Distributed tracing storage |
| **Grafana Alloy** | `12345`, `4317`, `4318` | `12345`, `14317`, `14318` | Telemetry collector / log forwarder |
| **k6** | *N/A* | *N/A* | Automated load test runner |

### Starting the Observability Stack

```bash
# Ensure GRAFANA_ADMIN_PASSWORD is configured in .env before starting
# Start the monitoring stack alongside PostgreSQL, Redis, and MinIO
docker compose --profile monitoring --profile postgres --profile redis --profile minio up -d

# Start only the application and monitoring stack
docker compose --profile monitoring up -d api
```

### Accessing the Grafana Dashboard
1. Open `http://localhost:3000` in your browser (bound to `127.0.0.1`).
2. Sign in with username `admin` (or `GRAFANA_ADMIN_USER`) and the password configured via `GRAFANA_ADMIN_PASSWORD` in `.env`. Anonymous access is disabled by default.
3. The **API Observability Dashboard** (`config/grafana/dashboard.json`) is automatically provisioned and includes:
   - Request Rate (RPS)
   - Latency Percentiles (P50, P95, P99)
   - Average Request Latency
   - Error Rate (%)
   - HTTP Status Code Distribution
   - Go Goroutines and Heap Memory
   - Process CPU Usage and Resident Memory
   - Embedded Loki Live Logs viewer with trace ID correlation

### Running Load Tests with k6
To generate synthetic traffic and test latency percentiles:

```bash
docker compose --profile monitoring run --rm k6
```
Or directly from the host:
```bash
k6 run config/k6/loadtest.js
```
