package resourcecheck

import (
	"github.com/vertti/preflight/pkg/check"
)

// Check verifies system resources meet requirements.
type Check struct {
	MinDisk   uint64 // minimum free disk space in bytes
	MinMemory uint64 // minimum available memory in bytes
	MinCPUs   int    // minimum number of CPUs
	Path      string // path for disk space check (default: ".")
	Checker   ResourceChecker
}

// Run executes the resource check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: "resource",
	}

	if c.Checker == nil {
		c.Checker = &RealResourceChecker{}
	}

	path := c.Path
	if path == "" {
		path = "."
	}

	// Check disk space
	if c.MinDisk > 0 {
		free, err := c.Checker.FreeDiskSpace(path)
		if err != nil {
			return result.Failf("failed to check disk space: %v", err)
		}

		result.AddDetailf("disk free: %s (path: %s)", FormatSize(free), path)

		if free < c.MinDisk {
			return result.Failf("disk space %s < required %s", FormatSize(free), FormatSize(c.MinDisk))
		}
	}

	// Check memory
	if c.MinMemory > 0 {
		mem, err := c.Checker.AvailableMemory()
		if err != nil {
			return result.Failf("failed to check memory: %v", err)
		}

		result.AddDetailf("memory: %s", FormatSize(mem))

		if mem < c.MinMemory {
			return result.Failf("memory %s < required %s", FormatSize(mem), FormatSize(c.MinMemory))
		}
	}

	// Check CPUs
	if c.MinCPUs > 0 {
		cpus := c.Checker.NumCPUs()

		result.AddDetailf("cpus: %d", cpus)

		if cpus < c.MinCPUs {
			return result.Failf("cpus %d < required %d", cpus, c.MinCPUs)
		}
	}

	result.Status = check.StatusOK
	return result
}
