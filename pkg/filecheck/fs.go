package filecheck

import (
	"io"
	"io/fs"
	"os"
)

type FileSystem interface {
	Stat(name string) (fs.FileInfo, error)
	ReadFile(name string, limit int64) ([]byte, error)
}

type RealFileSystem struct{}

func (r *RealFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile reads up to limit bytes. If limit <= 0, reads entire file.
func (r *RealFileSystem) ReadFile(name string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return os.ReadFile(name)
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	return io.ReadAll(io.LimitReader(f, limit))
}
