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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	resetFlags(rootCmd)
	err := rootCmd.Execute()
	return buf.String(), err
}

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

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestVersionFlag(t *testing.T) {
	output, err := executeCommand("--version")
	require.NoError(t, err)
	assert.Contains(t, output, "preflight")
}

func TestHelpFlag(t *testing.T) {
	output, err := executeCommand("--help")
	require.NoError(t, err)
	assert.Contains(t, output, "preflight")
}

func TestEnvCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"existing variable", []string{"env", "PATH"}, false},
		{"missing variable", []string{"env", "PREFLIGHT_NONEXISTENT_VAR_12345"}, true},
		{"not-set with unset variable", []string{"env", "--not-set", "PREFLIGHT_NONEXISTENT_VAR_12345"}, false},
		{"not-set with set variable", []string{"env", "--not-set", "PATH"}, true},
		{"missing argument", []string{"env"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnvExactMatch(t *testing.T) {
	t.Setenv("PREFLIGHT_TEST_VAR", "expected_value")
	_, err := executeCommand("env", "--exact", "expected_value", "PREFLIGHT_TEST_VAR")
	assert.NoError(t, err)
}

func TestEnvMinMaxValue(t *testing.T) {
	t.Setenv("PREFLIGHT_NUM_VAR", "50")

	t.Run("min-value pass", func(t *testing.T) {
		_, err := executeCommand("env", "--is-numeric", "--min-value", "10", "PREFLIGHT_NUM_VAR")
		assert.NoError(t, err)
	})

	t.Run("max-value pass", func(t *testing.T) {
		_, err := executeCommand("env", "--is-numeric", "--max-value", "100", "PREFLIGHT_NUM_VAR")
		assert.NoError(t, err)
	})

	t.Run("min-value fail", func(t *testing.T) {
		_, err := executeCommand("env", "--is-numeric", "--min-value", "60", "PREFLIGHT_NUM_VAR")
		assert.Error(t, err)
	})

	t.Run("max-value fail", func(t *testing.T) {
		_, err := executeCommand("env", "--is-numeric", "--max-value", "40", "PREFLIGHT_NUM_VAR")
		assert.Error(t, err)
	})
}

func TestSysCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"matching os", []string{"sys", "--os", runtime.GOOS}, false},
		{"matching arch", []string{"sys", "--arch", runtime.GOARCH}, false},
		{"matching os and arch", []string{"sys", "--os", runtime.GOOS, "--arch", runtime.GOARCH}, false},
		{"wrong os", []string{"sys", "--os", "plan9"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
		{"existing command with version-cmd", []string{"cmd", "--version-cmd", "version", "go"}, false},
		{"nonexistent command", []string{"cmd", "nonexistent_command_xyz_12345"}, true},
		{"missing argument", []string{"cmd"}, true},
		{"invalid min version", []string{"cmd", "--min", "not-a-version", "go"}, true},
		{"invalid max version", []string{"cmd", "--max", "not-a-version", "go"}, true},
		{"invalid exact version", []string{"cmd", "--exact", "not-a-version", "go"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileCommand(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "content")
		_, err := executeCommand("file", path)
		assert.NoError(t, err)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := executeCommand("file", "/nonexistent/path/file.txt")
		assert.Error(t, err)
	})

	t.Run("not-empty with content", func(t *testing.T) {
		path := writeTempFile(t, "nonempty.txt", "has content")
		_, err := executeCommand("file", "--not-empty", path)
		assert.NoError(t, err)
	})

	t.Run("not-empty with empty file", func(t *testing.T) {
		path := writeTempFile(t, "empty.txt", "")
		_, err := executeCommand("file", "--not-empty", path)
		assert.Error(t, err)
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("file")
		assert.Error(t, err)
	})
}

func TestUserCommand(t *testing.T) {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		t.Skip("USER environment variable not set")
	}

	t.Run("current user", func(t *testing.T) {
		_, err := executeCommand("user", currentUser)
		assert.NoError(t, err)
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("user")
		assert.Error(t, err)
	})
}

func TestResourceCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"min-memory 1MB", []string{"resource", "--min-memory", "1MB"}, false},
		{"min-cpus 1", []string{"resource", "--min-cpus", "1"}, false},
		{"min-disk 1MB", []string{"resource", "--min-disk", "1MB"}, false},
		{"combined flags", []string{"resource", "--min-memory", "1MB", "--min-cpus", "1"}, false},
		{"no flags", []string{"resource"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTCPCommand(t *testing.T) {
	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("tcp")
		assert.Error(t, err)
	})

	t.Run("reachable port", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()

		_, err = executeCommand("tcp", listener.Addr().String())
		assert.NoError(t, err)
	})

	t.Run("unreachable port", func(t *testing.T) {
		_, err := executeCommand("tcp", "--timeout", "100ms", "127.0.0.1:1")
		assert.Error(t, err)
	})
}

func TestJSONCommand(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		path := writeTempFile(t, "valid.json", `{"key": "value"}`)
		_, err := executeCommand("json", path)
		assert.NoError(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		path := writeTempFile(t, "invalid.json", `{not valid}`)
		_, err := executeCommand("json", path)
		assert.Error(t, err)
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("json")
		assert.Error(t, err)
	})
}

func TestHashCommand(t *testing.T) {
	t.Run("valid sha256", func(t *testing.T) {
		path := writeTempFile(t, "hashtest.txt", "test content")
		hash := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
		_, err := executeCommand("hash", "--sha256", hash, path)
		assert.NoError(t, err)
	})

	t.Run("wrong hash", func(t *testing.T) {
		path := writeTempFile(t, "hashtest.txt", "test content")
		_, err := executeCommand("hash", "--sha256", strings.Repeat("0", 64), path)
		assert.Error(t, err)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := executeCommand("hash", "--sha256", strings.Repeat("0", 64), "/nonexistent/file.txt")
		assert.Error(t, err)
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("hash")
		assert.Error(t, err)
	})
}

func TestGitCommand(t *testing.T) {
	t.Run("clean flag", func(t *testing.T) {
		_, err := executeCommand("git", "--clean")
		if err != nil {
			assert.Contains(t, err.Error(), "check failed")
		}
	})

	t.Run("tag flag with nonexistent tag", func(t *testing.T) {
		_, err := executeCommand("git", "--tag", "v999.999.999")
		assert.Error(t, err)
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("git")
		assert.Error(t, err)
	})
}

func TestRunCommand(t *testing.T) {
	// Note: Most run command tests are in integration_test.go because
	// runRun uses exec.Command which is hard to test in unit tests.
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := executeCommand("run", "--file", "/nonexistent/.preflight")
		assert.Error(t, err)
	})
}

func TestSubcommandHelp(t *testing.T) {
	subcommands := []string{"env", "sys", "cmd", "file", "user", "resource", "tcp", "json", "hash", "git", "run", "http", "prometheus"}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			output, err := executeCommand(subcmd, "--help")
			require.NoError(t, err)
			assert.NotEmpty(t, output)
		})
	}
}

func TestHTTPCommand(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		_, err := executeCommand("http", ts.URL)
		assert.NoError(t, err)
	})

	t.Run("wrong status code", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		_, err := executeCommand("http", ts.URL)
		assert.Error(t, err)
	})

	t.Run("expected status code", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		_, err := executeCommand("http", "--status", "404", ts.URL)
		assert.NoError(t, err)
	})

	t.Run("contains check", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status": "healthy"}`))
		}))
		defer ts.Close()

		_, err := executeCommand("http", "--contains", "healthy", ts.URL)
		assert.NoError(t, err)
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("http")
		assert.Error(t, err)
	})
}

func TestPrometheusCommand(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1234567890,"1"]}]}}`))
		}))
		defer ts.Close()

		_, err := executeCommand("prometheus", ts.URL, "--query", "up")
		assert.NoError(t, err)
	})

	t.Run("empty result", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}))
		defer ts.Close()

		_, err := executeCommand("prometheus", ts.URL, "--query", "nonexistent_metric")
		assert.Error(t, err)
	})

	t.Run("missing argument", func(t *testing.T) {
		_, err := executeCommand("prometheus")
		assert.Error(t, err)
	})
}

func TestHashCommandMore(t *testing.T) {
	t.Run("sha512", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "test")
		hash := "ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732c304cc5fa9ad8e6f57f50028a8ff"
		_, err := executeCommand("hash", "--sha512", hash, path)
		assert.NoError(t, err)
	})

	t.Run("sha384", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "test")
		hash := "768412320f7b0aa5812fce428dc4706b3cae50e02a64caa16a782249bfe8efc4b7ef1ccb126255d196047dfedf17a0a9"
		_, err := executeCommand("hash", "--sha384", hash, path)
		assert.NoError(t, err)
	})

	t.Run("sha1", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "test")
		hash := "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"
		_, err := executeCommand("hash", "--sha1", hash, path)
		assert.NoError(t, err)
	})

	t.Run("md5", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "test")
		hash := "098f6bcd4621d373cade4e832627b4f6"
		_, err := executeCommand("hash", "--md5", hash, path)
		assert.NoError(t, err)
	})

	t.Run("auto sha256", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "test")
		hash := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
		_, err := executeCommand("hash", "--auto", hash, path)
		assert.NoError(t, err)
	})

	t.Run("checksum file valid", func(t *testing.T) {
		targetPath := writeTempFile(t, "target.txt", "test")
		checksumContent := fmt.Sprintf("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08  %s\n", filepath.Base(targetPath))
		checksumPath := writeTempFile(t, "SHA256SUMS", checksumContent)

		oldWd, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(filepath.Dir(targetPath)))
		t.Cleanup(func() { _ = os.Chdir(oldWd) })

		_, err = executeCommand("hash", "--checksum-file", checksumPath, filepath.Base(targetPath))
		assert.NoError(t, err)
	})

	t.Run("no hash flag", func(t *testing.T) {
		path := writeTempFile(t, "test.txt", "test")
		_, err := executeCommand("hash", path)
		assert.Error(t, err)
	})
}

func TestRealFileChecker(t *testing.T) {
	t.Run("existing file returns true", func(t *testing.T) {
		path := writeTempFile(t, "testfile.txt", "content")
		assert.True(t, realFileChecker(path))
	})

	t.Run("nonexistent file returns false", func(t *testing.T) {
		assert.False(t, realFileChecker("/nonexistent/path/file.txt"))
	})

	t.Run("directory returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.False(t, realFileChecker(tmpDir))
	})
}

func TestJSONCommandErrorPaths(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := executeCommand("json", "/nonexistent/file.json")
		assert.Error(t, err)
	})

	t.Run("exact without key", func(t *testing.T) {
		path := writeTempFile(t, "test.json", `{"key": "value"}`)
		_, err := executeCommand("json", "--exact", "value", path)
		assert.Error(t, err)
	})

	t.Run("match without key", func(t *testing.T) {
		path := writeTempFile(t, "test.json", `{"key": "value"}`)
		_, err := executeCommand("json", "--match", "val.*", path)
		assert.Error(t, err)
	})

	t.Run("has-key check passes", func(t *testing.T) {
		path := writeTempFile(t, "test.json", `{"nested": {"key": "value"}}`)
		_, err := executeCommand("json", "--has-key", "nested.key", path)
		assert.NoError(t, err)
	})

	t.Run("key with exact check", func(t *testing.T) {
		path := writeTempFile(t, "test.json", `{"key": "expected"}`)
		_, err := executeCommand("json", "--key", "key", "--exact", "expected", path)
		assert.NoError(t, err)
	})

	t.Run("key with match check", func(t *testing.T) {
		path := writeTempFile(t, "test.json", `{"key": "hello123world"}`)
		_, err := executeCommand("json", "--key", "key", "--match", "hello.*world", path)
		assert.NoError(t, err)
	})
}

func TestPrometheusCommandErrorPaths(t *testing.T) {
	t.Run("invalid url", func(t *testing.T) {
		_, err := executeCommand("prometheus", "not-a-valid-url", "--query", "up")
		assert.Error(t, err)
	})

	t.Run("min check passes", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1234567890,"50"]}]}}`))
		}))
		defer ts.Close()

		_, err := executeCommand("prometheus", ts.URL, "--query", "up", "--min", "10")
		assert.NoError(t, err)
	})

	t.Run("max check fails", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1234567890,"50"]}]}}`))
		}))
		defer ts.Close()

		_, err := executeCommand("prometheus", ts.URL, "--query", "up", "--max", "10")
		assert.Error(t, err)
	})

	t.Run("exact check passes", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1234567890,"1"]}]}}`))
		}))
		defer ts.Close()

		_, err := executeCommand("prometheus", ts.URL, "--query", "up", "--exact", "1")
		assert.NoError(t, err)
	})
}

func TestResourceCommandErrorPaths(t *testing.T) {
	t.Run("disk path does not exist", func(t *testing.T) {
		_, err := executeCommand("resource", "--min-disk", "1MB", "--path", "/nonexistent/path")
		assert.Error(t, err)
	})

	t.Run("invalid memory size format", func(t *testing.T) {
		_, err := executeCommand("resource", "--min-memory", "invalid")
		assert.Error(t, err)
	})

	t.Run("invalid disk size format", func(t *testing.T) {
		_, err := executeCommand("resource", "--min-disk", "invalid")
		assert.Error(t, err)
	})

	t.Run("with custom path", func(t *testing.T) {
		_, err := executeCommand("resource", "--min-disk", "1MB", "--path", ".")
		assert.NoError(t, err)
	})
}
