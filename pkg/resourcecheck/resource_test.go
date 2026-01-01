//go:build unix

package resourcecheck

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealResourceChecker_FreeDiskSpace(t *testing.T) {
	r := &RealResourceChecker{}

	space, err := r.FreeDiskSpace(".")
	require.NoError(t, err)
	assert.Positive(t, space)

	_, err = r.FreeDiskSpace("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}

func TestRealResourceChecker_FreeDiskSpace_TempDir(t *testing.T) {
	r := &RealResourceChecker{}

	tmpDir := t.TempDir()
	space, err := r.FreeDiskSpace(tmpDir)
	require.NoError(t, err)
	assert.Positive(t, space)
}

func TestRealResourceChecker_AvailableMemory(t *testing.T) {
	r := &RealResourceChecker{}

	mem, err := r.AvailableMemory()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, mem, 100*MB, "expected at least 100MB")
}

func TestRealResourceChecker_NumCPUs(t *testing.T) {
	r := &RealResourceChecker{}
	assert.GreaterOrEqual(t, r.NumCPUs(), 1)
}

func TestReadCgroupMemoryLimit(t *testing.T) {
	_, err := readCgroupMemoryLimit("/nonexistent/path")
	require.Error(t, err)

	t.Run("numeric value", func(t *testing.T) {
		tmpFile := writeTempFile(t, "8589934592")
		mem, err := readCgroupMemoryLimit(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, uint64(8589934592), mem)
	})

	t.Run("max value", func(t *testing.T) {
		tmpFile := writeTempFile(t, "max")
		mem, err := readCgroupMemoryLimit(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), mem)
	})

	t.Run("value with newline", func(t *testing.T) {
		tmpFile := writeTempFile(t, "4294967296\n")
		mem, err := readCgroupMemoryLimit(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, uint64(4294967296), mem)
	})
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "memory_limit")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	return tmpFile.Name()
}

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
