package resourcecheck

import "runtime"

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

// NumCPUs returns the number of available CPUs.
// Go's runtime.NumCPU() already respects container CPU limits.
func (r *RealResourceChecker) NumCPUs() int {
	return runtime.NumCPU()
}
