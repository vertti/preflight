# Preflight

[![CI](https://github.com/vertti/preflight/actions/workflows/ci.yml/badge.svg)](https://github.com/vertti/preflight/actions/workflows/ci.yml)

> Sanity checks for containers. Tiny binary, zero dependencies.

## The Problem

Dockerfiles are littered with brittle shell checks:

```sh
RUN command -v node || (echo "node missing"; exit 1)
RUN [ -n "$DATABASE_URL" ] || (echo "DATABASE_URL not set"; exit 1)
```

## The Solution

```dockerfile
COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight

RUN preflight cmd node --min 18
RUN preflight env DATABASE_URL --match '^postgres://'
```

## Install

**Dockerfiles** (recommended):
```dockerfile
COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight
```

**Shell**:
```sh
curl -fsSL https://raw.githubusercontent.com/vertti/preflight/main/install.sh | sh
```

[Other install methods](docs/install.md)

## Usage

### Check commands

```sh
preflight cmd node                    # exists and runs
preflight cmd node --min 18           # minimum version
preflight cmd node --min 18 --max 22  # version range
preflight cmd node --exact 18.17.0    # exact version
preflight cmd go --version-cmd version # custom version flag
```

### Check environment variables

```sh
preflight env DATABASE_URL                       # exists and non-empty
preflight env DATABASE_URL --match '^postgres://' # matches pattern
preflight env NODE_ENV --exact production        # exact value
preflight env API_KEY --mask-value               # hide in output
```

## License

Apache 2.0
