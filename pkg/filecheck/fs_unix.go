//go:build unix

package filecheck

import (
	"fmt"
	"os"
	"syscall"
)

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
