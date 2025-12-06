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
