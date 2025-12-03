# Development Guidelines

## Workflow

1. **TDD Approach**: Write tests first, then implementation
2. **Small Steps**: Each commit should be focused and atomic
3. **Commit Often**: Working code gets committed immediately
4. **Always Lint/Format**: Run before every commit

## Before Committing

```sh
make fmt && make lint && make test
```

## Commit Message Style

Use conventional style:

- `Add X` - new feature or file
- `Fix X` - bug fix
- `Update X` - enhancement to existing feature
- `Remove X` - deletion

Keep messages concise (1 line preferred).

## Project Structure

```
preflight/
  cmd/preflight/     # CLI entrypoint
  pkg/
    check/           # Core types (CheckResult, Status)
    cmdcheck/        # Command/binary checks
    envcheck/        # Environment variable checks (future)
    filecheck/       # File/directory checks (future)
    version/         # Version parsing and comparison
```

## Testing

- Use table-driven tests where appropriate
- Mock external dependencies (exec, filesystem, env)
- Aim for high coverage on pkg/* code

## Pull Requests

- Do NOT add "Test plan" sections to PR descriptions
- Run tests yourself before opening PR
- If manual testing is needed, tell the user how to test
