package gitcheck

import (
	"errors"
	"os"
	"testing"
)

// TestRealGitRunner tests the actual git commands.
// These tests run against the real git repo (preflight itself).
func TestRealGitRunner_IsGitRepo(t *testing.T) {
	runner := &RealGitRunner{}

	isRepo, err := runner.IsGitRepo()
	if err != nil {
		t.Fatalf("IsGitRepo() error = %v", err)
	}
	if !isRepo {
		t.Skip("not running in a git repository")
	}
}

func TestRealGitRunner_Status(t *testing.T) {
	runner := &RealGitRunner{}

	// Just verify it doesn't error - actual content varies
	_, err := runner.Status()
	if err != nil {
		t.Errorf("Status() error = %v", err)
	}
}

func TestRealGitRunner_CurrentBranch(t *testing.T) {
	runner := &RealGitRunner{}

	branch, err := runner.CurrentBranch()
	if err != nil {
		t.Errorf("CurrentBranch() error = %v", err)
	}
	if branch == "" {
		t.Error("CurrentBranch() returned empty string")
	}
}

func TestRealGitRunner_TagsAtHead(t *testing.T) {
	runner := &RealGitRunner{}

	// May return empty slice, just verify no error
	_, err := runner.TagsAtHead()
	if err != nil {
		t.Errorf("TagsAtHead() error = %v", err)
	}
}

func TestRealGitRunner_IsGitRepo_NotRepo(t *testing.T) {
	// Change to a non-git directory
	tmpDir, err := os.MkdirTemp("", "preflight-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	runner := &RealGitRunner{}
	isRepo, err := runner.IsGitRepo()
	if err != nil {
		t.Fatalf("IsGitRepo() error = %v", err)
	}
	if isRepo {
		t.Error("IsGitRepo() = true in non-git directory")
	}
}

type mockGitRunner struct {
	IsGitRepoFunc     func() (bool, error)
	StatusFunc        func() (string, error)
	CurrentBranchFunc func() (string, error)
	TagsAtHeadFunc    func() ([]string, error)
}

func (m *mockGitRunner) IsGitRepo() (bool, error) {
	return m.IsGitRepoFunc()
}

func (m *mockGitRunner) Status() (string, error) {
	return m.StatusFunc()
}

func (m *mockGitRunner) CurrentBranch() (string, error) {
	return m.CurrentBranchFunc()
}

func (m *mockGitRunner) TagsAtHead() ([]string, error) {
	return m.TagsAtHeadFunc()
}

func TestMockGitRunner_IsGitRepo(t *testing.T) {
	mock := &mockGitRunner{
		IsGitRepoFunc: func() (bool, error) {
			return true, nil
		},
	}

	isRepo, err := mock.IsGitRepo()
	if err != nil {
		t.Fatalf("IsGitRepo() error = %v", err)
	}
	if !isRepo {
		t.Error("IsGitRepo() = false, want true")
	}

	mock.IsGitRepoFunc = func() (bool, error) {
		return false, errors.New("git not found")
	}

	_, err = mock.IsGitRepo()
	if err == nil {
		t.Error("IsGitRepo() error = nil, want error")
	}
}

func TestMockGitRunner_Status(t *testing.T) {
	mock := &mockGitRunner{
		StatusFunc: func() (string, error) {
			return " M modified.go\n?? untracked.txt", nil
		},
	}

	status, err := mock.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status == "" {
		t.Error("Status() returned empty string")
	}
}

func TestMockGitRunner_CurrentBranch(t *testing.T) {
	mock := &mockGitRunner{
		CurrentBranchFunc: func() (string, error) {
			return "main", nil
		},
	}

	branch, err := mock.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if branch != "main" {
		t.Errorf("CurrentBranch() = %q, want %q", branch, "main")
	}
}

func TestMockGitRunner_TagsAtHead(t *testing.T) {
	mock := &mockGitRunner{
		TagsAtHeadFunc: func() ([]string, error) {
			return []string{"v1.0.0", "release-1"}, nil
		},
	}

	tags, err := mock.TagsAtHead()
	if err != nil {
		t.Fatalf("TagsAtHead() error = %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("TagsAtHead() returned %d tags, want 2", len(tags))
	}
}
