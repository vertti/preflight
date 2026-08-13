package resourcecheck

import "runtime"

// ResourceChecker abstracts system resource detection for testability.
type ResourceChecker interface {
	// FreeDiskSpace returns free disk space in bytes at the given path.
	FreeDiskSpace(path string) (uint64, error)

	// AvailableMemory returns available memory in bytes.
	// In containers, this respects cgroup limits.
	AvailableMemory() (uint64, error)

	// AvailableCPUs returns the number of CPUs the process can use.
	// In containers, this respects CPU quotas, which are fractional.
	AvailableCPUs() float64
}

// RealResourceChecker implements ResourceChecker using actual system calls.
type RealResourceChecker struct{}

// AvailableCPUs returns the number of CPUs the process can use.
func (r *RealResourceChecker) AvailableCPUs() float64 {
	return availableCPUs(runtime.NumCPU(), cpuQuota)
}

// availableCPUs combines the two ways a container's CPUs can be restricted.
// runtime.NumCPU() reports how many CPUs the process may be scheduled on, which
// covers a cpuset but not a CFS quota — a container limited to 1.5 CPUs still
// sees every core on the host. The quota is a share of wall time, so it can be
// a fraction, and it only matters when it is the tighter of the two.
func availableCPUs(affinity int, quota func() (float64, bool)) float64 {
	cpus := float64(affinity)
	if limit, limited := quota(); limited && limit < cpus {
		return limit
	}
	return cpus
}
