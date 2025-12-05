package resourcecheck

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// ResourceChecker abstracts system resource detection for testability.
type ResourceChecker interface {
	// FreeDiskSpace returns free disk space in bytes at the given path.
	FreeDiskSpace(path string) (uint64, error)

	// AvailableMemory returns available memory in bytes.
	// In containers, this respects cgroup limits.
	AvailableMemory() (uint64, error)

	// NumCPUs returns the number of available CPUs.
	NumCPUs() int
}

// RealResourceChecker implements ResourceChecker using actual system calls.
type RealResourceChecker struct{}

// FreeDiskSpace returns free disk space in bytes.
func (r *RealResourceChecker) FreeDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Available blocks * block size
	return stat.Bavail * uint64(stat.Bsize), nil
}

// AvailableMemory returns available memory in bytes.
// First checks cgroup limits (for containers), then falls back to system memory.
func (r *RealResourceChecker) AvailableMemory() (uint64, error) {
	// Try cgroup v2 first
	if mem, err := readCgroupMemoryLimit("/sys/fs/cgroup/memory.max"); err == nil && mem > 0 {
		return mem, nil
	}

	// Try cgroup v1
	if mem, err := readCgroupMemoryLimit("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil && mem > 0 {
		return mem, nil
	}

	// Fall back to system memory
	return getSystemMemory()
}

// NumCPUs returns the number of available CPUs.
// Go's runtime.NumCPU() already respects container CPU limits.
func (r *RealResourceChecker) NumCPUs() int {
	return runtime.NumCPU()
}

// readCgroupMemoryLimit reads memory limit from a cgroup file.
func readCgroupMemoryLimit(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	content := strings.TrimSpace(string(data))

	// "max" means unlimited in cgroup v2
	if content == "max" {
		return 0, nil
	}

	return strconv.ParseUint(content, 10, 64)
}
