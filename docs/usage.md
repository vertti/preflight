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
| `--one-of <values>` | Value must be one of these (comma-separated) |
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

# Value from allowed list
preflight env NODE_ENV --one-of dev,staging,production

# Hide sensitive values in logs
preflight env API_KEY --hide-value   # shows: [hidden]
preflight env API_KEY --mask-value   # shows: sk-•••xyz
```

---

## `preflight file`

Checks that a file or directory exists and meets requirements. By default, the path must exist and be readable.

```sh
preflight file <path> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--dir` | Expect a directory (fail if it's a file) |
| `--writable` | Check write permission |
| `--executable` | Check execute permission |
| `--not-empty` | File must have size > 0 |
| `--min-size <bytes>` | Minimum file size |
| `--max-size <bytes>` | Maximum file size |
| `--match <pattern>` | Regex pattern to match against content |
| `--contains <string>` | Literal string to search in content |
| `--head <bytes>` | Limit content read to first N bytes |
| `--mode <perms>` | Minimum permissions (e.g., `0644` passes for `0666`) |
| `--mode-exact <perms>` | Exact permissions required |

### Examples

```sh
# Basic check - exists and readable
preflight file /etc/nginx/nginx.conf

# Directory checks
preflight file /var/log/app --dir --writable

# Permission checks (minimum - file has at least these perms)
preflight file /etc/ssl/private/key.pem --mode 0600

# Permission checks (exact)
preflight file /etc/ssl/private/key.pem --mode-exact 0600

# Content checks (reads full file)
preflight file /etc/nginx/nginx.conf --contains "worker_processes"
preflight file /etc/hosts --match "127\.0\.0\.1"

# Content checks (limited to first 1KB)
preflight file /var/log/huge.log --contains "ERROR" --head 1024

# Size constraints
preflight file /app/data.json --not-empty
preflight file /var/log/app.log --max-size 10485760  # 10MB
```

---

## CI & Container Verification

Preflight can verify container images in CI pipelines, replacing ad-hoc shell scripts. These examples assume preflight is installed in the container image.

### Verifying a built container image

```sh
# Instead of:
TAG=$(docker run "myapp:latest" -c "echo \$VERSION_TAG")
if [ "$TAG" != "$EXPECTED" ]; then exit 1; fi

# Use:
docker run myapp:latest preflight env VERSION_TAG --exact "$EXPECTED"
```

### Running multiple checks against a container

```sh
docker run myapp:latest sh -c '
  preflight env VERSION_TAG --exact "$EXPECTED_VERSION" &&
  preflight cmd node --min 18 &&
  preflight file /app/config.json --not-empty
'
```

### In GitHub Actions

```yaml
- name: Verify container
  run: |
    docker run myapp:${{ github.sha }} preflight env VERSION_TAG --exact "${{ github.sha }}"
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

[FAIL] file:/var/log/app
      not writable

[FAIL] file:/missing/path
      not found
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All checks passed |
| `1` | One or more checks failed |
