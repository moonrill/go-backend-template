# Security baseline

- Never commit credentials, tokens, or production `.env` files.
- Use parameterized SQL queries.
- Apply explicit CORS allowlists when browser access is needed.
- Configure trusted proxies explicitly before trusting forwarded headers.
- Keep metrics endpoints private or restrict them at the network layer.
- Ensure bounded metric label cardinality: HTTP request counters and duration histograms strictly use Gin parameterized route patterns (or `UNMATCHED`) and bounded HTTP method normalization (`normalizeMethod`) to prevent DoS via cardinality explosion.
- Maintain immutable API documentation specifications: Swagger/OpenAPI specs are served statically without per-request mutation of global singleton state (`docs.SwaggerInfo`) and without trusting unvalidated client `Host` or `X-Forwarded-Proto` headers.
- Harden local monitoring services: Grafana in Docker Compose binds exclusively to the local loopback interface (`127.0.0.1`), disables anonymous authentication by default, and enforces an explicitly configured strong administrator password.
- Avoid logging authorization headers, secrets, request bodies, or sensitive personal data.
- Validate request payloads and set explicit request-body limits for upload endpoints.
- Use TLS for public traffic and for sensitive service-to-service traffic.
- Run containers as a non-root user with dropped Linux capabilities where possible.
- Keep Go and dependencies patched; review vulnerability reports before releases.
- Authentication and authorization are intentionally not enabled by default. Add them before exposing protected business endpoints.
