//go:build unix

package filecheck

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// CanAccess asks the kernel whether this process may access the path, using the
// effective uid/gid. AT_EACCESS matters for setuid binaries, where the real and
// effective ids differ.
func (r *RealFileSystem) CanAccess(name string, mode AccessMode) (bool, error) {
	var mask uint32
	switch mode {
	case AccessWrite:
		mask = unix.W_OK
	case AccessExecute:
		mask = unix.X_OK
	default:
		return false, fmt.Errorf("unknown access mode %d", mode)
	}

	err := unix.Faccessat(unix.AT_FDCWD, name, mask, unix.AT_EACCESS)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, unix.EACCES), errors.Is(err, unix.EPERM), errors.Is(err, unix.EROFS):
		return false, nil
	default:
		return false, err
	}
}

// GetOwner returns the UID and GID of the file owner.
func (r *RealFileSystem) GetOwner(name string) (uid, gid uint32, err error) {
	info, err := os.Stat(name)
	if err != nil {
		return 0, 0, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected file info type for %s", name)
	}
	return stat.Uid, stat.Gid, nil
}
