# Usage Guide

Preflight provides consistent checks for validating your container environment. Use it at different stages:

- **Build time** (`RUN preflight ...`) – validate binaries, configs, and paths during image build
- **Runtime** (`CMD ["sh", "-c", "preflight ... && ./app"]`) – verify services are reachable before app startup
- **CI/CD** (`docker run myimage preflight ...`) – verify built images have correct tools and settings

This is especially useful for complex multi-stage builds where it's easy to make small mistakes (wrong COPY paths, missing shared libraries, misconfigured environment).

## Table of Contents

**Commands**

- [`preflight cmd`](#preflight-cmd) – verify binary exists and version
- [`preflight env`](#preflight-env) – validate environment variables
- [`preflight file`](#preflight-file) – check file/directory properties
- [`preflight json`](#preflight-json) – validate JSON and check keys
- [`preflight git`](#preflight-git) – verify git repository state
- [`preflight tcp`](#preflight-tcp) – check TCP connectivity
- [`preflight http`](#preflight-http) – HTTP health checks
- [`preflight hash`](#preflight-hash) – verify file checksums
- [`preflight sys`](#preflight-sys) – check OS and architecture
- [`preflight resource`](#preflight-resource) – verify system resources
- [`preflight user`](#preflight-user) – check user exists
- [`preflight run`](#preflight-run) – run checks from file

**Reference**

- [CI & Container Verification](#ci--container-verification)
- [Output Format](#output-format)
- [Exit Codes](#exit-codes)
- [Colored Output](#colored-output)

---

## `preflight cmd`

Verifies a command exists on PATH and can execute. By default, runs `<command> --version` to ensure the binary actually works (catches missing shared libraries, corrupt binaries, etc.).

```sh
preflight cmd <command> [flags]
```

### Flags

| Flag                    | Description                                   |
| ----------------------- | --------------------------------------------- |
| `--min <version>`       | Minimum version required (inclusive)          |
| `--max <version>`       | Maximum version allowed (exclusive)           |
| `--exact <version>`     | Exact version required                        |
| `--match <pattern>`     | Regex pattern to match against version output |
| `--version-regex <pat>` | Regex with capture group to extract version   |
| `--version-cmd <arg>`   | Override the default `--version` argument     |
| `--timeout <duration>`  | Timeout for version command (default: 30s)    |

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

# Custom timeout (for slow commands)
preflight cmd slow-binary --timeout 60s
preflight cmd quick-check --timeout 5s

# Extract version from messy output using regex capture group
# For output like "myapp version: 2.5.3 (built 2024-01-01)"
preflight cmd myapp --version-regex "version[:\s]+(\d+\.\d+\.\d+)" --min 2.0
```

---

## `preflight env`

Validates environment variables exist and match requirements. By default, the variable must exist and be non-empty.

```sh
preflight env <variable> [flags]
```

### Flags

| Flag                  | Description                                      |
| --------------------- | ------------------------------------------------ |
| `--required`          | Fail if not set (allows empty values)            |
| `--match <pattern>`   | Regex pattern to match against value             |
| `--exact <value>`     | Exact value required                             |
| `--one-of <values>`   | Value must be one of these (comma-separated)     |
| `--starts-with <str>` | Value must start with string                     |
| `--ends-with <str>`   | Value must end with string                       |
| `--contains <str>`    | Value must contain substring                     |
| `--is-numeric`        | Value must be a valid number                     |
| `--min-len <n>`       | Minimum string length                            |
| `--max-len <n>`       | Maximum string length                            |
| `--hide-value`        | Don't show value in output                       |
| `--mask-value`        | Show first/last 3 chars only (e.g., `sk-•••xyz`) |

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

# String matchers
preflight env MODEL_PATH --starts-with /models/
preflight env CONFIG_FILE --ends-with .yaml
preflight env WEBHOOK_URL --contains example.com

# Numeric and length validation
preflight env PORT --is-numeric
preflight env API_KEY --min-len 32
preflight env CODE --max-len 6

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

| Flag                      | Description                                          |
| ------------------------- | ---------------------------------------------------- |
| `--dir`                   | Expect a directory (fail if it's a file)             |
| `--socket`                | Expect a Unix socket (e.g., docker.sock)             |
| `--symlink`               | Expect a symbolic link                               |
| `--symlink-target <path>` | Expected symlink target path                         |
| `--writable`              | Check write permission                               |
| `--executable`            | Check execute permission                             |
| `--not-empty`             | File must have size > 0                              |
| `--min-size <bytes>`      | Minimum file size                                    |
| `--max-size <bytes>`      | Maximum file size                                    |
| `--match <pattern>`       | Regex pattern to match against content               |
| `--contains <string>`     | Literal string to search in content                  |
| `--head <bytes>`          | Limit content read to first N bytes                  |
| `--mode <perms>`          | Minimum permissions (e.g., `0644` passes for `0666`) |
| `--mode-exact <perms>`    | Exact permissions required                           |
| `--owner <uid>`           | Expected owner UID                                   |

### Examples

```sh
# Basic check - exists and readable
preflight file /etc/nginx/nginx.conf

# Directory checks
preflight file /var/log/app --dir --writable

# Unix socket checks (Docker-in-Docker, containerd)
preflight file /var/run/docker.sock --socket
preflight file /run/containerd/containerd.sock --socket

# Permission checks (minimum - file has at least these perms)
preflight file /etc/ssl/private/key.pem --mode 0600

# Permission checks (exact)
preflight file /etc/ssl/private/key.pem --mode-exact 0600

# Ownership checks (HashiCorp Consul/Vault pattern)
preflight file /data --owner 1000
preflight file /app/data --dir --owner 1000

# Symlink checks (capability management, alternative binaries)
preflight file /usr/bin/python --symlink
preflight file /usr/bin/python --symlink --symlink-target /usr/bin/python3

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

## `preflight json`

Validates JSON files and checks key/value assertions. Useful for verifying configuration files are valid and contain required settings.

```sh
preflight json <file> [flags]
```

### Flags

| Flag                | Description                                          |
| ------------------- | ---------------------------------------------------- |
| `--has-key <path>`  | Check key exists (dot notation for nested keys)      |
| `--key <path>`      | Key to check value of (dot notation for nested keys) |
| `--exact <value>`   | Exact value required (requires `--key`)              |
| `--match <pattern>` | Regex pattern for value (requires `--key`)           |

### Examples

```sh
# Validate JSON syntax only
preflight json config.json

# Check key exists
preflight json config.json --has-key database.host

# Check nested key exists
preflight json package.json --has-key dependencies.express

# Check exact value
preflight json config.json --key environment --exact production

# Check value matches pattern
preflight json package.json --key version --match "^1\."

# Combined: validate and check required key
preflight json config.json --has-key database.host
```

### Dot Notation

Use dot notation to access nested keys:

```json
{
  "database": {
    "host": "localhost",
    "port": 5432
  }
}
```

```sh
preflight json config.json --has-key database.host
preflight json config.json --key database.port --exact 5432
```

### Non-String Values

Non-string values are converted to strings for comparison:

```sh
# Numbers
preflight json config.json --key port --exact 8080

# Booleans
preflight json config.json --key enabled --exact true

# Null
preflight json config.json --key optional --exact null
```

### Use Cases

**CI - verify package.json:**

```sh
preflight json package.json --key version --match "^[0-9]+\.[0-9]+\.[0-9]+$"
```

**Docker - validate config before startup:**

```dockerfile
COPY config.json /app/
RUN preflight json /app/config.json --has-key database.host
```

**Kubernetes - verify ConfigMap mounted correctly:**

```yaml
readinessProbe:
  exec:
    command: ["preflight", "json", "/etc/config/app.json", "--has-key", "api.endpoint"]
```

### What This Is Not

`preflight json` is intentionally limited. For complex JSON queries, use `jq`:

- **Not supported:** Array indexing (`items[0]`)
- **Not supported:** JSONPath queries
- **Not supported:** Data extraction/transformation

---

## `preflight git`

Verifies git repository state. Useful for CI pipelines that need to ensure a clean working directory or verify branch/tag state before builds or releases.

```sh
preflight git [flags]
```

### Flags

| Flag               | Description                                         |
| ------------------ | --------------------------------------------------- |
| `--clean`          | Working directory must be clean (no changes at all) |
| `--no-uncommitted` | No staged or modified files allowed                 |
| `--no-untracked`   | No untracked files allowed                          |
| `--branch <name>`  | Must be on specified branch                         |
| `--tag-match <p>`  | HEAD must have tag matching glob pattern            |

At least one flag is required.

### Examples

```sh
# Verify clean state after code generation
go generate ./...
preflight git --clean

# Check formatting didn't change anything
go fmt ./...
preflight git --no-uncommitted

# Allow untracked files, but no uncommitted changes
preflight git --no-uncommitted

# Must be on specific branch
preflight git --branch main

# HEAD must have a version tag
preflight git --tag-match "v*"

# Combined checks for release
preflight git --clean --branch release --tag-match "v*"
```

### Use Cases

**CI vendor verification:**

```sh
# Verify go mod tidy didn't change anything
go mod tidy
preflight git --clean

# Verify formatting is consistent
go fmt ./...
preflight git --no-uncommitted
```

**Release verification:**

```sh
# Ensure releasing from correct branch with proper tag
preflight git --branch main --tag-match "v*"
```

**GitHub Actions:**

```yaml
- name: Verify clean state
  run: |
    go generate ./...
    preflight git --clean
```

### Tools Replaced

`preflight git` replaces common shell patterns for git state verification:

**Before (vendor verification):**

```bash
hugo mod vendor
if [ -n "$(git status --porcelain)" ]; then
    echo 'ERROR: Vendor result differs'
    exit 1
fi
```

**After:**

```sh
hugo mod vendor
preflight git --clean
```

**Before (formatting check):**

```bash
go fmt ./... && git diff --exit-code
```

**After:**

```sh
go fmt ./...
preflight git --no-uncommitted
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

### Tools Replaced

`preflight tcp` replaces several widely-used service waiting tools:

| Tool                                                       | Stars  | What preflight replaces           |
| ---------------------------------------------------------- | ------ | --------------------------------- |
| [wait-for-it.sh](https://github.com/vishnubob/wait-for-it) | 9,700+ | `wait-for-it.sh host:port`        |
| [dockerize](https://github.com/jwilder/dockerize)          | 4,800+ | `dockerize -wait tcp://host:port` |
| netcat                                                     | -      | `nc -z host port`                 |
| bash built-in                                              | -      | `echo > /dev/tcp/$HOST/$PORT`     |

**Before (wait-for-it.sh):**

```dockerfile
COPY wait-for-it.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/wait-for-it.sh
CMD ["wait-for-it.sh", "db:5432", "--", "./app"]
```

**After (preflight):**

```dockerfile
CMD ["sh", "-c", "preflight tcp db:5432 && ./app"]
```

---

## `preflight http`

HTTP health checks for verifying services are up and responding correctly. Replaces `curl --fail` or `wget --spider` in containers, eliminating the need to install those tools.

```sh
preflight http <url> [flags]
```

### Flags

| Flag                   | Description                         |
| ---------------------- | ----------------------------------- |
| `--status <code>`      | Expected HTTP status (default: 200) |
| `--timeout <duration>` | Request timeout (default: 5s)       |
| `--retry <n>`          | Retry count on failure              |
| `--retry-delay <dur>`  | Delay between retries (default: 1s) |
| `--method <method>`    | HTTP method: GET or HEAD            |
| `--header <key:value>` | Custom header (can be repeated)     |
| `--insecure`           | Skip TLS certificate verification   |

### Examples

```sh
# Basic health check
preflight http http://localhost:8080/health

# Custom expected status
preflight http http://localhost/api --status 204

# With timeout
preflight http http://slow-service:8080/ready --timeout 30s

# Retry on failure (3 retries = 4 total attempts)
preflight http http://localhost:8080/health --retry 3 --retry-delay 2s

# HEAD request (lighter weight)
preflight http http://localhost:8080/health --method HEAD

# Custom headers
preflight http http://localhost/api --header "Authorization:Bearer token123"
preflight http http://localhost/api --header "X-API-Key:secret" --header "Accept:application/json"

# Skip TLS verification (self-signed certs)
preflight http https://internal-service/health --insecure
```

### Redirect Handling

Redirects are **not followed** automatically. If the server returns a 3xx status, that status is checked against `--status`. This matches `curl --fail` behavior.

```sh
# This fails if server returns 302
preflight http http://localhost/old-path

# This passes if server returns 302
preflight http http://localhost/old-path --status 302
```

### Runtime Use Cases

HTTP checks are useful for **runtime validation** - ensuring services are healthy before your application starts.

**Container startup (HEALTHCHECK):**

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD preflight http http://localhost:8080/health
```

**Container startup script:**

```dockerfile
CMD ["sh", "-c", "preflight http http://api:8080/ready --retry 5 && ./myapp"]
```

**Kubernetes liveness/readiness probes:**

```yaml
livenessProbe:
  exec:
    command: ["preflight", "http", "http://localhost:8080/health"]
readinessProbe:
  exec:
    command: ["preflight", "http", "http://localhost:8080/ready", "--timeout", "10s"]
```

**Docker Compose health checks:**

```yaml
healthcheck:
  test: ["CMD", "preflight", "http", "http://localhost:8080/health"]
  interval: 30s
  timeout: 5s
  retries: 3
```

### Why Not curl?

`preflight http` provides several advantages over `curl --fail`:

1. **No extra dependencies** - curl is 3-8MB and requires TLS libraries
2. **Consistent exit codes** - curl's exit codes vary by error type
3. **Built-in retry** - no need for shell loops
4. **Same syntax** - consistent with other preflight commands

---

## `preflight hash`

Verifies file checksums for supply chain security. Replaces `sha256sum -c` and similar patterns in Dockerfiles.

```sh
preflight hash <file> [flags]
```

### Flags

| Flag                     | Description                                      |
| ------------------------ | ------------------------------------------------ |
| `--sha256 <hash>`        | Expected SHA256 hash (64 hex characters)         |
| `--sha512 <hash>`        | Expected SHA512 hash (128 hex characters)        |
| `--md5 <hash>`           | Expected MD5 hash (32 hex characters)            |
| `--checksum-file <path>` | Verify against checksum file (GNU or BSD format) |

### Examples

```sh
# Verify SHA256 hash
preflight hash --sha256 67574ee...2cf downloaded.tar.gz

# Verify SHA512 hash
preflight hash --sha512 abc123... binary.tar.gz

# Verify MD5 (legacy, not recommended for security)
preflight hash --md5 deadbeef... legacy.zip

# Verify against checksum file (GNU format: hash  filename)
preflight hash --checksum-file SHASUMS256.txt node-v20.tar.gz

# Verify against checksum file (BSD format: SHA256 (filename) = hash)
preflight hash --checksum-file checksums.txt myapp.tar.gz
```

### Checksum File Formats

Preflight supports two common checksum file formats:

**GNU format** (used by `sha256sum`, Node.js SHASUMS):

```
67574ee2f0d8e... myfile.tar.gz
abc123def456... otherfile.tar.gz
```

**BSD format** (used by macOS `shasum`):

```
SHA256 (myfile.tar.gz) = 67574ee2f0d8e...
SHA512 (otherfile.tar.gz) = abc123def456...
```

The algorithm is auto-detected from BSD format, or inferred from hash length in GNU format.

### Use Cases

**Dockerfile - verify downloaded binary:**

```dockerfile
RUN curl -fsSL https://example.com/app.tar.gz -o /tmp/app.tar.gz
RUN preflight hash --sha256 $EXPECTED_HASH /tmp/app.tar.gz
RUN tar -xzf /tmp/app.tar.gz -C /usr/local/bin
```

**Multi-stage build - verify with checksum file:**

```dockerfile
RUN curl -fsSL https://nodejs.org/dist/v20.0.0/SHASUMS256.txt -o /tmp/SHASUMS256.txt
RUN curl -fsSL https://nodejs.org/dist/v20.0.0/node-v20.0.0.tar.gz -o /tmp/node.tar.gz
RUN preflight hash --checksum-file /tmp/SHASUMS256.txt /tmp/node.tar.gz
```

---

## `preflight sys`

Checks the system's OS and architecture. Useful for multi-architecture container builds to verify the correct platform.

```sh
preflight sys [flags]
```

### Flags

| Flag            | Description                          |
| --------------- | ------------------------------------ |
| `--os <os>`     | Required OS (linux, darwin, windows) |
| `--arch <arch>` | Required architecture (amd64, arm64) |

At least one of `--os` or `--arch` is required.

### Examples

```sh
# Check OS only
preflight sys --os linux

# Check architecture only
preflight sys --arch amd64

# Check both
preflight sys --os linux --arch arm64
```

### Tools Replaced

`preflight sys` replaces common architecture detection patterns critical for **multi-platform builds**:

| Pattern                           | Use Case                     |
| --------------------------------- | ---------------------------- |
| `uname -m \| sed s/x86_64/amd64/` | Normalize arch names         |
| `dpkg --print-architecture`       | Debian-based arch detection  |
| `TARGETARCH` (BuildKit ARG)       | Multi-platform Docker builds |

**Before (binary download scripts):**

```bash
# Common pattern in official images downloading binaries
arch="$(dpkg --print-architecture)"; case "$arch" in
    amd64) GOSU_ARCH='amd64';;
    arm64) GOSU_ARCH='arm64';;
    *) echo "unsupported: $arch"; exit 1;;
esac
curl -L "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$GOSU_ARCH" -o /usr/local/bin/gosu
```

**After (preflight):**

```dockerfile
# Verify expected platform before downloading
RUN preflight sys --arch amd64
RUN curl -L "https://example.com/binary-amd64" -o /usr/local/bin/binary
```

**Runtime verification:**

```bash
# Before - brittle
arch=$(uname -m | sed s/aarch64/arm64/ | sed s/x86_64/amd64/)
if [ "$arch" != "amd64" ]; then
    echo "Unsupported architecture: $arch"
    exit 1
fi

# After - clear intent
preflight sys --arch amd64
```

### Use Cases

**Multi-arch Dockerfile:**

```dockerfile
# Verify we're building for the expected platform
RUN preflight sys --os linux --arch arm64
```

**CI platform verification:**

```yaml
# GitHub Actions
- name: Verify platform
  run: preflight sys --arch amd64
```

---

## `preflight resource`

Checks system resources meet minimum requirements. Critical for CI pipelines where runners have limited disk space, or containers with memory limits.

```sh
preflight resource [flags]
```

### Flags

| Flag                  | Description                                      |
| --------------------- | ------------------------------------------------ |
| `--min-disk <size>`   | Minimum free disk space (e.g., 10G, 500M)        |
| `--min-memory <size>` | Minimum available memory (e.g., 2G, 512M)        |
| `--min-cpus <n>`      | Minimum number of CPU cores                      |
| `--path <path>`       | Path for disk space check (default: current dir) |

At least one of `--min-disk`, `--min-memory`, or `--min-cpus` is required.

### Size Format

Sizes support common units (case-insensitive):

| Unit      | Example | Bytes             |
| --------- | ------- | ----------------- |
| B (bytes) | `1024`  | 1,024             |
| K/KB      | `500K`  | 512,000           |
| M/MB      | `500M`  | 524,288,000       |
| G/GB      | `10G`   | 10,737,418,240    |
| T/TB      | `1T`    | 1,099,511,627,776 |

Decimals are supported: `1.5G`, `2.5TB`

### Examples

```sh
# Check disk space
preflight resource --min-disk 10G

# Check disk space at specific path
preflight resource --min-disk 10G --path /var/lib/docker

# Check available memory
preflight resource --min-memory 2G

# Check CPU cores
preflight resource --min-cpus 4

# Combined checks
preflight resource --min-disk 10G --min-memory 2G --min-cpus 2
```

### Use Cases

**GitHub Actions - prevent disk space failures:**

```yaml
- name: Check disk space before Docker build
  run: preflight resource --min-disk 10G
```

**Docker build - verify container resources:**

```dockerfile
# Verify container has enough resources before heavy operations
RUN preflight resource --min-memory 1G --min-cpus 2
```

**CI pipeline - verify runner capacity:**

```sh
# Before starting parallel tests
preflight resource --min-cpus 4 --min-memory 4G
```

### Tools Replaced

`preflight resource` replaces common shell patterns for checking system resources:

**Before (disk space check):**

```bash
available=$(df -BG /var/lib/docker | tail -1 | awk '{print $4}' | tr -d 'G')
if [ "$available" -lt 10 ]; then
    echo "Not enough disk space"
    exit 1
fi
```

**After:**

```sh
preflight resource --min-disk 10G --path /var/lib/docker
```

**Before (memory check):**

```bash
mem_kb=$(grep MemAvailable /proc/meminfo | awk '{print $2}')
mem_gb=$((mem_kb / 1024 / 1024))
if [ "$mem_gb" -lt 2 ]; then
    echo "Not enough memory"
    exit 1
fi
```

**After:**

```sh
preflight resource --min-memory 2G
```

---

## `preflight user`

Checks that a user exists on the system and optionally validates uid, gid, and home directory. Useful for verifying non-root container configurations.

```sh
preflight user <username> [flags]
```

### Flags

| Flag            | Description               |
| --------------- | ------------------------- |
| `--uid <id>`    | Expected user ID          |
| `--gid <id>`    | Expected primary group ID |
| `--home <path>` | Expected home directory   |

### Examples

```sh
# Check user exists
preflight user appuser

# Verify specific uid/gid (common for non-root containers)
preflight user appuser --uid 1000 --gid 1000

# Verify home directory
preflight user appuser --home /app

# All constraints
preflight user appuser --uid 1000 --gid 1000 --home /app
```

### Use Cases

**Non-root container validation:**

```dockerfile
RUN preflight user appuser --uid 1000 --gid 1000
USER appuser
```

**Kubernetes security context verification:**

```sh
# Verify container runs as expected user
preflight user nobody --uid 65534
```

### Tools Replaced

`preflight user` replaces common shell patterns found in **virtually every official Docker image** for privilege management:

**Before (PostgreSQL, MySQL, MongoDB official images):**

```bash
if [ "$1" = 'postgres' ] && [ "$(id -u)" = '0' ]; then
    mkdir -p "$PGDATA"
    chown -R postgres "$PGDATA"
    chmod 700 "$PGDATA"
    exec gosu postgres "$BASH_SOURCE" "$@"
fi
```

**After (preflight):**

```dockerfile
# Verify user configuration at build time
RUN preflight user postgres --uid 999 --gid 999
```

**Volume mount validation:**

```bash
# Before - common Docker run pattern
docker run -u $(id -u):$(id -g) -v $(pwd):/workspace myimage

# After - verify inside container
preflight user appuser --uid 1000 --gid 1000
```

---

## `preflight run`

Run multiple checks from a `.preflight` file. This is useful for defining all your checks in one place and running them together.

```sh
preflight run [flags]
```

### Flags

| Flag            | Description                                     |
| --------------- | ----------------------------------------------- |
| `--file <path>` | Path to preflight file (default: auto-discover) |

### File Format

```sh
# .preflight
# Lines starting with # are comments
# Empty lines are ignored

# Commands can omit the "preflight" prefix
file /etc/localtime --not-empty
cmd myapp --min 2.0
env HOME

# Or include it explicitly
preflight tcp localhost:5432
```

- Lines starting with `#` are treated as comments
- Empty lines are ignored
- Lines without `preflight` prefix are automatically prepended with `preflight`
- Commands execute sequentially

### File Discovery

When run without `--file`, `preflight run` searches for a `.preflight` file:

1. Start from the current directory
2. Search upward through parent directories
3. Stop when finding `.preflight`, reaching `$HOME`, or encountering a `.git` directory

This allows you to run `preflight run` from any subdirectory in your project.

### Hashbang Support

Make `.preflight` files executable and run them directly:

```sh
#!/usr/bin/env preflight

file /models/bert.onnx --not-empty
cmd myapp
env PATH
```

```sh
chmod +x .preflight
./.preflight
```

### Examples

```sh
# Auto-discover .preflight file
preflight run

# Specify file explicitly
preflight run --file /path/to/.preflight

# In Dockerfile
COPY .preflight .
RUN preflight run
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

[OK] http: http://localhost:8080/health
      status 200

[FAIL] http: http://localhost:8080/health
       status 503, expected 200

[FAIL] http: http://localhost:9999/health
       request failed: dial tcp [::1]:9999: connect: connection refused

[OK] user:appuser
      uid: 1000
      gid: 1000
      home: /app

[FAIL] user:nonexistent
      user not found: user: unknown user nonexistent
```

---

## Exit Codes

| Code | Meaning                   |
| ---- | ------------------------- |
| `0`  | All checks passed         |
| `1`  | One or more checks failed |

---

## Colored Output

Preflight outputs colored status indicators:

- `[OK]` in green
- `[FAIL]` in red

### Where Colors Work Automatically

- **Terminals** - colors are detected automatically
- **GitHub Actions** - colors are enabled automatically
- **GitLab CI** - colors are enabled automatically
- **Other CI systems** - most are detected automatically

### Docker Builds

Docker builds don't allocate a TTY, so colors are disabled by default. To enable colors in your Dockerfile, set `PREFLIGHT_COLOR=1`:

```dockerfile
FROM alpine

ENV PREFLIGHT_COLOR=1

RUN preflight cmd myapp
RUN preflight env MODEL_PATH
RUN preflight file /app/config.yaml
```

### Environment Variables

| Variable            | Description                       |
| ------------------- | --------------------------------- |
| `PREFLIGHT_COLOR=1` | Enable colors (for Docker builds) |
| `NO_COLOR=1`        | Disable colors                    |

### Examples

```sh
# Disable colors
NO_COLOR=1 preflight cmd myapp

# Enable colors in scripts/Docker
PREFLIGHT_COLOR=1 preflight cmd myapp
```
