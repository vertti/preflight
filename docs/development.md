# Development

## Prerequisites

- Go 1.25+ (use [mise](https://mise.jdx.dev/) for version management)
- golangci-lint

## Setup

```sh
# Install dependencies via mise
mise install

# Install pre-commit hooks (runs gofmt, golangci-lint, dprint on commit)
hk install

# Run tests
mise run test

# Run linter
mise run lint

# Build
mise run build
```

Run `mise tasks` to list all available tasks.

## Project Structure

```
preflight/
  cmd/preflight/     # CLI entrypoint, one cmd_<name>.go per subcommand
  pkg/
    check/           # Core types (Result, Status) shared by every check
    <name>check/     # One package per subcommand: cmd, env, file, git, hash,
                     # http, json, prom, resource, sys, tcp, user
    exec/            # exec() passthrough for entrypoint mode
    jsonpath/        # Minimal JSON path lookup used by json and http
    output/          # Result rendering, colour and CI detection
    preflightfile/   # .preflight file discovery and parsing
    version/         # Version parsing and comparison
    testutil/        # Shared test helpers
```

## Workflow

1. **TDD Approach**: Write tests first, then implementation
2. **Small Steps**: Each commit should be focused and atomic
3. **Always Lint**: Run `mise run lint` before committing

## Pull Requests

- Use feature branches
- Keep PRs focused on a single feature/fix
- CI must pass before merging
