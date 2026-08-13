//go:build windows

package resourcecheck

import "errors"

// FreeDiskSpace is not supported on Windows.
func (r *RealResourceChecker) FreeDiskSpace(path string) (uint64, error) {
	return 0, errors.New("disk space check not supported on Windows")
}

// AvailableMemory is not supported on Windows.
func (r *RealResourceChecker) AvailableMemory() (uint64, error) {
	return 0, errors.New("memory check not supported on Windows")
}

// cpuQuota reports no quota: Windows job objects express CPU limits differently
// and are not read here.
func cpuQuota() (float64, bool) {
	return 0, false
}
