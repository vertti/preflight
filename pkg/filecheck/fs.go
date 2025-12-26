package filecheck

import (
	"io"
	"io/fs"
	"os"
)

type FileSystem interface {
	Stat(name string) (fs.FileInfo, error)
	Lstat(name string) (fs.FileInfo, error)
	ReadFile(name string, limit int64) ([]byte, error)
	Readlink(name string) (string, error)
	GetOwner(name string) (uid, gid uint32, err error)
}

type RealFileSystem struct{}

func (r *RealFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (r *RealFileSystem) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(name)
}

func (r *RealFileSystem) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

// ReadFile reads up to limit bytes. If limit <= 0, reads entire file.
func (r *RealFileSystem) ReadFile(name string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return os.ReadFile(name) //nolint:gosec // intentional: file path from user config
	}

	f, err := os.Open(name) //nolint:gosec // intentional: file path from user config
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	return io.ReadAll(io.LimitReader(f, limit))
}
