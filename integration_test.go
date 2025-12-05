package preflight_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/cmdcheck"
	"github.com/vertti/preflight/pkg/envcheck"
	"github.com/vertti/preflight/pkg/filecheck"
	"github.com/vertti/preflight/pkg/gitcheck"
	"github.com/vertti/preflight/pkg/hashcheck"
	"github.com/vertti/preflight/pkg/httpcheck"
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
		FS:       &filecheck.RealFileSystem{},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK (details: %v)", result.Status, result.Details)
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
