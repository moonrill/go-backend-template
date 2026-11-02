# Testing

Use the standard library `testing` package.

```bash
go test ./...
go test -race ./...
go vet ./...
```

Add unit tests for configuration, middleware, services, and validation. Integration tests should use explicitly configured test services and must never target production resources.
