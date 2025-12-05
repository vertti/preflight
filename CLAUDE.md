# Development Guidelines

## Workflow

1. **TDD Approach**: Write tests first, then implementation
2. **Small Steps**: Each commit should be focused and atomic
3. **Commit Often**: Working code gets committed immediately
4. **Always Lint/Format**: Run before every commit
5. **Work in Branches**: Each new feature gets its own branch and PR
6. **Never commit directly to main**: Always use PRs for code changes

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
- For new commands: add full unit test coverage AND one integration test in `integration_test.go`

## Adding New Commands

When implementing a new preflight command:

1. Create `pkg/<name>check/check.go` with interface for testability
2. Create `pkg/<name>check/check_test.go` with comprehensive unit tests
3. Create `cmd/preflight/cmd_<name>.go` for CLI wiring
4. Add one integration test to `integration_test.go`
5. Update `docs/usage.md` with examples
6. Update `README.md` if the command is noteworthy
7. If interesting, show the ugly shell script this feature replaces

## Pull Requests

- Do NOT add "Test plan" sections to PR descriptions
- Run tests yourself before opening PR
- If manual testing is needed, tell the user how to test
