package jsoncheck

import "os"

// FileSystem abstracts file operations for testing.
type FileSystem interface {
	ReadFile(name string) ([]byte, error)
}

// RealFileSystem implements FileSystem using the real file system.
type RealFileSystem struct{}

// ReadFile reads the entire file contents.
func (r *RealFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name) //nolint:gosec // intentional: file path from user config
}
