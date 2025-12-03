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

// MockFileSystem is a test double for FileSystem.
type MockFileSystem struct {
	StatFunc     func(name string) (fs.FileInfo, error)
	ReadFileFunc func(name string, limit int64) ([]byte, error)
}

// Stat calls the mock function.
func (m *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	return m.StatFunc(name)
}

// ReadFile calls the mock function.
func (m *MockFileSystem) ReadFile(name string, limit int64) ([]byte, error) {
	return m.ReadFileFunc(name, limit)
}

// MockFileInfo is a test double for fs.FileInfo.
type MockFileInfo struct {
	NameValue    string
	SizeValue    int64
	ModeValue    fs.FileMode
	IsDirValue   bool
	ModTimeValue int64
}

func (m *MockFileInfo) Name() string       { return m.NameValue }
func (m *MockFileInfo) Size() int64        { return m.SizeValue }
func (m *MockFileInfo) Mode() fs.FileMode  { return m.ModeValue }
func (m *MockFileInfo) IsDir() bool        { return m.IsDirValue }
func (m *MockFileInfo) Sys() interface{}   { return nil }
func (m *MockFileInfo) ModTime() time.Time { return time.Unix(m.ModTimeValue, 0) }
