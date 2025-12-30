package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// executeCommand runs a command with the given args and returns the output.
// It resets all flags to their default values before execution to avoid
// state pollution between tests.
func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	resetFlags(rootCmd)

	err := rootCmd.Execute()
	return buf.String(), err
}

// resetFlags resets all flags on a command and its subcommands to defaults.
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Value.Type() == "stringSlice" || f.Value.Type() == "intSlice" {
			_ = f.Value.Set("")
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

// writeTempFile creates a temp file with content and returns its path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func TestVersionFlag(t *testing.T) {
	output, err := executeCommand("--version")
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}
	if !strings.Contains(output, "preflight") {
		t.Errorf("output should contain 'preflight', got: %s", output)
	}
}

func TestHelpFlag(t *testing.T) {
	output, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	if !strings.Contains(output, "preflight") {
		t.Errorf("help should mention preflight, got: %s", output)
	}
}

func TestEnvCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func()
		wantErr bool
	}{
		{
			name:    "existing variable",
			args:    []string{"env", "PATH"},
			wantErr: false,
		},
		{
			name:    "missing variable",
			args:    []string{"env", "PREFLIGHT_NONEXISTENT_VAR_12345"},
			wantErr: true,
		},
		{
			name:    "not-set with unset variable",
			args:    []string{"env", "--not-set", "PREFLIGHT_NONEXISTENT_VAR_12345"},
			wantErr: false,
		},
		{
			name:    "not-set with set variable",
			args:    []string{"env", "--not-set", "PATH"},
			wantErr: true,
		},
		{
			name:    "missing argument",
			args:    []string{"env"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestEnvExactMatch(t *testing.T) {
	t.Setenv("PREFLIGHT_TEST_VAR", "expected_value")
	_, err := executeCommand("env", "--exact", "expected_value", "PREFLIGHT_TEST_VAR")
	if err != nil {
		t.Errorf("env --exact should pass for matching value: %v", err)
	}
}

func TestSysCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "matching os",
			args:    []string{"sys", "--os", runtime.GOOS},
			wantErr: false,
		},
		{
			name:    "matching arch",
			args:    []string{"sys", "--arch", runtime.GOARCH},
			wantErr: false,
		},
		{
			name:    "matching os and arch",
			args:    []string{"sys", "--os", runtime.GOOS, "--arch", runtime.GOARCH},
			wantErr: false,
		},
		{
			name:    "wrong os",
			args:    []string{"sys", "--os", "plan9"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestCmdCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "existing command with version-cmd (go)",
			args:    []string{"cmd", "--version-cmd", "version", "go"},
			wantErr: false,
		},
		{
			name:    "nonexistent command",
			args:    []string{"cmd", "nonexistent_command_xyz_12345"},
			wantErr: true,
		},
		{
			name:    "missing argument",
			args:    []string{"cmd"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestFileCommand(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "content")
		_, err := executeCommand("file", path)
		if err != nil {
			t.Errorf("file should pass for existing file: %v", err)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := executeCommand("file", "/nonexistent/path/file.txt")
		if err == nil {
			t.Error("file should fail for nonexistent file")
		}
	})

	t.Run("not-empty with content", func(t *testing.T) {
		path := writeTempFile(t, "nonempty.txt", "has content")
		_, err := executeCommand("file", "--not-empty", path)
		if err != nil {
			t.Errorf("file --not-empty should pass: %v", err)
		}
	})

	t.Run("not-empty with empty file", func(t *testing.T) {
		path := writeTempFile(t, "empty.txt", "")
		_, err := executeCommand("file", "--not-empty", path)
		if err == nil {
			t.Error("file --not-empty should fail for empty file")
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("file")
		if err == nil {
			t.Error("file should fail without argument")
		}
	})
}

func TestUserCommand(t *testing.T) {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		t.Skip("USER environment variable not set")
	}

	t.Run("current user", func(t *testing.T) {
		_, err := executeCommand("user", currentUser)
		if err != nil {
			t.Errorf("user should pass for current user: %v", err)
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("user")
		if err == nil {
			t.Error("user should fail without argument")
		}
	})
}

func TestResourceCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "min-memory 1MB",
			args:    []string{"resource", "--min-memory", "1MB"},
			wantErr: false,
		},
		{
			name:    "min-cpus 1",
			args:    []string{"resource", "--min-cpus", "1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommand(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestTCPCommand(t *testing.T) {
	_, err := executeCommand("tcp")
	if err == nil {
		t.Error("tcp should fail without argument")
	}
}

func TestJSONCommand(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		path := writeTempFile(t, "valid.json", `{"key": "value"}`)
		_, err := executeCommand("json", path)
		if err != nil {
			t.Errorf("json should pass for valid JSON: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		path := writeTempFile(t, "invalid.json", `{not valid}`)
		_, err := executeCommand("json", path)
		if err == nil {
			t.Error("json should fail for invalid JSON")
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("json")
		if err == nil {
			t.Error("json should fail without argument")
		}
	})
}

func TestHashCommand(t *testing.T) {
	t.Run("valid hash", func(t *testing.T) {
		path := writeTempFile(t, "hashtest.txt", "test content")
		// SHA256 of "test content"
		hash := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
		_, err := executeCommand("hash", "--sha256", hash, path)
		if err != nil {
			t.Errorf("hash should pass for correct hash: %v", err)
		}
	})

	t.Run("wrong hash", func(t *testing.T) {
		path := writeTempFile(t, "hashtest.txt", "test content")
		wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"
		_, err := executeCommand("hash", "--sha256", wrongHash, path)
		if err == nil {
			t.Error("hash should fail for wrong hash")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		_, err := executeCommand("hash", "--sha256", hash, "/nonexistent/file.txt")
		if err == nil {
			t.Error("hash should fail for nonexistent file")
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("hash")
		if err == nil {
			t.Error("hash should fail without argument")
		}
	})
}

func TestGitCommand(t *testing.T) {
	t.Run("clean flag", func(t *testing.T) {
		// Test --clean flag - git status check runs in actual repo
		_, err := executeCommand("git", "--clean")
		// Result depends on repo state, but shouldn't error
		if err != nil && !strings.Contains(err.Error(), "check failed") {
			t.Errorf("git --clean unexpected error: %v", err)
		}
	})

	t.Run("tag flag with no tag", func(t *testing.T) {
		// HEAD likely doesn't have a tag in test environment
		_, err := executeCommand("git", "--tag", "v999.999.999")
		if err == nil {
			t.Error("git --tag with nonexistent tag should fail")
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("git")
		if err == nil {
			t.Error("git should fail without flags")
		}
	})
}

func TestRunCommand(t *testing.T) {
	// Note: We only test error paths here because run spawns subprocesses
	// via exec.Command which makes it difficult to test the happy path.
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := executeCommand("run", "--file", "/nonexistent/.preflight")
		if err == nil {
			t.Error("run should fail for nonexistent file")
		}
	})
}

func TestSubcommandHelp(t *testing.T) {
	subcommands := []string{"env", "sys", "cmd", "file", "user", "resource", "tcp", "json", "hash", "git", "run", "http", "prometheus"}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			output, err := executeCommand(subcmd, "--help")
			if err != nil {
				t.Errorf("%s --help failed: %v", subcmd, err)
			}
			if output == "" {
				t.Errorf("%s --help returned empty output", subcmd)
			}
		})
	}
}

func TestHTTPCommand(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		_, err := executeCommand("http", ts.URL)
		if err != nil {
			t.Errorf("http should pass for 200 response: %v", err)
		}
	})

	t.Run("wrong status code", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		_, err := executeCommand("http", ts.URL)
		if err == nil {
			t.Error("http should fail for 404 response")
		}
	})

	t.Run("expected status code", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		_, err := executeCommand("http", "--status", "404", ts.URL)
		if err != nil {
			t.Errorf("http --status 404 should pass: %v", err)
		}
	})

	t.Run("contains check", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status": "healthy"}`))
		}))
		defer ts.Close()

		_, err := executeCommand("http", "--contains", "healthy", ts.URL)
		if err != nil {
			t.Errorf("http --contains should pass: %v", err)
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("http")
		if err == nil {
			t.Error("http should fail without argument")
		}
	})
}

func TestTCPCommandWithServer(t *testing.T) {
	// Start a TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	addr := listener.Addr().String()

	t.Run("reachable port", func(t *testing.T) {
		_, err := executeCommand("tcp", addr)
		if err != nil {
			t.Errorf("tcp should pass for listening port: %v", err)
		}
	})

	t.Run("unreachable port", func(t *testing.T) {
		// Use a port that's definitely not listening
		_, err := executeCommand("tcp", "--timeout", "100ms", "127.0.0.1:1")
		if err == nil {
			t.Error("tcp should fail for unreachable port")
		}
	})
}

func TestPrometheusCommand(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1234567890,"1"]}]}}`))
		}))
		defer ts.Close()

		_, err := executeCommand("prometheus", ts.URL, "--query", "up")
		if err != nil {
			t.Errorf("prometheus should pass: %v", err)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}))
		defer ts.Close()

		_, err := executeCommand("prometheus", ts.URL, "--query", "nonexistent_metric")
		if err == nil {
			t.Error("prometheus should fail for empty result")
		}
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("prometheus")
		if err == nil {
			t.Error("prometheus should fail without argument")
		}
	})
}

func TestResourceCommandMore(t *testing.T) {
	t.Run("min-disk", func(t *testing.T) {
		_, err := executeCommand("resource", "--min-disk", "1MB")
		if err != nil {
			t.Errorf("resource --min-disk 1MB should pass: %v", err)
		}
	})

	t.Run("combined flags", func(t *testing.T) {
		_, err := executeCommand("resource", "--min-memory", "1MB", "--min-cpus", "1")
		if err != nil {
			t.Errorf("resource with multiple flags should pass: %v", err)
		}
	})

	t.Run("no flags", func(t *testing.T) {
		_, err := executeCommand("resource")
		if err == nil {
			t.Error("resource without flags should fail")
		}
	})
}

func TestHashCommandMore(t *testing.T) {
	t.Run("sha512", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "test")
		// SHA512 of "test"
		hash := "ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732c304cc5fa9ad8e6f57f50028a8ff"
		_, err := executeCommand("hash", "--sha512", hash, path)
		if err != nil {
			t.Errorf("hash --sha512 should pass: %v", err)
		}
	})

	t.Run("md5", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "test")
		// MD5 of "test"
		hash := "098f6bcd4621d373cade4e832627b4f6"
		_, err := executeCommand("hash", "--md5", hash, path)
		if err != nil {
			t.Errorf("hash --md5 should pass: %v", err)
		}
	})

	t.Run("checksum file valid", func(t *testing.T) {
		content := "test"
		targetPath := writeTempFile(t, "target.txt", content)
		// SHA256 of "test"
		checksumContent := fmt.Sprintf("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08  %s\n", filepath.Base(targetPath))
		checksumPath := writeTempFile(t, "SHA256SUMS", checksumContent)

		// Change to temp dir so relative path works
		oldWd, _ := os.Getwd()
		if err := os.Chdir(filepath.Dir(targetPath)); err != nil {
			t.Fatalf("failed to chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		_, err := executeCommand("hash", "--checksum-file", checksumPath, filepath.Base(targetPath))
		if err != nil {
			t.Errorf("hash --checksum-file should pass for valid checksum: %v", err)
		}
	})
}
