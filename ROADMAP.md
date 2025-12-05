# Preflight Roadmap

This roadmap documents planned enhancements based on analysis of real-world shell validation patterns in Dockerfiles, CI pipelines, and entrypoint scripts. Each feature includes the shell patterns it replaces—proof these solve real problems.

## Already Implemented

| Command          | Purpose                                                    |
| ---------------- | ---------------------------------------------------------- |
| `preflight cmd`  | Verify binary exists and runs, version constraints         |
| `preflight env`  | Validate environment variables with pattern matching       |
| `preflight file` | Check files/directories exist with permissions and content |
| `preflight http` | HTTP health checks with status, retry, headers             |
| `preflight tcp`  | TCP port connectivity                                      |
| `preflight user` | Verify user exists with uid/gid/home                       |
| `preflight run`  | Run checks from `.preflight` file                          |

---

## Priority 1: High Impact

### `preflight hash`

Supply chain security—official Docker images universally verify downloaded binaries.

**Shell patterns replaced:**

```bash
# SHA256 checksum verification
echo "67574ee...2cf myfile.tar.gz" | sha256sum -c

# From checksum file
sha256sum -c checksums.txt

# Node.js official image pattern
grep " node-v$VERSION.tar.gz\$" SHASUMS256.txt | sha256sum -c -
```

**Must-have flags:**

| Flag              | Description                        |
| ----------------- | ---------------------------------- |
| `--sha256 <hash>` | Expected SHA256 hash               |
| `--file <path>`   | File to verify (or positional arg) |

**Optional flags:**

| Flag                     | Description                  |
| ------------------------ | ---------------------------- |
| `--sha512 <hash>`        | SHA512 hash                  |
| `--md5 <hash>`           | MD5 hash (legacy)            |
| `--checksum-file <path>` | Verify against checksum file |

---

## Priority 2: System & Resources

### `preflight dns`

DNS resolution validation for service discovery.

**Shell patterns replaced:**

```bash
# DNS lookup
getent hosts "hostname" || { echo "DNS lookup failed"; exit 1; }

# nslookup fallback
nslookup myservice.local || exit 1
```

**Must-have flags:**

| Flag                   | Description         |
| ---------------------- | ------------------- |
| (positional)           | Hostname to resolve |
| `--timeout <duration>` | Lookup timeout      |

**Optional flags:**

| Flag     | Description         |
| -------- | ------------------- |
| `--ipv4` | Require IPv4 result |
| `--ipv6` | Require IPv6 result |

---

### `preflight sys`

Multi-architecture containers require platform detection.

**Shell patterns replaced:**

```bash
# Architecture detection (note: uname -m returns x86_64/aarch64)
arch=$(uname -m | sed s/aarch64/arm64/ | sed s/x86_64/amd64/)

# OS detection
case $(uname -s | tr '[:upper:]' '[:lower:]') in
  linux*)  echo 'linux' ;;
  darwin*) echo 'darwin' ;;
esac

# Kernel version check
kernelVersion="$(uname -r)"
```

**Must-have flags:**

| Flag            | Description                          |
| --------------- | ------------------------------------ |
| `--arch <arch>` | Required architecture (amd64, arm64) |
| `--os <os>`     | Required OS (linux, darwin)          |

**Optional flags:**

| Flag                     | Description                               |
| ------------------------ | ----------------------------------------- |
| `--kernel-min <version>` | Minimum kernel version                    |
| `--distro <name>`        | Linux distribution (alpine, debian, etc.) |

---

### `preflight resource`

CI environments (GitHub Actions runners) need resource validation.

**Shell patterns replaced:**

```bash
# Disk space
df -h /var/lib/docker

# Memory (container-aware via cgroups)
cat /sys/fs/cgroup/memory/memory.limit_in_bytes

# CPU cores
nproc

# File descriptor limits
ulimit -n
```

**Must-have flags:**

| Flag                  | Description                   |
| --------------------- | ----------------------------- |
| `--min-disk <size>`   | Minimum free disk (e.g., 10G) |
| `--min-memory <size>` | Minimum memory (e.g., 2G)     |

**Optional flags:**

| Flag                  | Description              |
| --------------------- | ------------------------ |
| `--min-cpus <n>`      | Minimum CPU cores        |
| `--path <path>`       | Check disk space at path |
| `--ulimit <resource>` | Check ulimit value       |

---

## Priority 3: Advanced

### `preflight pkg`

Package verification varies by distribution.

**Shell patterns replaced:**

```bash
# Debian/Ubuntu
dpkg-query -W -f='${Status}' curl | grep "ok installed"

# RHEL/CentOS
rpm -q openssh-server

# Alpine
apk info git

# Python
pip show numpy > /dev/null 2>&1

# Node.js
npm list express --depth=0
```

**Must-have flags:**

| Flag               | Description                               |
| ------------------ | ----------------------------------------- |
| (positional)       | Package name                              |
| `--manager <type>` | Package manager (apt, rpm, apk, pip, npm) |

**Optional flags:**

| Flag                | Description                 |
| ------------------- | --------------------------- |
| `--min-version <v>` | Minimum package version     |
| `--auto-detect`     | Auto-detect package manager |

---

### `preflight proc`

Validate dependent services before starting applications.

**Shell patterns replaced:**

```bash
# Process running check
pgrep -f "nginx" || exit 1

# pidof for exact name
pidof postgres > /dev/null

# systemd service
systemctl is-active --quiet sshd
```

**Must-have flags:**

| Flag                  | Description                          |
| --------------------- | ------------------------------------ |
| `--running <pattern>` | Process name/pattern must be running |

**Optional flags:**

| Flag                | Description                             |
| ------------------- | --------------------------------------- |
| `--service <name>`  | Systemd service must be active          |
| `--pid-file <path>` | PID file must exist and process running |

---

### `preflight file` Extensions

Extend existing file check with socket and symlink support.

**Shell patterns replaced:**

```bash
# Socket existence
test -S /var/run/docker.sock

# Wait for MySQL socket
while [ ! -S /var/run/mysqld/mysqld.sock ]; do sleep 1; done

# Symlink detection
if [ -L "$0" ]; then
    DIR=$(dirname $(readlink -f "$0"))
fi
```

**New flags:**

| Flag                      | Description                  |
| ------------------------- | ---------------------------- |
| `--socket`                | Path must be a socket        |
| `--symlink`               | Path must be a symlink       |
| `--symlink-target <path>` | Symlink must point to target |

---

### `preflight git`

CI pipelines validate repository state.

**Shell patterns replaced:**

```bash
# Working directory clean
if [ -z "$(git status --porcelain)" ]; then
    echo "Clean"
fi

# Check for uncommitted changes
git diff --exit-code

# Version from tags
git describe --tags --abbrev=0
```

**Must-have flags:**

| Flag      | Description                     |
| --------- | ------------------------------- |
| `--clean` | Working directory must be clean |

**Optional flags:**

| Flag                    | Description                 |
| ----------------------- | --------------------------- |
| `--branch <name>`       | Must be on branch           |
| `--tag-match <pattern>` | HEAD must match tag pattern |
| `--no-untracked`        | No untracked files          |

---

### `preflight cert`

TLS certificate validation for internal registries.

**Shell patterns replaced:**

```bash
# Certificate chain validation
openssl s_client -verify 5 -connect registry:443 -showcerts

# Check expiration
openssl x509 -checkend 86400 -noout -in /path/to/cert.pem
```

**Must-have flags:**

| Flag                        | Description             |
| --------------------------- | ----------------------- |
| `--verify-host <host:port>` | Verify TLS connection   |
| `--cert-file <path>`        | Verify certificate file |

**Optional flags:**

| Flag                 | Description                     |
| -------------------- | ------------------------------- |
| `--not-expired`      | Certificate must not be expired |
| `--min-days <n>`     | Minimum days until expiration   |
| `--issuer <pattern>` | Issuer must match pattern       |
| `--ca-bundle <path>` | Custom CA bundle                |

---

## Summary

| Priority | Command    | Impact                    |
| -------- | ---------- | ------------------------- |
| 1        | `hash`     | Supply chain security     |
| 2        | `dns`      | Service discovery         |
| 2        | `sys`      | Multi-arch containers     |
| 2        | `resource` | CI environment validation |
| 3        | `pkg`      | Package verification      |
| 3        | `proc`     | Service orchestration     |
| 3        | `file` ext | Sockets, symlinks         |
| 3        | `git`      | CI automation             |
| 3        | `cert`     | TLS validation            |
