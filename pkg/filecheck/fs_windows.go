//go:build windows

package filecheck

import (
	"errors"
	"fmt"
	"os"
)

// GetOwner is not supported on Windows.
func (r *RealFileSystem) GetOwner(name string) (uid, gid uint32, err error) {
	return 0, 0, errors.New("file ownership check not supported on Windows")
}

// CanAccess approximates access from the permission bits. Windows has no
// access(2) and Go synthesizes 0666 (or 0444 when read-only) for regular files,
// so this reflects the read-only attribute rather than the real ACL.
func (r *RealFileSystem) CanAccess(name string, mode AccessMode) (bool, error) {
	info, err := os.Stat(name)
	if err != nil {
		return false, err
	}

	perm := info.Mode().Perm()
	switch mode {
	case AccessWrite:
		return perm&0o222 != 0, nil
	case AccessExecute:
		return perm&0o111 != 0, nil
	default:
		return false, fmt.Errorf("unknown access mode %d", mode)
	}
}
