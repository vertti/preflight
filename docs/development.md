# Development

## Prerequisites

- Go 1.25+ (use [mise](https://mise.jdx.dev/) for version management)
- golangci-lint

## Setup

```sh
# Install dependencies via mise
mise install

# Run tests
make test

# Run linter
make lint

# Build
make build
```

## Project Structure

```
preflight/
  cmd/preflight/     # CLI entrypoint
  pkg/
    check/           # Core types (Result, Status)
    cmdcheck/        # Command/binary checks
    envcheck/        # Environment variable checks
    version/         # Version parsing and comparison
```

## Workflow

1. **TDD Approach**: Write tests first, then implementation
2. **Small Steps**: Each commit should be focused and atomic
3. **Always Lint**: Run `make lint` before committing

## Pull Requests

- Use feature branches
- Keep PRs focused on a single feature/fix
- CI must pass before merging
