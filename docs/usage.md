# Usage Guide

## `preflight cmd`

Verifies a command exists on PATH and can execute. By default, runs `<command> --version` to ensure the binary actually works (catches missing shared libraries, corrupt binaries, etc.).

```sh
preflight cmd <command> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--min <version>` | Minimum version required (inclusive) |
| `--max <version>` | Maximum version allowed (exclusive) |
| `--exact <version>` | Exact version required |
| `--match <pattern>` | Regex pattern to match against version output |
| `--version-cmd <arg>` | Override the default `--version` argument |

### Examples

```sh
# Basic check - exists and runs
preflight cmd node

# Version constraints
preflight cmd node --min 18
preflight cmd node --min 18 --max 22
preflight cmd node --exact 18.17.0

# Regex match on version output
preflight cmd node --match "^v18\."

# Custom version command
preflight cmd go --version-cmd version      # runs: go version
preflight cmd java --version-cmd "-version" # runs: java -version
```

---

## `preflight env`

Validates environment variables exist and match requirements. By default, the variable must exist and be non-empty.

```sh
preflight env <variable> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--required` | Fail if not set (allows empty values) |
| `--match <pattern>` | Regex pattern to match against value |
| `--exact <value>` | Exact value required |
| `--hide-value` | Don't show value in output |
| `--mask-value` | Show first/last 3 chars only (e.g., `sk-•••xyz`) |

### Examples

```sh
# Basic check - exists and non-empty
preflight env DATABASE_URL

# Allow empty values (just check if defined)
preflight env OPTIONAL_VAR --required

# Pattern matching
preflight env DATABASE_URL --match '^postgres://'

# Exact value
preflight env NODE_ENV --exact production

# Hide sensitive values in logs
preflight env API_KEY --hide-value   # shows: [hidden]
preflight env API_KEY --mask-value   # shows: sk-•••xyz
```

---

## Output Format

### Success

```
[OK] cmd:node
      path: /usr/bin/node
      version: 18.17.0
```

### Failure

```
[FAIL] cmd:node
      not found in PATH

[FAIL] cmd:node
      version 16.0.0 < minimum 18.0.0

[FAIL] env:DATABASE_URL
      not set

[FAIL] env:DATABASE_URL
      value does not match pattern "^postgres://"
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All checks passed |
| `1` | One or more checks failed |
