# Usage Guide

Preflight provides consistent checks for validating your container environment. Use it at different stages:

- **Build time** (`RUN preflight ...`) – validate binaries, configs, and paths during image build
- **Runtime** (`CMD ["sh", "-c", "preflight ... && ./app"]`) – verify services are reachable before app startup
- **CI/CD** (`docker run myimage preflight ...`) – verify built images have correct tools and settings

This is especially useful for complex multi-stage builds where it's easy to make small mistakes (wrong COPY paths, missing shared libraries, misconfigured environment).

---

## `preflight cmd`

Verifies a command exists on PATH and can execute. By default, runs `<command> --version` to ensure the binary actually works (catches missing shared libraries, corrupt binaries, etc.).

```sh
preflight cmd <command> [flags]
```

### Flags

| Flag                  | Description                                   |
| --------------------- | --------------------------------------------- |
| `--min <version>`     | Minimum version required (inclusive)          |
| `--max <version>`     | Maximum version allowed (exclusive)           |
| `--exact <version>`   | Exact version required                        |
| `--match <pattern>`   | Regex pattern to match against version output |
| `--version-cmd <arg>` | Override the default `--version` argument     |

### Examples

```sh
# Basic check - exists and runs
preflight cmd myapp

# Version constraints
preflight cmd myapp --min 2.0
preflight cmd myapp --min 2.0 --max 3.0
preflight cmd onnxruntime --exact 1.16.0

# Regex match on version output
preflight cmd myapp --match "^v2\."

# Custom version command
preflight cmd ffmpeg --version-cmd -version  # runs: ffmpeg -version
preflight cmd java --version-cmd "-version"  # runs: java -version
```

---

## `preflight env`

Validates environment variables exist and match requirements. By default, the variable must exist and be non-empty.

```sh
preflight env <variable> [flags]
```

### Flags

| Flag                | Description                                      |
| ------------------- | ------------------------------------------------ |
| `--required`        | Fail if not set (allows empty values)            |
| `--match <pattern>` | Regex pattern to match against value             |
| `--exact <value>`   | Exact value required                             |
| `--one-of <values>` | Value must be one of these (comma-separated)     |
| `--hide-value`      | Don't show value in output                       |
| `--mask-value`      | Show first/last 3 chars only (e.g., `sk-•••xyz`) |

### Examples

```sh
# Basic check - exists and non-empty
preflight env MODEL_PATH

# Allow empty values (just check if defined)
preflight env OPTIONAL_CONFIG --required

# Pattern matching
preflight env MODEL_PATH --match '^/models/'
preflight env AWS_SECRET_ARN --match '^arn:aws:secretsmanager:'

# Exact value
preflight env APP_ENV --exact production

# Value from allowed list
preflight env APP_ENV --one-of dev,staging,production

# Hide sensitive values in logs
preflight env AWS_SECRET_ARN --hide-value   # shows: [hidden]
preflight env AWS_SECRET_ARN --mask-value   # shows: arn•••xyz
```

---

## `preflight file`

Checks that a file or directory exists and meets requirements. By default, the path must exist and be readable.

```sh
preflight file <path> [flags]
```

### Flags

| Flag                   | Description                                          |
| ---------------------- | ---------------------------------------------------- |
| `--dir`                | Expect a directory (fail if it's a file)             |
| `--writable`           | Check write permission                               |
| `--executable`         | Check execute permission                             |
| `--not-empty`          | File must have size > 0                              |
| `--min-size <bytes>`   | Minimum file size                                    |
| `--max-size <bytes>`   | Maximum file size                                    |
| `--match <pattern>`    | Regex pattern to match against content               |
| `--contains <string>`  | Literal string to search in content                  |
| `--head <bytes>`       | Limit content read to first N bytes                  |
| `--mode <perms>`       | Minimum permissions (e.g., `0644` passes for `0666`) |
| `--mode-exact <perms>` | Exact permissions required                           |

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

## `preflight tcp`

Checks TCP connectivity to a host:port. Useful for verifying that a database, cache, or other service is reachable before starting your application.

```sh
preflight tcp <host:port> [flags]
```

### Flags

| Flag              | Description                     |
| ----------------- | ------------------------------- |
| `--timeout <dur>` | Connection timeout (default 5s) |

### Examples

```sh
# Check database is reachable
preflight tcp localhost:5432

# Check Redis with custom timeout
preflight tcp redis:6379 --timeout 10s

# Check multiple services in container startup
preflight tcp postgres:5432 && preflight tcp redis:6379
```

### Runtime Use Cases

TCP checks are most useful for **runtime validation** - ensuring services are reachable before your application starts.

**Container startup scripts:**

```dockerfile
CMD ["sh", "-c", "preflight tcp postgres:5432 && ./myapp"]
```

**Kubernetes readiness probes:**

```yaml
readinessProbe:
  exec:
    command: ["preflight", "tcp", "redis:6379"]
```

**CI with service containers (GitHub Actions):**

```yaml
services:
  postgres:
    image: postgres:15
steps:
  - run: preflight tcp localhost:5432 --timeout 30s
```

**Docker Compose health checks:**

```yaml
healthcheck:
  test: ["CMD", "preflight", "tcp", "db:5432"]
```

---

## CI & Container Verification

Preflight can verify container images in CI pipelines, replacing ad-hoc shell scripts. These examples assume preflight is installed in the container image.

### Verifying a built container image

```sh
# Instead of:
TAG=$(docker run "myapp:latest" sh -c "echo \$APP_VERSION")
if [ "$TAG" != "$EXPECTED" ]; then exit 1; fi

# Use:
docker run myapp:latest preflight env APP_VERSION --exact "$EXPECTED"
```

### Running multiple checks against a container

```sh
docker run myapp:latest sh -c '
  preflight env APP_VERSION --exact "$EXPECTED_VERSION" &&
  preflight cmd myapp &&
  preflight file /models/model.onnx --not-empty
'
```

### In GitHub Actions

```yaml
- name: Verify container
  run: |
    docker run myapp:${{ github.sha }} preflight env APP_VERSION --exact "${{ github.sha }}"
```

---

## Output Format

### Success

```
[OK] cmd:myapp
      path: /usr/local/bin/myapp
      version: 2.1.0
```

### Failure

```
[FAIL] cmd:myapp
      not found in PATH

[FAIL] cmd:myapp
      version 1.5.0 < minimum 2.0.0

[FAIL] env:MODEL_PATH
      not set

[FAIL] env:MODEL_PATH
      value does not match pattern "^/models/"

[FAIL] file:/var/log/app
      not writable

[FAIL] file:/missing/path
      not found

[OK] tcp:localhost:5432
      connected to localhost:5432

[FAIL] tcp:localhost:9999
      connection failed: dial tcp [::1]:9999: connect: connection refused
```

---

## Exit Codes

| Code | Meaning                   |
| ---- | ------------------------- |
| `0`  | All checks passed         |
| `1`  | One or more checks failed |
