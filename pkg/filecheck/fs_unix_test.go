//go:build unix

package filecheck

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A file whose owner class denies write but whose group/other bits allow it is
// the case permission-bit inspection gets wrong: mode&0o222 is non-zero, yet
// POSIX applies the owner class and denies the owner. Root bypasses this, so
// the test only means something as an unprivileged user.
func TestRealFileSystem_CanAccessUsesEffectivePermissions(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}

	path := filepath.Join(t.TempDir(), "owner-denied")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
	require.NoError(t, os.Chmod(path, 0o022)) //nolint:gosec // the permissive mode is the fixture: group/other may write, owner may not

	rfs := &RealFileSystem{}

	writable, err := rfs.CanAccess(path, AccessWrite)
	require.NoError(t, err)
	assert.False(t, writable, "owner class denies write, so the owner cannot write")
	assert.NotZero(t, os.FileMode(0o022)&0o222, "precondition: a mode-bit check would say writable")

	require.NoError(t, os.Chmod(path, 0o011)) //nolint:gosec // as above, for the execute bit
	executable, err := rfs.CanAccess(path, AccessExecute)
	require.NoError(t, err)
	assert.False(t, executable, "owner class denies execute, so the owner cannot execute")
}

func TestRealFileSystem_CanAccessAllowsWhatOwnerMay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owner-allowed")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))

	rfs := &RealFileSystem{}

	writable, err := rfs.CanAccess(path, AccessWrite)
	require.NoError(t, err)
	assert.True(t, writable)

	require.NoError(t, os.Chmod(path, 0o744)) //nolint:gosec // an owner-executable file is the fixture
	executable, err := rfs.CanAccess(path, AccessExecute)
	require.NoError(t, err)
	assert.True(t, executable)
}

func TestRealFileSystem_CanAccessMissingPath(t *testing.T) {
	rfs := &RealFileSystem{}
	_, err := rfs.CanAccess(filepath.Join(t.TempDir(), "nope"), AccessWrite)
	assert.Error(t, err, "a missing path is an error, not a denial")
}
