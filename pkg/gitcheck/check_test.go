package gitcheck

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
)

func TestGitCheck_Run(t *testing.T) {
	gitRepo := func() (bool, error) { return true, nil }
	gitNotRepo := func() (bool, error) { return false, nil }
	gitRepoErr := func() (bool, error) { return false, errors.New("git not found") }
	clean := func() (string, error) { return "", nil }
	modified := func() (string, error) { return " M modified.go", nil }
	staged := func() (string, error) { return "A  staged.go", nil }
	untracked := func() (string, error) { return "?? newfile.txt", nil }
	mixed := func() (string, error) { return " M modified.go\n?? untracked.txt", nil }
	statusErr := func() (string, error) { return "", errors.New("git status failed") }
	branchMain := func() (string, error) { return "main", nil }
	branchFeature := func() (string, error) { return "feature", nil }
	branchHead := func() (string, error) { return "HEAD", nil }
	branchErr := func() (string, error) { return "", errors.New("git error") }
	tagV1 := func() ([]string, error) { return []string{"v1.0.0"}, nil }
	tagMultiple := func() ([]string, error) { return []string{"v1.0.0", "release-2024-01"}, nil }
	tagNone := func() ([]string, error) { return nil, nil }
	tagNoMatch := func() ([]string, error) { return []string{"release-1.0"}, nil }
	tagErr := func() ([]string, error) { return nil, errors.New("git error") }

	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
	}{
		// Repository checks
		{"not a git repository", Check{Clean: true, Runner: &mockGitRunner{IsGitRepoFunc: gitNotRepo}}, check.StatusFail},
		{"git repo check fails", Check{Clean: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepoErr}}, check.StatusFail},

		// --clean
		{"clean on clean repo", Check{Clean: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: clean}}, check.StatusOK},
		{"clean with uncommitted", Check{Clean: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: modified}}, check.StatusFail},
		{"clean with staged", Check{Clean: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: staged}}, check.StatusFail},
		{"clean with untracked", Check{Clean: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: untracked}}, check.StatusFail},
		{"clean with mixed changes", Check{Clean: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: mixed}}, check.StatusFail},
		{"clean status error", Check{Clean: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: statusErr}}, check.StatusFail},

		// --no-uncommitted
		{"no-uncommitted on clean", Check{NoUncommitted: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: clean}}, check.StatusOK},
		{"no-uncommitted with only untracked", Check{NoUncommitted: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: untracked}}, check.StatusOK},
		{"no-uncommitted with staged", Check{NoUncommitted: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: staged}}, check.StatusFail},
		{"no-uncommitted with modified", Check{NoUncommitted: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: modified}}, check.StatusFail},

		// --no-untracked
		{"no-untracked on clean", Check{NoUntracked: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: clean}}, check.StatusOK},
		{"no-untracked with only uncommitted", Check{NoUntracked: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: modified}}, check.StatusOK},
		{"no-untracked with untracked", Check{NoUntracked: true, Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: untracked}}, check.StatusFail},

		// --branch
		{"branch correct", Check{Branch: "main", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, CurrentBranchFunc: branchMain}}, check.StatusOK},
		{"branch wrong", Check{Branch: "main", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, CurrentBranchFunc: branchFeature}}, check.StatusFail},
		{"branch detached HEAD", Check{Branch: "main", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, CurrentBranchFunc: branchHead}}, check.StatusFail},
		{"branch error", Check{Branch: "main", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, CurrentBranchFunc: branchErr}}, check.StatusFail},

		// --tag-match
		{"tag-match matching", Check{TagMatch: "v*", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, TagsAtHeadFunc: tagV1}}, check.StatusOK},
		{"tag-match multiple one matching", Check{TagMatch: "release-*", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, TagsAtHeadFunc: tagMultiple}}, check.StatusOK},
		{"tag-match no tags", Check{TagMatch: "v*", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, TagsAtHeadFunc: tagNone}}, check.StatusFail},
		{"tag-match non-matching", Check{TagMatch: "v*", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, TagsAtHeadFunc: tagNoMatch}}, check.StatusFail},
		{"tag-match invalid pattern", Check{TagMatch: "[invalid", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, TagsAtHeadFunc: tagV1}}, check.StatusFail},
		{"tag-match error", Check{TagMatch: "v*", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, TagsAtHeadFunc: tagErr}}, check.StatusFail},

		// Combined flags
		{"clean and branch both pass", Check{Clean: true, Branch: "main", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: clean, CurrentBranchFunc: branchMain}}, check.StatusOK},
		{"clean passes branch fails", Check{Clean: true, Branch: "main", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: clean, CurrentBranchFunc: branchFeature}}, check.StatusFail},
		{"branch passes clean fails", Check{Clean: true, Branch: "main", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: untracked, CurrentBranchFunc: branchMain}}, check.StatusFail},
		{"all checks pass", Check{Clean: true, Branch: "main", TagMatch: "v*", Runner: &mockGitRunner{IsGitRepoFunc: gitRepo, StatusFunc: clean, CurrentBranchFunc: branchMain, TagsAtHeadFunc: tagV1}}, check.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			assert.Equal(t, "git", result.Name)
		})
	}
}
