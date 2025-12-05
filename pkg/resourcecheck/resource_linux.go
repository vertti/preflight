//go:build linux

package resourcecheck

import "syscall"

// getSystemMemory returns total system memory on Linux.
func getSystemMemory() (uint64, error) {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0, err
	}
	return info.Totalram * uint64(info.Unit), nil
}
