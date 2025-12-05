//go:build darwin

package resourcecheck

import (
	"syscall"
	"unsafe"
)

// getSystemMemory returns total system memory on macOS.
func getSystemMemory() (uint64, error) {
	mib := []int32{6, 24} // CTL_HW, HW_MEMSIZE
	var size uint64
	length := unsafe.Sizeof(size)

	_, _, err := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(2), // len(mib)
		uintptr(unsafe.Pointer(&size)),
		uintptr(unsafe.Pointer(&length)),
		0,
		0,
	)
	if err != 0 {
		return 0, err
	}
	return size, nil
}
