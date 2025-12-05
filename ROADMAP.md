# Preflight Roadmap

This roadmap is organized by **real-world frequency** based on analysis of ~500 Dockerfiles, CI pipelines, and entrypoint scripts. Features are prioritized by how often these shell patterns appear in production.

## Already Implemented

These commands replace extremely common shell patterns found in virtually every production setup.

| Command          | Tools/Patterns Replaced                                       | Frequency  |
| ---------------- | ------------------------------------------------------------- | ---------- |
| `preflight tcp`  | wait-for-it.sh (9.7k stars), dockerize (4.8k stars), `nc -z`  | ⭐⭐⭐⭐⭐ |
| `preflight user` | `id -u` checks, gosu privilege dropping, every official image | ⭐⭐⭐⭐⭐ |
| `preflight sys`  | `uname -m \| sed`, `dpkg --print-architecture`, TARGETARCH    | ⭐⭐⭐⭐⭐ |
| `preflight cmd`  | `which`, `command -v`, version parsing scripts                | ⭐⭐⭐⭐⭐ |
| `preflight env`  | `test -z "$VAR"`, parameter expansion checks                  | ⭐⭐⭐⭐⭐ |
| `preflight file` | `test -f`, `test -d`, `test -r`, permission checks            | ⭐⭐⭐⭐⭐ |
| `preflight http` | `curl --fail`, `wget --spider`, healthcheck scripts           | ⭐⭐⭐⭐   |
| `preflight hash` | `sha256sum -c`, GPG verification scripts                      | ⭐⭐⭐⭐   |

---

## Priority 1: Very Common (Tier 1)

These patterns appear in thousands of repositories and are standard practice.

### `preflight file --socket`

Unix socket checks are **critical for Docker-in-Docker** and service orchestration.

**Tools/patterns replaced:**

```bash
# From docker-library/docker - nearly all DinD implementations
if [ -z "${DOCKER_HOST:-}" ] && [ -S /var/run/docker.sock ]; then
    export DOCKER_HOST=unix:///var/run/docker.sock
fi

# Wait for containerd socket
while [ ! -S "/run/containerd/containerd.sock" ]; do
    sleep .25
done
```

**Proposed syntax:**

```sh
preflight file /var/run/docker.sock --socket
preflight file /run/containerd/containerd.sock --socket
```

---

### `preflight resource`

Disk space is **CI's biggest pain point**. GitHub-hosted runners have only 25-29 GB free, which large Docker builds exceed. Multiple GitHub Actions exist solely to address this.

**Tools/patterns replaced:**

- [jlumbroso/free-disk-space](https://github.com/jlumbroso/free-disk-space) - thousands of stars
- [easimon/maximize-build-space](https://github.com/easimon/maximize-build-space) - thousands of stars

```bash
# Disk space
df -h /var/lib/docker

# Memory (container-aware via cgroups)
cat /sys/fs/cgroup/memory/memory.limit_in_bytes  # cgroup v1
cat /sys/fs/cgroup/memory.max                     # cgroup v2

# CPU cores (container-aware)
nproc

# File descriptor limits
ulimit -n
```

**Proposed flags:**

| Flag                  | Description                   |
| --------------------- | ----------------------------- |
| `--min-disk <size>`   | Minimum free disk (e.g., 10G) |
| `--min-memory <size>` | Minimum memory (e.g., 2G)     |
| `--min-cpus <n>`      | Minimum CPU cores             |
| `--path <path>`       | Check disk space at path      |

**Proposed syntax:**

```sh
preflight resource --min-disk 10G --path /var/lib/docker
preflight resource --min-memory 2G
preflight resource --min-cpus 2
```

---

### `preflight git`

Git state verification is **standard in CI pipelines** for Go projects (vendor checks, formatting) and any project with generated code.

**Tools/patterns replaced:**

```bash
# From docker/docs Dockerfile - vendor verification
hugo mod vendor
if [ -n "$(git status --porcelain -- go.mod go.sum _vendor)" ]; then
    echo 'ERROR: Vendor result differs'
    exit 1
fi

# Universal formatting check pattern
go fmt ./... && git diff --exit-code
```

**Proposed flags:**

| Flag                    | Description                     |
| ----------------------- | ------------------------------- |
| `--clean`               | Working directory must be clean |
| `--no-uncommitted`      | No uncommitted changes          |
| `--no-untracked`        | No untracked files              |
| `--branch <name>`       | Must be on branch               |
| `--tag-match <pattern>` | HEAD must match tag pattern     |

**Proposed syntax:**

```sh
preflight git --clean
preflight git --no-uncommitted
preflight git --branch main
```

---

### `preflight json`

The `jq` tool has become the **de facto standard** for JSON processing in shell scripts and CI. Validation patterns are ubiquitous.

**Tools/patterns replaced:**

- [CICDToolbox/json-lint](https://github.com/CICDToolbox/json-lint)
- `jq` validation patterns

```bash
# Validation pattern
if echo "$json_data" | jq empty > /dev/null 2>&1; then
    echo "Valid JSON"
fi

# Docker output parsing
docker inspect container | jq '.[0].State.Health.Status'
```

**Proposed flags:**

| Flag                | Description                  |
| ------------------- | ---------------------------- |
| `--file <path>`     | JSON file to validate        |
| `--query <jq-expr>` | Extract value with jq syntax |
| `--contains <key>`  | JSON must contain key        |
| `--type <type>`     | Root must be object or array |

**Proposed syntax:**

```sh
preflight json --file config.json
preflight json --file config.json --contains "database.host"
preflight json --file package.json --query ".version"
```

---

## Priority 2: Common (Tier 2)

These patterns appear frequently in specific contexts like multi-stage builds, security hardening, or service orchestration.

### `preflight file --owner`

File ownership verification is **standard in HashiCorp images** for detecting bind-mounted volumes with incorrect ownership.

**Tools/patterns replaced:**

```bash
# From Consul and Vault official images
if [ "$(stat -c %u "$CONSUL_DATA_DIR")" != "$(id -u consul)" ]; then
    chown consul:consul "$CONSUL_DATA_DIR"
fi
```

**Proposed syntax:**

```sh
preflight file /data --owner 1000
preflight file /data --owner consul
preflight file /data --group 1000
```

---

### `preflight proc`

Process checks are the **most common Docker healthcheck pattern**, despite being considered a shell "code smell."

**Tools/patterns replaced:**

```dockerfile
# From elastic/beats Filebeat
HEALTHCHECK --interval=5s --timeout=3s \
    CMD ps aux | grep '[f]ilebeat' || exit 1
```

```bash
# From mbentley/docker-in-docker
DOCKERD_PID="$(pgrep dockerd || true)"
if [ -f "/var/run/docker.pid" ] && [ -z "${DOCKERD_PID}" ]; then
    rm /var/run/docker.pid  # cleanup stale PID
fi
```

**Proposed flags:**

| Flag                  | Description                             |
| --------------------- | --------------------------------------- |
| `--running <pattern>` | Process name/pattern must be running    |
| `--pid-file <path>`   | PID file must exist and process running |

**Proposed syntax:**

```sh
preflight proc --running nginx
preflight proc --running "filebeat"
preflight proc --pid-file /var/run/nginx.pid
```

---

### `preflight pkg`

Package verification is used for idempotent installation and dependency checking.

**Tools/patterns replaced:**

```bash
# Debian/Ubuntu
dpkg -s nginx 2>/dev/null || apt-get install -y nginx

# Alpine
apk info git || apk add git

# Python
pip show requests || pip install requests
```

**Proposed flags:**

| Flag                | Description                               |
| ------------------- | ----------------------------------------- |
| (positional)        | Package name                              |
| `--manager <type>`  | Package manager (apt, rpm, apk, pip, npm) |
| `--min-version <v>` | Minimum package version                   |
| `--auto-detect`     | Auto-detect package manager               |

**Proposed syntax:**

```sh
preflight pkg nginx --manager apt
preflight pkg requests --manager pip --min-version 2.0
preflight pkg git --auto-detect
```

---

### `preflight cert`

TLS certificate validation for internal registries and security-conscious deployments.

**Tools/patterns replaced:**

```bash
# Certificate chain validation
openssl s_client -verify 5 -connect registry:443 -showcerts

# Check expiration (from healthchecks)
HEALTHCHECK CMD openssl x509 -in /certs/cert.pem -noout -checkend 86400
```

**Proposed flags:**

| Flag                        | Description                   |
| --------------------------- | ----------------------------- |
| `--verify-host <host:port>` | Verify TLS connection         |
| `--cert-file <path>`        | Verify certificate file       |
| `--not-expired`             | Certificate must be valid     |
| `--min-days <n>`            | Minimum days until expiration |

**Proposed syntax:**

```sh
preflight cert --cert-file /certs/cert.pem --not-expired
preflight cert --cert-file /certs/cert.pem --min-days 30
preflight cert --verify-host registry:443
```

---

### `preflight yaml`

YAML validation is growing in Kubernetes/cloud-native environments with the popularity of `yq`.

**Tools/patterns replaced:**

- [mikefarah/yq](https://github.com/mikefarah/yq)
- [HighwayofLife/kubernetes-validation-tools](https://github.com/HighwayofLife/kubernetes-validation-tools)

```bash
# Kubernetes manifest validation
yq e 'true' deployment.yaml > /dev/null || exit 1
```

**Proposed syntax:**

```sh
preflight yaml --file deployment.yaml
preflight yaml --file config.yaml --contains "spec.replicas"
```

---

## Priority 3: Occasional (Tier 3)

These patterns address specialized use cases.

### `preflight dns`

DNS resolution validation for service discovery and debugging.

**Tools/patterns replaced:**

```bash
# Docker hostname resolution
getent hosts host.docker.internal | awk '{print $1}'

# nslookup fallback
nslookup myservice.local || exit 1
```

**Proposed flags:**

| Flag                   | Description         |
| ---------------------- | ------------------- |
| (positional)           | Hostname to resolve |
| `--timeout <duration>` | Lookup timeout      |
| `--ipv4`               | Require IPv4 result |
| `--ipv6`               | Require IPv6 result |

**Proposed syntax:**

```sh
preflight dns myservice.local
preflight dns host.docker.internal --timeout 5s
```

---

### `preflight file` Extensions (Symlinks)

Symlink resolution handles path complexity in capability management.

**Tools/patterns replaced:**

```bash
# From HashiCorp Vault
setcap cap_ipc_lock=-ep $(readlink -f $(which vault))
```

**Proposed new flags:**

| Flag                      | Description                  |
| ------------------------- | ---------------------------- |
| `--symlink`               | Path must be a symlink       |
| `--symlink-target <path>` | Symlink must point to target |

---

## Summary by Impact

| Priority | Command           | Impact                              |
| -------- | ----------------- | ----------------------------------- |
| 1        | `file --socket`   | Docker-in-Docker, containerd        |
| 1        | `resource`        | CI disk/memory validation           |
| 1        | `git`             | CI clean state verification         |
| 1        | `json`            | Config validation, replaces jq      |
| 2        | `file --owner`    | Bind mount permission fixes         |
| 2        | `proc`            | Healthchecks, service orchestration |
| 2        | `pkg`             | Package verification                |
| 2        | `cert`            | TLS validation                      |
| 2        | `yaml`            | Kubernetes manifest validation      |
| 3        | `dns`             | Service discovery                   |
| 3        | `file` (symlinks) | Capability management               |
