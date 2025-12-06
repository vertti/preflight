//go:build unix

package filecheck

import (
	"os"
	"syscall"
)

// GetOwner returns the UID and GID of the file owner.
func (r *RealFileSystem) GetOwner(name string) (uid, gid uint32, err error) {
	info, err := os.Stat(name)
	if err != nil {
		return 0, 0, err
	}
	stat := info.Sys().(*syscall.Stat_t)
	return stat.Uid, stat.Gid, nil
}
