# Preflight Roadmap

This roadmap is organized by **real-world frequency** based on analysis of ~500 Dockerfiles, CI pipelines, and entrypoint scripts. Features are prioritized by how often these shell patterns appear in production.

## Already Implemented

These commands replace extremely common shell patterns found in virtually every production setup.

| Command              | Tools/Patterns Replaced                                       | Frequency  |
| -------------------- | ------------------------------------------------------------- | ---------- |
| `preflight tcp`      | wait-for-it.sh (9.7k stars), dockerize (4.8k stars), `nc -z`  | ⭐⭐⭐⭐⭐ |
| `preflight user`     | `id -u` checks, gosu privilege dropping, every official image | ⭐⭐⭐⭐⭐ |
| `preflight sys`      | `uname -m \| sed`, `dpkg --print-architecture`, TARGETARCH    | ⭐⭐⭐⭐⭐ |
| `preflight cmd`      | `which`, `command -v`, version parsing scripts                | ⭐⭐⭐⭐⭐ |
| `preflight env`      | `test -z "$VAR"`, parameter expansion checks                  | ⭐⭐⭐⭐⭐ |
| `preflight file`     | `test -f`, `test -d`, `test -r`, `-S` socket, ownership       | ⭐⭐⭐⭐⭐ |
| `preflight http`     | `curl --fail`, `wget --spider`, healthcheck scripts           | ⭐⭐⭐⭐   |
| `preflight hash`     | `sha256sum -c`, GPG verification scripts                      | ⭐⭐⭐⭐   |
| `preflight git`      | `git status --porcelain`, `git diff --exit-code`, CI checks   | ⭐⭐⭐⭐   |
| `preflight resource` | `df`, cgroup memory limits, `nproc`                           | ⭐⭐⭐⭐   |
| `preflight json`     | `jq empty`, JSON validation, key extraction                   | ⭐⭐⭐⭐   |

---

## Priority 1: Common (Next Up)

These patterns appear frequently in specific contexts like multi-stage builds, security hardening, or service orchestration.

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

## Priority 2: Occasional

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
| 1        | `proc`            | Healthchecks, service orchestration |
| 1        | `pkg`             | Package verification                |
| 1        | `cert`            | TLS validation                      |
| 1        | `yaml`            | Kubernetes manifest validation      |
| 2        | `dns`             | Service discovery                   |
| 2        | `file` (symlinks) | Capability management               |
