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

## Pull Requests

- Do NOT add "Test plan" sections to PR descriptions
- Run tests yourself before opening PR
- If manual testing is needed, tell the user how to test

---

# Codebase Architecture

## High-Level Structure

```
preflight/
├── cmd/preflight/          # CLI layer (~1,200 lines)
│   ├── main.go             # Entry point, hashbang detection
│   ├── root.go             # Cobra root command setup
│   ├── run.go              # runCheck() helper, shared by all commands
│   ├── validation.go       # Flag validation helpers
│   └── cmd_*.go            # One file per command (11 commands)
│
├── pkg/                    # Core packages (~6,500 lines)
│   ├── check/              # Shared types: Result, Status, Checker interface
│   ├── output/             # Terminal output formatting with colors
│   ├── version/            # Semantic version parsing & comparison
│   ├── preflightfile/      # .preflight file discovery & parsing
│   └── *check/             # 11 check implementations (see below)
│
├── integration_test.go     # Real system integration tests
└── docs/usage.md           # Command documentation
```

## Core Design Patterns

### Pattern 1: Interface Injection for Testability

Every check package defines an interface for external dependencies:

```go
// pkg/filecheck/fs.go
type FileSystem interface {
    Stat(name string) (fs.FileInfo, error)
    ReadFile(name string, limit int64) ([]byte, error)
}

type RealFileSystem struct{}  // Production implementation
```

Check structs take the interface as a field, enabling mock-based unit testing:

```go
type Check struct {
    Path string
    FS   FileSystem  // Injected: &RealFileSystem{} in prod, mock in tests
}
```

### Pattern 2: Check struct + Run() → Result

All checks implement the same pattern:

```go
// pkg/check/checker.go
type Checker interface {
    Run() Result
}

// Every check package:
func (c *Check) Run() check.Result {
    result := check.Result{Name: "type: identifier"}

    // Validation logic...
    if failed {
        return result.Fail("reason", err)
    }

    result.Status = check.StatusOK
    return result
}
```

### Pattern 3: Result Helper Methods

`pkg/check/helpers.go` provides a builder pattern:

```go
result.Fail(detail, err)           // Set FAIL status, append detail
result.Failf(format, args...)      // Sprintf version
result.AddDetail(s)                // Append info detail (non-failing)
result.AddDetailf(format, args...) // Sprintf version
check.CompileRegex(pattern)        // Returns nil if pattern empty
```

### Pattern 4: CLI Command Structure

Each `cmd_*.go` follows this structure:

```go
var (
    flagName bool  // Flag variables at package level
)

var cmdName = &cobra.Command{
    Use:   "name <arg>",
    Short: "Description",
    Args:  cobra.ExactArgs(1),
    RunE:  runNameCheck,
}

func init() {
    cmdName.Flags().BoolVar(&flagName, "flag", false, "description")
    rootCmd.AddCommand(cmdName)
}

func runNameCheck(_ *cobra.Command, args []string) error {
    c := &namecheck.Check{
        Field:  args[0],
        Flag:   flagName,
        Runner: &namecheck.RealRunner{},  // Inject real implementation
    }
    return runCheck(c)  // Shared helper in run.go
}
```

### Pattern 5: Flag Validation

`cmd/preflight/validation.go` provides:

```go
requireExactlyOne(flagValue...)  // Mutually exclusive flags
requireAtLeastOne(flagSet...)    // At least one required
```

### Pattern 6: Platform-Specific Code

When code requires platform-specific implementations (syscalls, OS APIs):

1. **Common file** (no build tag): Interface + struct + cross-platform methods
2. **Platform files** (`*_unix.go`, `*_windows.go`): Platform-specific method implementations

```
pkg/filecheck/
├── fs.go           # Interface + struct + Stat() + ReadFile()
├── fs_unix.go      # //go:build unix   → GetOwner() using syscall.Stat_t
└── fs_windows.go   # //go:build windows → GetOwner() returns error

pkg/resourcecheck/
├── resource_common.go   # Interface + struct + NumCPUs()
├── resource_unix.go     # //go:build unix → FreeDiskSpace(), AvailableMemory()
├── resource_darwin.go   # //go:build darwin → getSystemMemory() for macOS
├── resource_linux.go    # //go:build linux → getSystemMemory() for Linux
└── resource_windows.go  # //go:build windows → stub methods returning errors
```

Key rules:

- Interface is defined **once** in the common file (never duplicated)
- Use `//go:build unix` for code shared by darwin/linux/bsd
- Use `//go:build darwin` or `//go:build linux` for OS-specific helpers
- Unsupported features return `errors.New("X not supported on Windows")`, not panic
- Always test builds: `GOOS=windows go build ./...`

## Check Packages Reference

| Package           | Interface         | Key Fields                                           | Purpose                    |
| ----------------- | ----------------- | ---------------------------------------------------- | -------------------------- |
| **cmdcheck**      | `CmdRunner`       | Name, MinVersion, MaxVersion, MatchPattern           | Binary existence & version |
| **envcheck**      | `EnvGetter`       | Name, Match, Exact, OneOf, HideValue, MaskValue      | Env var validation         |
| **filecheck**     | `FileSystem`      | Path, ExpectDir, ExpectSocket, Mode, Match, Contains | File/dir checks            |
| **gitcheck**      | `GitRunner`       | Clean, NoUncommitted, Branch, TagMatch               | Git repo state             |
| **hashcheck**     | `HashFileOpener`  | File, Algorithm, ExpectedHash, ChecksumFile          | Checksum verification      |
| **httpcheck**     | `HTTPClient`      | URL, ExpectedStatus, Timeout, Retry                  | HTTP health checks         |
| **jsoncheck**     | `FileSystem`      | File, HasKey, Key, Exact, Match                      | JSON validation            |
| **resourcecheck** | `ResourceChecker` | MinDisk, MinMemory, MinCPUs, Path                    | System resources           |
| **syscheck**      | `SysInfo`         | OSType, DistroMatch                                  | OS/distro detection        |
| **tcpcheck**      | `TCPDialer`       | Address, Timeout                                     | TCP connectivity           |
| **usercheck**     | `UserLookup`      | Username, UID, GID                                   | User existence             |

## Output & Exit Handling

```
Check.Run() → Result{Status, Details, Err}
     ↓
output.PrintResult()  → Formats with colors, prints to stdout
     ↓
runCheck()           → if !result.OK() { os.Exit(1) }
```

- Exit 0 = all checks passed
- Exit 1 = any check failed
- Colors: green for OK, red for FAIL (respects PREFLIGHT_COLOR env var)

## Testing Patterns

### Mock Pattern

```go
type mockRunner struct {
    RunFunc func(args ...string) (string, error)
}

func (m *mockRunner) Run(args ...string) (string, error) {
    return m.RunFunc(args...)
}
```

### Table-Driven Tests

```go
tests := []struct {
    name       string
    check      Check
    wantStatus check.Status
    wantDetail string
}{
    {
        name:       "descriptive test name",
        check:      Check{Field: "value", Runner: &mockRunner{...}},
        wantStatus: check.StatusOK,
    },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := tt.check.Run()
        if result.Status != tt.wantStatus {
            t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
        }
    })
}
```

### Integration Tests

`integration_test.go` tests each check with real system resources:

```go
func TestIntegration_Cmd(t *testing.T) {
    c := cmdcheck.Check{
        Name:   "bash",
        Runner: &cmdcheck.RealCmdRunner{},
    }
    result := c.Run()
    if result.Status != check.StatusOK {
        t.Errorf("Status = %v, want OK", result.Status)
    }
}
```

## Adding New Commands

1. Create `pkg/<name>check/check.go`:
   - Define `Check` struct with config fields
   - Define interface for external deps (e.g., `Runner`)
   - Implement `Run() check.Result`
   - Create `Real*` implementation

2. Create `pkg/<name>check/check_test.go`:
   - Define mock struct implementing interface
   - Write table-driven tests covering success/failure cases

3. Create `cmd/preflight/cmd_<name>.go`:
   - Declare flag variables
   - Create cobra.Command with Use, Short, Args, RunE
   - Register flags in init()
   - Implement run function that constructs Check and calls runCheck()

4. Add integration test to `integration_test.go`

5. Update `docs/usage.md` with flags table and examples

6. Update `README.md` if command is noteworthy

## Key Files Quick Reference

| File                          | Purpose                                                        |
| ----------------------------- | -------------------------------------------------------------- |
| `cmd/preflight/main.go`       | Entry point, hashbang detection for `#!/usr/bin/env preflight` |
| `cmd/preflight/root.go`       | Cobra root command, global flags                               |
| `cmd/preflight/run.go`        | `runCheck()` helper shared by all commands                     |
| `cmd/preflight/validation.go` | `requireExactlyOne()`, `requireAtLeastOne()`                   |
| `pkg/check/result.go`         | `Result` struct, `Status` type (OK/FAIL)                       |
| `pkg/check/helpers.go`        | `Fail()`, `AddDetail()`, `CompileRegex()`                      |
| `pkg/check/checker.go`        | `Checker` interface                                            |
| `pkg/output/output.go`        | `PrintResult()` with color support                             |
| `pkg/version/version.go`      | Semantic version parsing & comparison                          |
| `pkg/preflightfile/`          | `.preflight` file discovery & parsing                          |

## Commands Quick Reference

| Command    | Example                                       | Purpose                 |
| ---------- | --------------------------------------------- | ----------------------- |
| `cmd`      | `preflight cmd node --min 18.0`               | Check binary & version  |
| `env`      | `preflight env DB_URL --match '^postgres://'` | Validate env var        |
| `file`     | `preflight file /app/config --socket`         | File/dir properties     |
| `git`      | `preflight git --branch main --clean`         | Git repo state          |
| `hash`     | `preflight hash app.tar.gz --sha256 abc...`   | Verify checksums        |
| `http`     | `preflight http :8080/health --retry 3`       | HTTP health check       |
| `json`     | `preflight json config.json --has-key db`     | JSON validation         |
| `resource` | `preflight resource --min-disk 10G`           | System resources        |
| `sys`      | `preflight sys --os linux`                    | OS/distro check         |
| `tcp`      | `preflight tcp postgres:5432`                 | TCP connectivity        |
| `user`     | `preflight user postgres`                     | User existence          |
| `run`      | `preflight run`                               | Execute .preflight file |
