# Getting started

1. Install a supported Go release compatible with `go.mod`.
2. Copy `.env.example` to `.env`.
3. Export environment variables or run with Compose.
4. Run `go mod tidy`.
5. Run `go test ./...`.
6. Run `go run ./cmd/api`.
7. Check `GET http://localhost:8080/api/v1/health/live`.

The `.env` file is not automatically loaded by the Go process in this scaffold. Compose can load it through `env_file`; otherwise export variables in your shell.
