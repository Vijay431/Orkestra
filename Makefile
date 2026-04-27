.PHONY: test test-unit test-integration test-e2e test-all cover

test:
	go test ./internal/... ./cmd/...

test-unit:
	go test ./internal/ticket/... ./internal/toon/...

test-integration:
	go test ./internal/mcp/... ./cmd/server/...

test-e2e:
	go test -tags e2e -timeout 60s ./test/e2e/...

test-all: test test-e2e

cover:
	go test -coverprofile=coverage.out ./internal/... ./cmd/...
	go tool cover -html=coverage.out -o coverage.html
