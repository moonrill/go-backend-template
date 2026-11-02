# MinIO / S3-compatible storage

Object storage is optional and planned for a later phase using AWS SDK for Go v2 against an S3-compatible endpoint.

Production considerations:
- Use least-privilege credentials.
- Prefer TLS outside trusted local networks.
- Validate content and enforce upload size limits.
- Do not expose buckets publicly by default.
- Avoid using untrusted client filenames directly as object keys.
- Do not enable `MINIO_ENABLED` until the integration is implemented.
