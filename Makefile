.PHONY: all build test test-coverage lint fmt clean

# Build the preflight binary
build:
	go build -o bin/preflight ./cmd/preflight

# Run tests with race detection
test:
	go test -race ./...

# Run tests with coverage report
test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linters
lint:
	golangci-lint run

# Format code
fmt:
	goimports -w .

# Run all checks (lint, test, build)
all: lint test build

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html
