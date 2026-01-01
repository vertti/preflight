package preflight_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/cmdcheck"
	"github.com/vertti/preflight/pkg/envcheck"
	"github.com/vertti/preflight/pkg/filecheck"
	"github.com/vertti/preflight/pkg/gitcheck"
	"github.com/vertti/preflight/pkg/hashcheck"
	"github.com/vertti/preflight/pkg/httpcheck"
	"github.com/vertti/preflight/pkg/jsoncheck"
	"github.com/vertti/preflight/pkg/resourcecheck"
	"github.com/vertti/preflight/pkg/syscheck"
	"github.com/vertti/preflight/pkg/tcpcheck"
	"github.com/vertti/preflight/pkg/usercheck"
)

// Integration tests verify Real* implementations work with actual system resources.
// Unit tests in each package cover edge cases; these tests verify end-to-end integration.

func TestIntegration_Cmd(t *testing.T) {
	c := cmdcheck.Check{
		Name:   "bash", // bash --version is universally available
		Runner: &cmdcheck.RealCmdRunner{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_Env(t *testing.T) {
	t.Setenv("PREFLIGHT_TEST_VAR", "test-value")

	c := envcheck.Check{
		Name:   "PREFLIGHT_TEST_VAR",
		Getter: &envcheck.RealEnvGetter{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_File(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "preflight-integration-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString("test content")
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	c := filecheck.Check{
		Path:     tmpFile.Name(),
		NotEmpty: true,
		Owner:    -1, // Don't check owner
		FS:       &filecheck.RealFileSystem{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_FileOwner(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "preflight-owner-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	// Get current user's UID
	uid := os.Getuid()

	// Test that owner matches current user
	c := filecheck.Check{
		Path:  tmpFile.Name(),
		Owner: uid,
		FS:    &filecheck.RealFileSystem{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Test owner mismatch (use impossible UID)
	c = filecheck.Check{
		Path:  tmpFile.Name(),
		Owner: 99999,
		FS:    &filecheck.RealFileSystem{},
	}

	result = c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want Fail (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_FileSocket(t *testing.T) {
	// Create a temporary Unix socket
	tmpDir, err := os.MkdirTemp("", "preflight-socket-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := tmpDir + "/test.sock"
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create unix socket: %v", err)
	}
	defer func() { _ = listener.Close() }()

	// Test that --socket flag works
	c := filecheck.Check{
		Path:         socketPath,
		ExpectSocket: true,
		Owner:        -1, // Don't check owner
		FS:           &filecheck.RealFileSystem{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Verify we see the socket type in details
	foundSocketType := slices.Contains(result.Details, "type: socket")
	if !foundSocketType {
		t.Errorf("Details = %v, want to contain 'type: socket'", result.Details)
	}
}

func TestIntegration_FileSocketFail(t *testing.T) {
	// Create a regular file, not a socket
	tmpFile, err := os.CreateTemp("", "preflight-not-socket-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	// Test that --socket fails on a regular file
	c := filecheck.Check{
		Path:         tmpFile.Name(),
		ExpectSocket: true,
		Owner:        -1, // Don't check owner
		FS:           &filecheck.RealFileSystem{},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want Fail (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_FileSymlink(t *testing.T) {
	// Create a temp file and a symlink to it
	tmpFile, err := os.CreateTemp("", "preflight-symlink-target-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	symlinkPath := tmpFile.Name() + ".link"
	if err := os.Symlink(tmpFile.Name(), symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}
	defer func() { _ = os.Remove(symlinkPath) }()

	// Test that --symlink flag works
	c := filecheck.Check{
		Path:          symlinkPath,
		ExpectSymlink: true,
		Owner:         -1,
		FS:            &filecheck.RealFileSystem{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Verify we see the symlink type in details
	foundSymlinkType := slices.Contains(result.Details, "type: symlink")
	if !foundSymlinkType {
		t.Errorf("Details = %v, want to contain 'type: symlink'", result.Details)
	}

	// Test --symlink-target
	c = filecheck.Check{
		Path:          symlinkPath,
		ExpectSymlink: true,
		SymlinkTarget: tmpFile.Name(),
		Owner:         -1,
		FS:            &filecheck.RealFileSystem{},
	}

	result = c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Test wrong target fails
	c = filecheck.Check{
		Path:          symlinkPath,
		ExpectSymlink: true,
		SymlinkTarget: "/nonexistent",
		Owner:         -1,
		FS:            &filecheck.RealFileSystem{},
	}

	result = c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want Fail (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_JSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "preflight-integration-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(`{"name": "test", "version": "1.2.3", "db": {"host": "localhost"}}`)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	// Test basic validation
	c := jsoncheck.Check{
		File: tmpFile.Name(),
		FS:   &jsoncheck.RealFileSystem{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Test --has-key with nested path
	c = jsoncheck.Check{
		File:   tmpFile.Name(),
		HasKey: "db.host",
		FS:     &jsoncheck.RealFileSystem{},
	}

	result = c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Test --key with --exact
	c = jsoncheck.Check{
		File:  tmpFile.Name(),
		Key:   "version",
		Exact: "1.2.3",
		FS:    &jsoncheck.RealFileSystem{},
	}

	result = c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Test --key with --match
	c = jsoncheck.Check{
		File:  tmpFile.Name(),
		Key:   "version",
		Match: `^1\.`,
		FS:    &jsoncheck.RealFileSystem{},
	}

	result = c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_JSONInvalid(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "preflight-invalid-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString(`{invalid json}`)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	c := jsoncheck.Check{
		File: tmpFile.Name(),
		FS:   &jsoncheck.RealFileSystem{},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want Fail (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_Hash(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "preflight-integration-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString("test content")
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	// SHA256 of "test content"
	expectedHash := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"

	c := hashcheck.Check{
		File:         tmpFile.Name(),
		ExpectedHash: expectedHash,
		Algorithm:    hashcheck.AlgorithmSHA256,
		// Opener is nil - uses RealFileOpener
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_HTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := httpcheck.Check{
		URL:    server.URL,
		Client: &httpcheck.RealHTTPClient{Timeout: 5 * time.Second},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_TCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	addr := listener.Addr().String()

	c := tcpcheck.Check{
		Address: addr,
		Timeout: 5 * time.Second,
		Dialer:  &tcpcheck.RealTCPDialer{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_User(t *testing.T) {
	username := os.Getenv("USER")
	if username == "" {
		t.Skip("USER environment variable not set")
	}

	c := usercheck.Check{
		Username: username,
		Lookup:   &usercheck.RealUserLookup{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_Sys(t *testing.T) {
	// Get the actual OS from RealSysInfo and verify it matches
	info := &syscheck.RealSysInfo{}

	c := syscheck.Check{
		ExpectedOS: info.OS(), // Use actual OS so test always passes
		Info:       info,
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_Git(t *testing.T) {
	// This test runs in the preflight repo itself, which is a git repository
	runner := &gitcheck.RealGitRunner{}

	// Verify we're in a git repo
	isRepo, err := runner.IsGitRepo()
	if err != nil {
		t.Fatalf("IsGitRepo() error = %v", err)
	}
	if !isRepo {
		t.Skip("not running in a git repository")
	}

	// Test that CurrentBranch works
	branch, err := runner.CurrentBranch()
	if err != nil {
		t.Errorf("CurrentBranch() error = %v", err)
	}
	if branch == "" {
		t.Error("CurrentBranch() returned empty string")
	}

	// Test that Status works (output varies, just verify no error)
	_, err = runner.Status()
	if err != nil {
		t.Errorf("Status() error = %v", err)
	}

	// Test that TagsAtHead works (may return empty, just verify no error)
	_, err = runner.TagsAtHead()
	if err != nil {
		t.Errorf("TagsAtHead() error = %v", err)
	}

	// Test full check with branch verification
	c := gitcheck.Check{
		Branch: branch, // Use actual branch so test passes
		Runner: runner,
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_Resource(t *testing.T) {
	checker := &resourcecheck.RealResourceChecker{}

	// Test disk space (should have at least 1MB free)
	c := resourcecheck.Check{
		MinDisk: 1 * 1024 * 1024, // 1MB
		Checker: checker,
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Test memory (should have at least 100MB)
	c = resourcecheck.Check{
		MinMemory: 100 * 1024 * 1024, // 100MB
		Checker:   checker,
	}

	result = c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}

	// Test CPUs (should have at least 1)
	c = resourcecheck.Check{
		MinCPUs: 1,
		Checker: checker,
	}

	result = c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
	}
}

func TestIntegration_Examples(t *testing.T) {
	// Verify example files exist and contain expected preflight commands
	// This catches CLI drift where docs/examples get out of sync with the tool

	examples := []struct {
		path             string
		expectedContains []string
	}{
		{
			path: "examples/multistage-dockerfile/Dockerfile",
			expectedContains: []string{
				"preflight cmd",
				"preflight file",
				"ghcr.io/vertti/preflight",
			},
		},
		{
			path: "examples/runtime-checks-with-entrypoint/Dockerfile",
			expectedContains: []string{
				"ghcr.io/vertti/preflight",
				"entrypoint.sh",
			},
		},
		{
			path: "examples/runtime-checks-with-entrypoint/entrypoint.sh",
			expectedContains: []string{
				"#!/bin/sh",
				"set -e",
				"preflight tcp",
				"preflight env",
				"preflight file",
				"exec \"$@\"",
			},
		},
	}

	for _, ex := range examples {
		t.Run(ex.path, func(t *testing.T) {
			content, err := os.ReadFile(ex.path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", ex.path, err)
			}

			for _, expected := range ex.expectedContains {
				if !strings.Contains(string(content), expected) {
					t.Errorf("%s: expected to contain %q", ex.path, expected)
				}
			}
		})
	}
}

func TestIntegration_ExamplesShellSyntax(t *testing.T) {
	// Verify shell scripts have valid syntax
	shellScripts := []string{
		"examples/runtime-checks-with-entrypoint/entrypoint.sh",
	}

	for _, script := range shellScripts {
		t.Run(script, func(t *testing.T) {
			cmd := exec.Command("sh", "-n", script) //nolint:gosec // intentional: validating shell syntax of example scripts
			if err := cmd.Run(); err != nil {
				t.Errorf("shell syntax error in %s: %v", script, err)
			}
		})
	}
}

func TestIntegration_ExecMode(t *testing.T) {
	// Build preflight binary for testing
	tmpDir := t.TempDir()
	binaryPath := tmpDir + "/preflight"

	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/preflight") //nolint:gosec // intentional: building test binary
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build preflight: %v\n%s", err, output)
	}

	t.Run("exec after successful check", func(t *testing.T) {
		// Run: preflight env PATH -- echo "success"
		cmd := exec.Command(binaryPath, "env", "PATH", "--", "echo", "exec-success-marker") //nolint:gosec // intentional: testing exec mode
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %v\n%s", err, output)
		}

		// Should see both the check output and the exec'd command output
		outputStr := string(output)
		if !strings.Contains(outputStr, "[OK] env: PATH") {
			t.Errorf("expected check output, got: %s", outputStr)
		}
		if !strings.Contains(outputStr, "exec-success-marker") {
			t.Errorf("expected exec'd command output, got: %s", outputStr)
		}
	})

	t.Run("no exec after failed check", func(t *testing.T) {
		// Run: preflight env NONEXISTENT_VAR_12345 -- echo "should-not-print"
		cmd := exec.Command(binaryPath, "env", "NONEXISTENT_VAR_12345", "--", "echo", "should-not-print") //nolint:gosec // intentional: testing exec mode
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		output, err := cmd.CombinedOutput()

		// Command should fail
		if err == nil {
			t.Fatal("expected command to fail")
		}

		// Should see the failure output but NOT the exec'd command
		outputStr := string(output)
		if !strings.Contains(outputStr, "[FAIL]") {
			t.Errorf("expected failure output, got: %s", outputStr)
		}
		if strings.Contains(outputStr, "should-not-print") {
			t.Errorf("exec'd command should not have run, got: %s", outputStr)
		}
	})

	t.Run("exec with arguments", func(t *testing.T) {
		// Run: preflight env PATH -- echo arg1 arg2 arg3
		cmd := exec.Command(binaryPath, "env", "PATH", "--", "echo", "arg1", "arg2", "arg3") //nolint:gosec // intentional: testing exec mode
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command failed: %v\n%s", err, output)
		}

		// Should see all arguments passed through
		outputStr := string(output)
		if !strings.Contains(outputStr, "arg1 arg2 arg3") {
			t.Errorf("expected arguments to be passed through, got: %s", outputStr)
		}
	})

	t.Run("exec command not found", func(t *testing.T) {
		// Run: preflight env PATH -- nonexistent-command-xyz-12345
		cmd := exec.Command(binaryPath, "env", "PATH", "--", "nonexistent-command-xyz-12345") //nolint:gosec // intentional: testing exec mode
		cmd.Env = append(os.Environ(), "NO_COLOR=1")
		output, err := cmd.CombinedOutput()

		// Command should fail because exec'd command doesn't exist
		if err == nil {
			t.Fatal("expected command to fail")
		}

		// Should see the check pass but exec fail
		outputStr := string(output)
		if !strings.Contains(outputStr, "[OK] env: PATH") {
			t.Errorf("expected check to pass, got: %s", outputStr)
		}
		if !strings.Contains(outputStr, "exec:") {
			t.Errorf("expected exec error message, got: %s", outputStr)
		}
	})
}
