# Deployment

The Dockerfile uses a multi-stage build and a minimal `scratch` runtime. Confirm CA certificates and any required runtime assets before adding outbound TLS calls. Pin the builder image to a reviewed patch release before production use.

Compose is intended for local development. Production secrets should be injected by the deployment environment rather than committed to files.

The container runs as UID/GID 65532 and should be deployed with restricted network access, resource limits, and a private metrics endpoint.
