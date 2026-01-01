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

func TestRealGitRunner_IsGitRepo_NotRepo(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()

	require.NoError(t, os.Chdir(tmpDir))

	runner := &RealGitRunner{}
	isRepo, err := runner.IsGitRepo()
	require.NoError(t, err)
	assert.False(t, isRepo)
}

func TestRealGitRunner_TagsAtHead_WithTag(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()

	require.NoError(t, os.Chdir(tmpDir))

	// Initialize git repo with a commit and tag
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@test.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test").Run())
	require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
	require.NoError(t, exec.Command("git", "add", "test.txt").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "initial").Run())
	require.NoError(t, exec.Command("git", "tag", "v1.0.0").Run())

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
