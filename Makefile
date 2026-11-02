.PHONY: run fmt test vet build check

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

check: test vet build
