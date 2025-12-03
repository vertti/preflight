# Installation

## Quick Install

```sh
curl -fsSL https://raw.githubusercontent.com/vertti/preflight/main/install.sh | sh
```

## In Dockerfiles

```dockerfile
COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight
```

## Manual Download

Download the binary for your platform from [GitHub Releases](https://github.com/vertti/preflight/releases):

```sh
# Linux (amd64)
curl -fsSL https://github.com/vertti/preflight/releases/latest/download/preflight-linux-amd64 \
  -o /usr/local/bin/preflight && chmod +x /usr/local/bin/preflight

# Linux (arm64)
curl -fsSL https://github.com/vertti/preflight/releases/latest/download/preflight-linux-arm64 \
  -o /usr/local/bin/preflight && chmod +x /usr/local/bin/preflight

# macOS (Intel)
curl -fsSL https://github.com/vertti/preflight/releases/latest/download/preflight-darwin-amd64 \
  -o /usr/local/bin/preflight && chmod +x /usr/local/bin/preflight

# macOS (Apple Silicon)
curl -fsSL https://github.com/vertti/preflight/releases/latest/download/preflight-darwin-arm64 \
  -o /usr/local/bin/preflight && chmod +x /usr/local/bin/preflight
```

## From Source

```sh
go install github.com/vertti/preflight/cmd/preflight@latest
```
