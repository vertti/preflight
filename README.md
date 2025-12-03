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

Coming soon.

## Usage

```sh
# Check if a command exists and runs
preflight cmd node --min "18" --max "22"

# Validate environment variables
preflight env DATABASE_URL --required --regex '^postgres://'
preflight env APP_ENV --required --one-of dev,staging,production

# Check files and directories
preflight file /app/package.json --must-exist
preflight file /usr/local/bin/my-tool --must-exist --executable
```

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
