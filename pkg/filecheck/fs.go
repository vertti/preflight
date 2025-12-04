package filecheck

import (
	"io"
	"io/fs"
	"os"
	"time"
)

// FileSystem abstracts file system operations for testability.
type FileSystem interface {
	Stat(name string) (fs.FileInfo, error)
	ReadFile(name string, limit int64) ([]byte, error)
}

// RealFileSystem implements FileSystem using the actual file system.
type RealFileSystem struct{}

// Stat returns file info for the given path.
func (r *RealFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile reads a file's contents, optionally limited to the first `limit` bytes.
// If limit is 0 or negative, reads the entire file.
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

// mockFileSystem is a test double for FileSystem.
type mockFileSystem struct {
	StatFunc     func(name string) (fs.FileInfo, error)
	ReadFileFunc func(name string, limit int64) ([]byte, error)
}

// Stat calls the mock function.
func (m *mockFileSystem) Stat(name string) (fs.FileInfo, error) {
	return m.StatFunc(name)
}

// ReadFile calls the mock function.
func (m *mockFileSystem) ReadFile(name string, limit int64) ([]byte, error) {
	return m.ReadFileFunc(name, limit)
}

// mockFileInfo is a test double for fs.FileInfo.
type mockFileInfo struct {
	NameValue    string
	SizeValue    int64
	ModeValue    fs.FileMode
	IsDirValue   bool
	ModTimeValue int64
}

func (m *mockFileInfo) Name() string       { return m.NameValue }
func (m *mockFileInfo) Size() int64        { return m.SizeValue }
func (m *mockFileInfo) Mode() fs.FileMode  { return m.ModeValue }
func (m *mockFileInfo) IsDir() bool        { return m.IsDirValue }
func (m *mockFileInfo) Sys() interface{}   { return nil }
func (m *mockFileInfo) ModTime() time.Time { return time.Unix(m.ModTimeValue, 0) }
