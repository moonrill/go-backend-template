# Security baseline

- Never commit credentials, tokens, or production `.env` files.
- Use parameterized SQL queries.
- Apply explicit CORS allowlists when browser access is needed.
- Configure trusted proxies explicitly before trusting forwarded headers.
- Keep metrics endpoints private or restrict them at the network layer.
- Avoid logging authorization headers, secrets, request bodies, or sensitive personal data.
- Validate request payloads and set explicit request-body limits for upload endpoints.
- Use TLS for public traffic and for sensitive service-to-service traffic.
- Run containers as a non-root user with dropped Linux capabilities where possible.
- Keep Go and dependencies patched; review vulnerability reports before releases.
- Authentication and authorization are intentionally not enabled by default. Add them before exposing protected business endpoints.
