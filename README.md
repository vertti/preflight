# Preflight

[![CI](https://github.com/vertti/preflight/actions/workflows/ci.yml/badge.svg)](https://github.com/vertti/preflight/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/vertti/preflight/graph/badge.svg)](https://codecov.io/gh/vertti/preflight)

> A swiss army knife for CI and container checks. Single binary, zero dependencies.

Stop copying brittle shell scripts and installing a variety of tools for container validation. Preflight is a small, dependency-free binary that handles service readiness, health checks, environment validation, file verification, and more.

```dockerfile
COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight

RUN preflight cmd myapp --min 2.0
RUN preflight env MODEL_PATH --match '^/models/'
RUN preflight file /app/config.yaml --not-empty
RUN preflight tcp postgres:5432
```

**Use it for:**

- **Docker builds** – verify binaries, configs, and paths during image build
- **Container startup** – wait for databases and services before your app starts
- **CI pipelines** – validate environments, check connectivity, verify checksums
- **Health checks** – HTTP and TCP checks without curl or netcat

**Clear output, CI-friendly exit codes:**

Every check tells you exactly what passed or failed—no more guessing why your build broke.

```
[OK] cmd: node
     path: /usr/local/bin/node
     version: 20.10.0

[OK] file: /app/config.yaml
     size: 1.2KB
     mode: -rw-r--r--

[FAIL] cmd: python
       version 3.9.0 < minimum 3.11

[FAIL] tcp: postgres:5432
       connection refused
```

Exit code `0` on success, `1` on failure. Works with `set -e`, Docker `RUN`, and CI pipelines out of the box.

## Install

**Dockerfiles** (recommended):

```dockerfile
COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight
```

**Shell**:

```sh
curl -fsSL https://raw.githubusercontent.com/vertti/preflight/main/install.sh | sh
```

[Other install methods](docs/install.md)

## Usage

See the **[full usage guide](docs/usage.md)** for all commands and options.

### Check commands

Like `which`, but actually runs the binary to verify it works.

```sh
preflight cmd node                            # exists and runs
preflight cmd node --min 18.0                 # version constraint
preflight cmd ffmpeg --version-cmd -version   # custom version flag
```

[All cmd options](docs/usage.md#preflight-cmd)

### Check environment variables

```sh
preflight env DATABASE_URL                       # exists and non-empty
preflight env MODEL_PATH --match '^/models/'     # matches pattern
preflight env APP_ENV --one-of dev,staging,prod  # allowed values
```

[All env options](docs/usage.md#preflight-env)

### Check files and directories

```sh
preflight file /models/bert.onnx --not-empty   # file exists and has content
preflight file /var/log/app --dir --writable   # directory is writable
preflight file /app/entrypoint.sh --executable # script is executable
```

[All file options](docs/usage.md#preflight-file)

### Check HTTP endpoints

Health checks for services without requiring curl or wget in your container.

```sh
preflight http http://localhost:8080/health         # basic health check
preflight http https://api.example.com --status 204 # custom status code
preflight http http://localhost/ready --retry 3     # retry on failure
```

[All http options](docs/usage.md#preflight-http)

### Verify file checksums

Supply chain security - verify downloaded binaries match expected hashes.

```sh
preflight hash --sha256 67574ee...2cf myfile.tar.gz  # verify SHA256
preflight hash --checksum-file SHASUMS256.txt app.tar.gz  # from checksum file
```

[All hash options](docs/usage.md#preflight-hash)

### Run checks from a file

Create a `.preflight` file in your project to define all checks in one place:

```sh
# .preflight
file /etc/localtime --not-empty
cmd myapp --min 2.0
cmd go
env HOME
```

Run all checks:

```sh
preflight run                             # finds .preflight automatically
preflight run --file /path/to/.preflight  # specify file explicitly
```

[File format, discovery, and hashbang support](docs/usage.md#preflight-run)

[Full usage guide](docs/usage.md)

## License

Apache 2.0
