package resourcecheck

import (
	"os"
	"testing"
)

func TestRealResourceChecker_FreeDiskSpace(t *testing.T) {
	r := &RealResourceChecker{}

	// Test current directory (should always work)
	space, err := r.FreeDiskSpace(".")
	if err != nil {
		t.Errorf("FreeDiskSpace(\".\") error = %v", err)
	}
	if space == 0 {
		t.Error("FreeDiskSpace(\".\") = 0, expected > 0")
	}

	// Test non-existent path
	_, err = r.FreeDiskSpace("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("FreeDiskSpace(nonexistent) error = nil, expected error")
	}
}

func TestRealResourceChecker_FreeDiskSpace_TempDir(t *testing.T) {
	r := &RealResourceChecker{}

	tmpDir, err := os.MkdirTemp("", "preflight-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	space, err := r.FreeDiskSpace(tmpDir)
	if err != nil {
		t.Errorf("FreeDiskSpace(tmpDir) error = %v", err)
	}
	if space == 0 {
		t.Error("FreeDiskSpace(tmpDir) = 0, expected > 0")
	}
}

func TestRealResourceChecker_AvailableMemory(t *testing.T) {
	r := &RealResourceChecker{}

	mem, err := r.AvailableMemory()
	if err != nil {
		t.Errorf("AvailableMemory() error = %v", err)
	}
	if mem == 0 {
		t.Error("AvailableMemory() = 0, expected > 0")
	}

	// Should be at least 100MB (reasonable minimum for any system)
	if mem < 100*MB {
		t.Errorf("AvailableMemory() = %d, expected at least 100MB", mem)
	}
}

func TestRealResourceChecker_NumCPUs(t *testing.T) {
	r := &RealResourceChecker{}

	cpus := r.NumCPUs()
	if cpus < 1 {
		t.Errorf("NumCPUs() = %d, expected >= 1", cpus)
	}
}

func TestReadCgroupMemoryLimit(t *testing.T) {
	// Test with non-existent file (common case on non-Linux or non-container)
	_, err := readCgroupMemoryLimit("/nonexistent/path")
	if err == nil {
		t.Error("readCgroupMemoryLimit(nonexistent) error = nil, expected error")
	}
}

// Mock implementation for unit testing Check
type mockResourceChecker struct {
	freeDiskSpace   uint64
	freeDiskErr     error
	availableMemory uint64
	availableMemErr error
	numCPUs         int
}

func (m *mockResourceChecker) FreeDiskSpace(path string) (uint64, error) {
	return m.freeDiskSpace, m.freeDiskErr
}

func (m *mockResourceChecker) AvailableMemory() (uint64, error) {
	return m.availableMemory, m.availableMemErr
}

func (m *mockResourceChecker) NumCPUs() int {
	return m.numCPUs
}
