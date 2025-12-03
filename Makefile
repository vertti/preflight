.PHONY: all build test test-coverage lint fmt fmt-md check-md clean install-hooks

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

# Format markdown files
fmt-md:
	dprint fmt

# Check markdown formatting
check-md:
	dprint check

# Run all checks (lint, test, build)
all: lint test build

# Install pre-commit hooks
install-hooks:
	hk install --mise

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html
