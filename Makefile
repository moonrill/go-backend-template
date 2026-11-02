.PHONY: run fmt test vet build check sqlc migrate-up migrate-down

SQLC_IMAGE ?= sqlc/sqlc:1.28.0

run:
	go run ./cmd/api

fmt:
	gofmt -w .

test:
	go test ./...

vet:
	go vet ./...

build:
	go build ./...

sqlc:
	@if command -v sqlc >/dev/null 2>&1; then \
		sqlc generate; \
	else \
		docker run --rm -v "$(CURDIR):/src" -w /src $(SQLC_IMAGE) generate; \
	fi

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

check: test vet build
