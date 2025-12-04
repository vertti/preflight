package filecheck

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

func TestCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
		wantDetail string
	}{
		// Basic existence tests
		{
			name: "file exists",
			check: Check{
				Path: "/etc/config",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "config",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
			wantDetail: "type: file",
		},
		{
			name: "file not found",
			check: Check{
				Path: "/missing",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return nil, os.ErrNotExist
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "not found",
		},
		{
			name: "permission denied",
			check: Check{
				Path: "/secret",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return nil, os.ErrPermission
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "permission denied",
		},

		// Directory tests
		{
			name: "directory exists",
			check: Check{
				Path:      "/var/log",
				ExpectDir: true,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue:  "log",
							IsDirValue: true,
							ModeValue:  0o755 | fs.ModeDir,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
			wantDetail: "type: directory",
		},
		{
			name: "expected directory got file",
			check: Check{
				Path:      "/etc/config",
				ExpectDir: true,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "config",
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "expected directory, got file",
		},

		// Size tests
		{
			name: "not empty - passes",
			check: Check{
				Path:     "/data.json",
				NotEmpty: true,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "data.json",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "not empty - fails",
			check: Check{
				Path:     "/empty.txt",
				NotEmpty: true,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "empty.txt",
							SizeValue: 0,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "file is empty",
		},
		{
			name: "min size - passes",
			check: Check{
				Path:    "/data.json",
				MinSize: 50,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "data.json",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "min size - fails",
			check: Check{
				Path:    "/data.json",
				MinSize: 200,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "data.json",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "size 100 < minimum 200",
		},
		{
			name: "max size - passes",
			check: Check{
				Path:    "/data.json",
				MaxSize: 200,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "data.json",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "max size - fails",
			check: Check{
				Path:    "/data.json",
				MaxSize: 50,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "data.json",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "size 100 > maximum 50",
		},

		// Permission tests
		{
			name: "writable - passes",
			check: Check{
				Path:     "/data.json",
				Writable: true,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "data.json",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "writable - fails",
			check: Check{
				Path:     "/readonly.txt",
				Writable: true,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "readonly.txt",
							SizeValue: 100,
							ModeValue: 0o444,
						}, nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "not writable",
		},
		{
			name: "executable - passes",
			check: Check{
				Path:       "/script.sh",
				Executable: true,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "script.sh",
							SizeValue: 100,
							ModeValue: 0o755,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "executable - fails",
			check: Check{
				Path:       "/data.txt",
				Executable: true,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "data.txt",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "not executable",
		},

		// Mode tests
		{
			name: "mode minimum - passes (exact match)",
			check: Check{
				Path: "/key.pem",
				Mode: "0600",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "key.pem",
							SizeValue: 100,
							ModeValue: 0o600,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "mode minimum - passes (more permissive)",
			check: Check{
				Path: "/key.pem",
				Mode: "0600",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "key.pem",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "mode minimum - fails",
			check: Check{
				Path: "/key.pem",
				Mode: "0644",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "key.pem",
							SizeValue: 100,
							ModeValue: 0o600,
						}, nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "permissions -rw------- do not include minimum -rw-r--r--",
		},
		{
			name: "mode exact - passes",
			check: Check{
				Path:      "/key.pem",
				ModeExact: "0600",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "key.pem",
							SizeValue: 100,
							ModeValue: 0o600,
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "mode exact - fails",
			check: Check{
				Path:      "/key.pem",
				ModeExact: "0600",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "key.pem",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "permissions -rw-r--r-- != required -rw-------",
		},

		// Content tests
		{
			name: "contains - passes",
			check: Check{
				Path:     "/config.txt",
				Contains: "database",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "config.txt",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
					ReadFileFunc: func(name string, limit int64) ([]byte, error) {
						return []byte("database_url=postgres://localhost"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "contains - fails",
			check: Check{
				Path:     "/config.txt",
				Contains: "redis",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "config.txt",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
					ReadFileFunc: func(name string, limit int64) ([]byte, error) {
						return []byte("database_url=postgres://localhost"), nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "content does not contain \"redis\"",
		},
		{
			name: "match regex - passes",
			check: Check{
				Path:  "/hosts",
				Match: "^127\\.0\\.0\\.1",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "hosts",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
					ReadFileFunc: func(name string, limit int64) ([]byte, error) {
						return []byte("127.0.0.1 localhost"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "match regex - fails",
			check: Check{
				Path:  "/hosts",
				Match: "^192\\.168",
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "hosts",
							SizeValue: 100,
							ModeValue: 0o644,
						}, nil
					},
					ReadFileFunc: func(name string, limit int64) ([]byte, error) {
						return []byte("127.0.0.1 localhost"), nil
					},
				},
			},
			wantStatus: check.StatusFail,
			wantDetail: "content does not match pattern \"^192\\\\.168\"",
		},
		{
			name: "head limit is passed to ReadFile",
			check: Check{
				Path:     "/large.log",
				Contains: "ERROR",
				Head:     1024,
				FS: &mockFileSystem{
					StatFunc: func(name string) (fs.FileInfo, error) {
						return &mockFileInfo{
							NameValue: "large.log",
							SizeValue: 10000,
							ModeValue: 0o644,
						}, nil
					},
					ReadFileFunc: func(name string, limit int64) ([]byte, error) {
						if limit != 1024 {
							return nil, errors.New("expected limit 1024")
						}
						return []byte("ERROR: something failed"), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", result.Status, tt.wantStatus)
			}

			if tt.wantDetail != "" {
				found := false
				for _, d := range result.Details {
					if d == tt.wantDetail {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected detail %q not found in %v", tt.wantDetail, result.Details)
				}
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
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOctalMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseOctalMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
