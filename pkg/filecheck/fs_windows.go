//go:build windows

package filecheck

import "errors"

// GetOwner is not supported on Windows.
func (r *RealFileSystem) GetOwner(name string) (uid, gid uint32, err error) {
	return 0, 0, errors.New("file ownership check not supported on Windows")
}
