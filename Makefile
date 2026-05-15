.PHONY: fmt test build run run-with-web check web-install web-dev web-typecheck web-build web-preview

WEB_DIR := ./web

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

build:
	go build ./...

run:
	go run ./cmd/goroute

run-with-web: web-build
	go run ./cmd/goroute

check: test web-typecheck

web-install:
	pnpm --dir $(WEB_DIR) install

web-dev:
	pnpm --dir $(WEB_DIR) dev

web-typecheck:
	pnpm --dir $(WEB_DIR) typecheck

web-build:
	pnpm --dir $(WEB_DIR) build

web-preview:
	pnpm --dir $(WEB_DIR) preview
