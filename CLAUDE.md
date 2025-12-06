# Development Guidelines

## Workflow

- TDD: Write tests first
- Work in branches, PRs to main (never commit directly)
- Commit often, keep commits atomic

## Before Committing

```sh
make fmt && make lint && make test
```

## Commit Messages

- `Add X` - new feature
- `Fix X` - bug fix
- `Update X` - enhancement
- `Remove X` - deletion

## PRs

- No "Test plan" sections
- Run tests before opening

## Platform-Specific Code

When using syscalls/OS APIs:

- Interface defined **once** in common file (no build tag)
- Platform methods in `*_unix.go`, `*_windows.go`
- Use `//go:build unix` for darwin/linux shared code
- Unsupported features return errors, not panic
- Test with: `GOOS=windows go build ./...`

## Adding New Commands

1. `pkg/<name>check/check.go` - Check struct, interface, Run(), Real* impl
2. `pkg/<name>check/check_test.go` - mock + table-driven tests
3. `cmd/preflight/cmd_<name>.go` - cobra command + flags
4. `integration_test.go` - real system test
5. `docs/usage.md` - document flags and examples
