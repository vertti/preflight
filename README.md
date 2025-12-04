# Preflight

[![CI](https://github.com/vertti/preflight/actions/workflows/ci.yml/badge.svg)](https://github.com/vertti/preflight/actions/workflows/ci.yml)

> Sanity checks for containers. Tiny binary, zero dependencies.

Preflight validates your container environment at build time, runtime, or in CI. It replaces brittle shell scripts with clear, consistent checks.

**Use it for:**

- **Docker builds** – verify binaries built from source actually work
- **Multi-stage builds** – catch broken paths, missing libs, or copy mistakes
- **Container startup** – validate services are reachable before your app starts
- **CI pipelines** – verify container images have the right tools and config

## The Problem

Complex multi-stage Docker builds make it easy to break things silently. A typo in a COPY path, a missing shared library, or a misconfigured environment variable might not surface until production.

```sh
RUN command -v myapp || (echo "myapp missing"; exit 1)
RUN [ -n "$MODEL_PATH" ] || (echo "MODEL_PATH not set"; exit 1)
RUN [ -x /usr/local/bin/inference ] || (echo "inference not executable"; exit 1)
```

## The Solution

```dockerfile
COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight

RUN preflight cmd myapp
RUN preflight env MODEL_PATH --match '^/models/'
RUN preflight file /usr/local/bin/inference --executable
```

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

### Check commands

Like `which`, but verifies the binary actually runs (catches missing `.so` dependencies).

```sh
preflight cmd myapp                   # exists and runs
preflight cmd myapp --min 2.0         # minimum version
preflight cmd onnxruntime             # ML runtime with native deps
preflight cmd ffmpeg --version-cmd -version  # custom version flag
```

### Check environment variables

```sh
preflight env MODEL_PATH                           # exists and non-empty
preflight env MODEL_PATH --match '^/models/'       # matches pattern
preflight env APP_ENV --one-of dev,staging,prod    # allowed values
preflight env AWS_SECRET_ARN --mask-value          # hide in output
```

### Check files and directories

```sh
preflight file /models/bert.onnx --not-empty     # model file exists
preflight file /app/config --dir --writable      # directory is writable
preflight file /app/config.yaml --contains "api" # content check
preflight file /usr/local/bin/myapp --executable # binary is executable
```

### Run checks from a file

Create a `.preflight` file in your project to define all checks in one place:

```sh
# .preflight
file /etc/localtime --not-empty
cmd myapp --min 2.0
cmd go
preflight env HOME
```

Run all checks:

```sh
preflight run                             # finds .preflight automatically
preflight run --file /path/to/.preflight  # specify file explicitly
```

#### File discovery

`preflight run` searches up from the current directory until it finds a `.preflight` file, reaches `$HOME`, or encounters a `.git` directory (whichever comes first).

### Hashbang support

Make `.preflight` files executable and run them directly:

```sh
#!/usr/bin/env preflight

file /models/bert.onnx --not-empty
cmd myapp
preflight env PATH
```

```sh
chmod +x preflight
./preflight
```

**File format**:

- Lines starting with `#` are treated as comments
- Empty lines are ignored
- Lines without `preflight` prefix are automatically prepended with `preflight`
- Commands execute sequentially and can exit as needed

### Output

> ${\color{green}[OK]}$ ${\color{gray}cmd:}$ myapp
> ${\color{gray}path:}$ /usr/local/bin/myapp
> ${\color{gray}version:}$ 2.1.0
>
> ${\color{red}[FAIL]}$ ${\color{gray}env:}$ MODEL_PATH
> ${\color{gray}error:}$ not set

Exit code `0` on success, `1` on failure.

[Full usage guide](docs/usage.md)

## License

Apache 2.0
