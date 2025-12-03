# Preflight

[![CI](https://github.com/vertti/preflight/actions/workflows/ci.yml/badge.svg)](https://github.com/vertti/preflight/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> Docker preflight checks for your runtime environment.

Preflight is a tiny, compiled CLI tool designed to run **sanity checks** on container and CI environments.

## What it does

- **Command checks**: Verify tools exist on PATH and can execute (not just present, but actually runnable)
- **Environment checks**: Validate environment variables are set and match expected patterns
- **File checks**: Confirm files and directories exist with correct properties

## Why?

Instead of scattering ad-hoc shell snippets like this across Dockerfiles:

```sh
RUN command -v node || (echo "node missing"; exit 1)
RUN [ -n "$DATABASE_URL" ] || (echo "DATABASE_URL not set"; exit 1)
```

Use a single, consistent tool:

```sh
RUN preflight cmd node --min "18"
RUN preflight env DATABASE_URL --required --regex '^postgres://'
```

## Installation

### Quick Install

```sh
curl -fsSL https://raw.githubusercontent.com/vertti/preflight/main/install.sh | sh
```

### In Dockerfiles

```dockerfile
# Add preflight to your container
RUN curl -fsSL https://github.com/vertti/preflight/releases/latest/download/preflight-linux-amd64 \
    -o /usr/local/bin/preflight && chmod +x /usr/local/bin/preflight

# Use it to verify your environment
RUN preflight cmd node --min 18
```

### Manual Download

Download the binary for your platform from [GitHub Releases](https://github.com/vertti/preflight/releases):

```sh
# Linux (amd64)
curl -fsSL https://github.com/vertti/preflight/releases/latest/download/preflight-linux-amd64 \
  -o /usr/local/bin/preflight && chmod +x /usr/local/bin/preflight

# macOS (Apple Silicon)
curl -fsSL https://github.com/vertti/preflight/releases/latest/download/preflight-darwin-arm64 \
  -o /usr/local/bin/preflight && chmod +x /usr/local/bin/preflight
```

### From Source

```sh
go install github.com/vertti/preflight/cmd/preflight@latest
```

## Usage

```sh
# Check if a command exists and can run
preflight cmd node

# Check with version constraints
preflight cmd node --min 18 --max 22

# Use custom version command (for tools that don't use --version)
preflight cmd go --version-cmd "version"
```

### Coming Soon

- `preflight env` - Validate environment variables
- `preflight file` - Check files and directories

## Development

### Prerequisites

- Go 1.25+ (use [mise](https://mise.jdx.dev/) for version management)
- golangci-lint

### Setup

```sh
# Install Go via mise
mise install

# Run tests
make test

# Run linter
make lint

# Build
make build
```

## License

Apache 2.0 - see [LICENSE](LICENSE) for details.
