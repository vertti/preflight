package filecheck

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/testutil"
)

type mockFileSystem struct {
	StatFunc     func(name string) (fs.FileInfo, error)
	LstatFunc    func(name string) (fs.FileInfo, error)
	ReadFileFunc func(name string, limit int64) ([]byte, error)
	ReadlinkFunc func(name string) (string, error)
	GetOwnerFunc func(name string) (uid, gid uint32, err error)
}

func (m *mockFileSystem) Stat(name string) (fs.FileInfo, error)  { return m.StatFunc(name) }
func (m *mockFileSystem) Lstat(name string) (fs.FileInfo, error) { return m.LstatFunc(name) }
func (m *mockFileSystem) ReadFile(name string, limit int64) ([]byte, error) {
	return m.ReadFileFunc(name, limit)
}
func (m *mockFileSystem) Readlink(name string) (string, error) {
	if m.ReadlinkFunc != nil {
		return m.ReadlinkFunc(name)
	}
	return "", nil
}
func (m *mockFileSystem) GetOwner(name string) (uid, gid uint32, err error) {
	if m.GetOwnerFunc != nil {
		return m.GetOwnerFunc(name)
	}
	return 0, 0, nil
}

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
func (m *mockFileInfo) Sys() any           { return nil }
func (m *mockFileInfo) ModTime() time.Time { return time.Unix(m.ModTimeValue, 0) }

func statFile(mode fs.FileMode, size int64) func(string) (fs.FileInfo, error) {
	return func(string) (fs.FileInfo, error) {
		return &mockFileInfo{NameValue: "f", ModeValue: mode, SizeValue: size}, nil
	}
}

func statDir() func(string) (fs.FileInfo, error) {
	return func(string) (fs.FileInfo, error) {
		return &mockFileInfo{NameValue: "d", ModeValue: 0o755 | fs.ModeDir, IsDirValue: true}, nil
	}
}

func statErr(err error) func(string) (fs.FileInfo, error) {
	return func(string) (fs.FileInfo, error) { return nil, err }
}

func TestCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
		wantDetail string
	}{
		// Basic existence
		{"file exists", Check{Path: "/etc/config", FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusOK, "type: file"},
		{"file not found", Check{Path: "/missing", FS: &mockFileSystem{StatFunc: statErr(os.ErrNotExist)}}, check.StatusFail, "not found"},
		{"permission denied", Check{Path: "/secret", FS: &mockFileSystem{StatFunc: statErr(os.ErrPermission)}}, check.StatusFail, "permission denied"},
		{"generic stat error", Check{Path: "/broken", FS: &mockFileSystem{StatFunc: statErr(errors.New("I/O error"))}}, check.StatusFail, "stat failed: I/O error"},

		// Directory
		{"directory exists", Check{Path: "/var/log", ExpectDir: true, FS: &mockFileSystem{StatFunc: statDir()}}, check.StatusOK, "type: directory"},
		{"expected dir got file", Check{Path: "/etc/config", ExpectDir: true, FS: &mockFileSystem{StatFunc: statFile(0o644, 0)}}, check.StatusFail, "expected directory, got file"},

		// Socket
		{"socket exists", Check{Path: "/var/run/docker.sock", ExpectSocket: true, FS: &mockFileSystem{StatFunc: func(string) (fs.FileInfo, error) {
			return &mockFileInfo{NameValue: "docker.sock", ModeValue: 0o600 | fs.ModeSocket}, nil
		}}}, check.StatusOK, "type: socket"},
		{"expected socket got file", Check{Path: "/etc/config", ExpectSocket: true, FS: &mockFileSystem{StatFunc: statFile(0o644, 0)}}, check.StatusFail, "expected socket"},
		{"expected socket got dir", Check{Path: "/var/log", ExpectSocket: true, FS: &mockFileSystem{StatFunc: statDir()}}, check.StatusFail, "expected socket"},

		// Size
		{"not empty passes", Check{Path: "/data.json", NotEmpty: true, FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusOK, ""},
		{"not empty fails", Check{Path: "/empty.txt", NotEmpty: true, FS: &mockFileSystem{StatFunc: statFile(0o644, 0)}}, check.StatusFail, "file is empty"},
		{"min size passes", Check{Path: "/data.json", MinSize: 50, FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusOK, ""},
		{"min size fails", Check{Path: "/data.json", MinSize: 200, FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusFail, "size 100 < minimum 200"},
		{"max size passes", Check{Path: "/data.json", MaxSize: 200, FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusOK, ""},
		{"max size fails", Check{Path: "/data.json", MaxSize: 50, FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusFail, "size 100 > maximum 50"},

		// Permissions
		{"writable passes", Check{Path: "/data.json", Writable: true, FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusOK, ""},
		{"writable fails", Check{Path: "/readonly.txt", Writable: true, FS: &mockFileSystem{StatFunc: statFile(0o444, 100)}}, check.StatusFail, "not writable"},
		{"executable passes", Check{Path: "/script.sh", Executable: true, FS: &mockFileSystem{StatFunc: statFile(0o755, 100)}}, check.StatusOK, ""},
		{"executable fails", Check{Path: "/data.txt", Executable: true, FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusFail, "not executable"},

		// Mode
		{"mode minimum exact", Check{Path: "/key.pem", Mode: "0600", FS: &mockFileSystem{StatFunc: statFile(0o600, 100)}}, check.StatusOK, ""},
		{"mode minimum permissive", Check{Path: "/key.pem", Mode: "0600", FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusOK, ""},
		{"mode minimum fails", Check{Path: "/key.pem", Mode: "0644", FS: &mockFileSystem{StatFunc: statFile(0o600, 100)}}, check.StatusFail, "do not include minimum"},
		{"mode exact passes", Check{Path: "/key.pem", ModeExact: "0600", FS: &mockFileSystem{StatFunc: statFile(0o600, 100)}}, check.StatusOK, ""},
		{"mode exact fails", Check{Path: "/key.pem", ModeExact: "0600", FS: &mockFileSystem{StatFunc: statFile(0o644, 100)}}, check.StatusFail, "!= required"},
		{"invalid mode string", Check{Path: "/file.txt", Mode: "invalid", FS: &mockFileSystem{StatFunc: statFile(0o644, 0)}}, check.StatusFail, ""},
		{"invalid mode-exact string", Check{Path: "/file.txt", ModeExact: "notoctal", FS: &mockFileSystem{StatFunc: statFile(0o644, 0)}}, check.StatusFail, ""},

		// Content
		{"contains passes", Check{Path: "/config.txt", Contains: "database", FS: &mockFileSystem{StatFunc: statFile(0o644, 100), ReadFileFunc: func(string, int64) ([]byte, error) { return []byte("database_url=postgres"), nil }}}, check.StatusOK, ""},
		{"contains fails", Check{Path: "/config.txt", Contains: "redis", FS: &mockFileSystem{StatFunc: statFile(0o644, 100), ReadFileFunc: func(string, int64) ([]byte, error) { return []byte("database_url=postgres"), nil }}}, check.StatusFail, "does not contain"},
		{"match regex passes", Check{Path: "/hosts", Match: `^127\.0\.0\.1`, FS: &mockFileSystem{StatFunc: statFile(0o644, 100), ReadFileFunc: func(string, int64) ([]byte, error) { return []byte("127.0.0.1 localhost"), nil }}}, check.StatusOK, ""},
		{"match regex fails", Check{Path: "/hosts", Match: `^192\.168`, FS: &mockFileSystem{StatFunc: statFile(0o644, 100), ReadFileFunc: func(string, int64) ([]byte, error) { return []byte("127.0.0.1 localhost"), nil }}}, check.StatusFail, "does not match pattern"},
		{"head limit passed", Check{Path: "/large.log", Contains: "ERROR", Head: 1024, FS: &mockFileSystem{StatFunc: statFile(0o644, 10000), ReadFileFunc: func(_ string, limit int64) ([]byte, error) {
			if limit != 1024 {
				return nil, errors.New("expected limit 1024")
			}
			return []byte("ERROR: something failed"), nil
		}}}, check.StatusOK, ""},
		{"read file error", Check{Path: "/unreadable.txt", Contains: "test", FS: &mockFileSystem{StatFunc: statFile(0o644, 0), ReadFileFunc: func(string, int64) ([]byte, error) { return nil, errors.New("permission denied") }}}, check.StatusFail, ""},
		{"invalid regex", Check{Path: "/config.txt", Match: "[invalid", FS: &mockFileSystem{StatFunc: statFile(0o644, 0), ReadFileFunc: func(string, int64) ([]byte, error) { return []byte("content"), nil }}}, check.StatusFail, ""},

		// Symlink
		{"symlink exists", Check{Path: "/usr/bin/python", ExpectSymlink: true, FS: &mockFileSystem{LstatFunc: func(string) (fs.FileInfo, error) {
			return &mockFileInfo{NameValue: "python", ModeValue: 0o777 | fs.ModeSymlink}, nil
		}, ReadlinkFunc: func(string) (string, error) { return "/usr/bin/python3", nil }}}, check.StatusOK, "type: symlink"},
		{"expected symlink got file", Check{Path: "/usr/bin/python", ExpectSymlink: true, FS: &mockFileSystem{LstatFunc: statFile(0o755, 0)}}, check.StatusFail, "expected symlink"},
		{"symlink target matches", Check{Path: "/usr/bin/python", ExpectSymlink: true, SymlinkTarget: "/usr/bin/python3", FS: &mockFileSystem{LstatFunc: func(string) (fs.FileInfo, error) {
			return &mockFileInfo{NameValue: "python", ModeValue: 0o777 | fs.ModeSymlink}, nil
		}, ReadlinkFunc: func(string) (string, error) { return "/usr/bin/python3", nil }}}, check.StatusOK, "target: /usr/bin/python3"},
		{"symlink target mismatch", Check{Path: "/usr/bin/python", ExpectSymlink: true, SymlinkTarget: "/usr/bin/python3.11", FS: &mockFileSystem{LstatFunc: func(string) (fs.FileInfo, error) {
			return &mockFileInfo{NameValue: "python", ModeValue: 0o777 | fs.ModeSymlink}, nil
		}, ReadlinkFunc: func(string) (string, error) { return "/usr/bin/python3.10", nil }}}, check.StatusFail, "symlink target"},
		{"readlink error", Check{Path: "/usr/bin/python", ExpectSymlink: true, SymlinkTarget: "/usr/bin/python3", FS: &mockFileSystem{LstatFunc: func(string) (fs.FileInfo, error) {
			return &mockFileInfo{NameValue: "python", ModeValue: fs.ModeSymlink | 0o777}, nil
		}, ReadlinkFunc: func(string) (string, error) { return "", errors.New("readlink failed") }}}, check.StatusFail, "failed to read symlink target"},

		// Owner
		{"owner matches", Check{Path: "/data", Owner: 1000, FS: &mockFileSystem{StatFunc: statFile(0o644, 0), GetOwnerFunc: func(string) (uid, gid uint32, err error) { return 1000, 1000, nil }}}, check.StatusOK, "owner: 1000"},
		{"owner mismatch", Check{Path: "/data", Owner: 1000, FS: &mockFileSystem{StatFunc: statFile(0o644, 0), GetOwnerFunc: func(string) (uid, gid uint32, err error) { return 0, 0, nil }}}, check.StatusFail, "owner 0 != expected 1000"},
		{"owner root", Check{Path: "/data", Owner: 0, FS: &mockFileSystem{StatFunc: statFile(0o644, 0), GetOwnerFunc: func(string) (uid, gid uint32, err error) { return 0, 0, nil }}}, check.StatusOK, "owner: 0"},
		{"owner not checked", Check{Path: "/data", Owner: -1, FS: &mockFileSystem{StatFunc: statFile(0o644, 0)}}, check.StatusOK, ""},
		{"GetOwner error", Check{Path: "/data", Owner: 1000, FS: &mockFileSystem{StatFunc: statFile(0o644, 0), GetOwnerFunc: func(string) (uid, gid uint32, err error) { return 0, 0, errors.New("unable to get owner") }}}, check.StatusFail, "failed to get owner"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			if tt.wantDetail != "" {
				assert.True(t, testutil.ContainsDetail(result.Details, tt.wantDetail), "details %v should contain %q", result.Details, tt.wantDetail)
			}
		})
	}
}

func TestParseOctalMode(t *testing.T) {
	tests := []struct {
		input   string
		want    fs.FileMode
		wantErr bool
	}{
		{"0644", 0o644, false},
		{"644", 0o644, false},
		{"0755", 0o755, false},
		{"0600", 0o600, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseOctalMode(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRealFileSystem(t *testing.T) {
	rfs := &RealFileSystem{}

	t.Run("Stat", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-stat-*")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()
		_ = tmpFile.Close()

		info, err := rfs.Stat(tmpFile.Name())
		require.NoError(t, err)
		assert.NotNil(t, info)
	})

	t.Run("Stat nonexistent", func(t *testing.T) {
		_, err := rfs.Stat("/nonexistent/file/path")
		assert.Error(t, err)
	})

	t.Run("Lstat symlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := tmpDir + "/target"
		link := tmpDir + "/link"

		require.NoError(t, os.WriteFile(target, []byte("content"), 0o600))
		require.NoError(t, os.Symlink(target, link))

		info, err := rfs.Lstat(link)
		require.NoError(t, err)
		assert.True(t, info.Mode()&os.ModeSymlink != 0)
	})

	t.Run("Readlink", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := tmpDir + "/target"
		link := tmpDir + "/link"

		require.NoError(t, os.WriteFile(target, []byte("content"), 0o600))
		require.NoError(t, os.Symlink(target, link))

		dest, err := rfs.Readlink(link)
		require.NoError(t, err)
		assert.Equal(t, target, dest)
	})

	t.Run("ReadFile full", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := tmpDir + "/testfile"
		content := []byte("test content here")

		require.NoError(t, os.WriteFile(path, content, 0o600))

		data, err := rfs.ReadFile(path, 0)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("ReadFile limited", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := tmpDir + "/testfile"
		content := []byte("test content here that is longer")

		require.NoError(t, os.WriteFile(path, content, 0o600))

		data, err := rfs.ReadFile(path, 10)
		require.NoError(t, err)
		assert.Len(t, data, 10)
	})

	t.Run("ReadFile nonexistent", func(t *testing.T) {
		_, err := rfs.ReadFile("/nonexistent/file", 0)
		assert.Error(t, err)
	})

	t.Run("ReadFile limited nonexistent", func(t *testing.T) {
		_, err := rfs.ReadFile("/nonexistent/file", 10)
		assert.Error(t, err)
	})

	t.Run("GetOwner", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-owner-*")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()
		_ = tmpFile.Close()

		_, _, err = rfs.GetOwner(tmpFile.Name())
		require.NoError(t, err)
	})
}
