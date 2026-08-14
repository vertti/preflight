# Installation

## Quick Install

```sh
curl -fsSL https://raw.githubusercontent.com/vertti/preflight/main/install.sh | sh
```

The installer verifies the downloaded binary against the release's
`checksums.txt` and aborts if it cannot. On an image carrying neither
`sha256sum` nor `shasum` it stops rather than installing something unverified;
`PREFLIGHT_SKIP_CHECKSUM=1` opts out deliberately.

## In Dockerfiles

```dockerfile
COPY --from=ghcr.io/vertti/preflight:latest /preflight /usr/local/bin/preflight
```

Copying only `/preflight` gives you the binary, and it uses whatever CA bundle the
destination image has. If that image is `scratch` or `distroless/base` and you
need `preflight http https://...`, copy a bundle across as well:

```dockerfile
COPY --from=ghcr.io/vertti/preflight:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
```

Running the preflight image directly (`docker run ghcr.io/vertti/preflight …`)
needs nothing extra — the bundle ships in it.

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
