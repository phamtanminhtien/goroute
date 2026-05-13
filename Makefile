.PHONY: fmt test build run

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

build:
	go build ./...

run:
	go run ./cmd/goroute
