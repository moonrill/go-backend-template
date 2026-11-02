# Getting started

1. Install a supported Go release compatible with `go.mod`.
2. Copy `.env.example` to `.env` and review the values.
3. Run `go mod tidy`.
4. Run `go test ./...`.
5. Run `go run ./cmd/api` (automatically loads `.env` if present).
6. Check `GET http://localhost:8080/api/v1/health/live`.
