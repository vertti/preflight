//go:build windows

package resourcecheck

import (
	"fmt"
	"runtime"
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
// Not implemented on Windows.
func (r *RealResourceChecker) FreeDiskSpace(path string) (uint64, error) {
	return 0, fmt.Errorf("disk space check not supported on Windows")
}

// AvailableMemory returns available memory in bytes.
// Not implemented on Windows.
func (r *RealResourceChecker) AvailableMemory() (uint64, error) {
	return 0, fmt.Errorf("memory check not supported on Windows")
}

// NumCPUs returns the number of available CPUs.
// Go's runtime.NumCPU() already respects container CPU limits.
func (r *RealResourceChecker) NumCPUs() int {
	return runtime.NumCPU()
}
