package gitcheck

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealGitRunner_IsGitRepo(t *testing.T) {
	runner := &RealGitRunner{}
	isRepo, err := runner.IsGitRepo()
	require.NoError(t, err)
	if !isRepo {
		t.Skip("not running in a git repository")
	}
}

func TestRealGitRunner_Status(t *testing.T) {
	runner := &RealGitRunner{}
	_, err := runner.Status()
	assert.NoError(t, err)
}

func TestRealGitRunner_CurrentBranch(t *testing.T) {
	runner := &RealGitRunner{}
	branch, err := runner.CurrentBranch()
	require.NoError(t, err)
	assert.NotEmpty(t, branch)
}

func TestRealGitRunner_TagsAtHead(t *testing.T) {
	runner := &RealGitRunner{}
	_, err := runner.TagsAtHead()
	assert.NoError(t, err)
}

// git runs a command with the developer's global and system config ignored.
// Inheriting them made these tests fail on real machines: commit.gpgsign=true
// aborts the commit with exit 128, and a global core.hooksPath with a failing
// pre-commit hook does the same.
func git(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec // args are literals from this test file
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

func TestRealGitRunner_IsGitRepo_NotRepo(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	require.NoError(t, os.Chdir(tmpDir))

	runner := &RealGitRunner{}
	isRepo, err := runner.IsGitRepo()
	require.NoError(t, err)
	assert.False(t, isRepo)
}

func TestRealGitRunner_TagsAtHead_WithTag(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	require.NoError(t, os.Chdir(tmpDir))

	// Initialize git repo with a commit and tag
	git(t, "init")
	git(t, "config", "user.email", "test@test.com")
	git(t, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
	git(t, "add", "test.txt")
	git(t, "commit", "-m", "initial")
	git(t, "tag", "v1.0.0")

	runner := &RealGitRunner{}
	tags, err := runner.TagsAtHead()
	require.NoError(t, err)
	assert.Equal(t, []string{"v1.0.0"}, tags)
}

type mockGitRunner struct {
	IsGitRepoFunc     func() (bool, error)
	StatusFunc        func() (string, error)
	CurrentBranchFunc func() (string, error)
	TagsAtHeadFunc    func() ([]string, error)
}

func (m *mockGitRunner) IsGitRepo() (bool, error)       { return m.IsGitRepoFunc() }
func (m *mockGitRunner) Status() (string, error)        { return m.StatusFunc() }
func (m *mockGitRunner) CurrentBranch() (string, error) { return m.CurrentBranchFunc() }
func (m *mockGitRunner) TagsAtHead() ([]string, error)  { return m.TagsAtHeadFunc() }
