package gitcheck

import (
	"errors"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

func TestGitCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
	}{
		// Repository checks
		{
			name: "not a git repository",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return false, nil },
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "git repo check fails",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) {
						return false, errors.New("git not found")
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// --clean tests
		{
			name: "clean passes on clean repo",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc:    func() (string, error) { return "", nil },
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "clean fails with uncommitted changes",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return " M modified.go", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "clean fails with staged changes",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return "A  staged.go", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "clean fails with untracked files",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return "?? newfile.txt", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// --no-uncommitted tests
		{
			name: "no-uncommitted passes with clean repo",
			check: Check{
				NoUncommitted: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc:    func() (string, error) { return "", nil },
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "no-uncommitted passes with only untracked",
			check: Check{
				NoUncommitted: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return "?? untracked.txt", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "no-uncommitted fails with staged file",
			check: Check{
				NoUncommitted: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return "A  staged.go", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "no-uncommitted fails with modified file",
			check: Check{
				NoUncommitted: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return " M modified.go", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// --no-untracked tests
		{
			name: "no-untracked passes with clean repo",
			check: Check{
				NoUntracked: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc:    func() (string, error) { return "", nil },
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "no-untracked passes with only uncommitted",
			check: Check{
				NoUntracked: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return " M modified.go", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "no-untracked fails with untracked files",
			check: Check{
				NoUntracked: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return "?? newfile.txt", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// --branch tests
		{
			name: "branch passes on correct branch",
			check: Check{
				Branch: "main",
				Runner: &mockGitRunner{
					IsGitRepoFunc:     func() (bool, error) { return true, nil },
					CurrentBranchFunc: func() (string, error) { return "main", nil },
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "branch fails on wrong branch",
			check: Check{
				Branch: "main",
				Runner: &mockGitRunner{
					IsGitRepoFunc:     func() (bool, error) { return true, nil },
					CurrentBranchFunc: func() (string, error) { return "feature", nil },
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "branch fails in detached HEAD",
			check: Check{
				Branch: "main",
				Runner: &mockGitRunner{
					IsGitRepoFunc:     func() (bool, error) { return true, nil },
					CurrentBranchFunc: func() (string, error) { return "HEAD", nil },
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "branch check error",
			check: Check{
				Branch: "main",
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					CurrentBranchFunc: func() (string, error) {
						return "", errors.New("git error")
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// --tag-match tests
		{
			name: "tag-match passes with matching tag",
			check: Check{
				TagMatch: "v*",
				Runner: &mockGitRunner{
					IsGitRepoFunc:  func() (bool, error) { return true, nil },
					TagsAtHeadFunc: func() ([]string, error) { return []string{"v1.0.0"}, nil },
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "tag-match passes with multiple tags one matching",
			check: Check{
				TagMatch: "release-*",
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					TagsAtHeadFunc: func() ([]string, error) {
						return []string{"v1.0.0", "release-2024-01"}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "tag-match fails with no tags",
			check: Check{
				TagMatch: "v*",
				Runner: &mockGitRunner{
					IsGitRepoFunc:  func() (bool, error) { return true, nil },
					TagsAtHeadFunc: func() ([]string, error) { return nil, nil },
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "tag-match fails with non-matching tag",
			check: Check{
				TagMatch: "v*",
				Runner: &mockGitRunner{
					IsGitRepoFunc:  func() (bool, error) { return true, nil },
					TagsAtHeadFunc: func() ([]string, error) { return []string{"release-1.0"}, nil },
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "tag-match invalid pattern",
			check: Check{
				TagMatch: "[invalid",
				Runner: &mockGitRunner{
					IsGitRepoFunc:  func() (bool, error) { return true, nil },
					TagsAtHeadFunc: func() ([]string, error) { return []string{"v1.0.0"}, nil },
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "tag-match error getting tags",
			check: Check{
				TagMatch: "v*",
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					TagsAtHeadFunc: func() ([]string, error) {
						return nil, errors.New("git error")
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// Combined flags
		{
			name: "clean and branch both pass",
			check: Check{
				Clean:  true,
				Branch: "main",
				Runner: &mockGitRunner{
					IsGitRepoFunc:     func() (bool, error) { return true, nil },
					StatusFunc:        func() (string, error) { return "", nil },
					CurrentBranchFunc: func() (string, error) { return "main", nil },
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "clean passes but branch fails",
			check: Check{
				Clean:  true,
				Branch: "main",
				Runner: &mockGitRunner{
					IsGitRepoFunc:     func() (bool, error) { return true, nil },
					StatusFunc:        func() (string, error) { return "", nil },
					CurrentBranchFunc: func() (string, error) { return "develop", nil },
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "branch passes but clean fails",
			check: Check{
				Clean:  true,
				Branch: "main",
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return "?? untracked.txt", nil
					},
					CurrentBranchFunc: func() (string, error) { return "main", nil },
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "all checks pass together",
			check: Check{
				Clean:    true,
				Branch:   "main",
				TagMatch: "v*",
				Runner: &mockGitRunner{
					IsGitRepoFunc:     func() (bool, error) { return true, nil },
					StatusFunc:        func() (string, error) { return "", nil },
					CurrentBranchFunc: func() (string, error) { return "main", nil },
					TagsAtHeadFunc:    func() ([]string, error) { return []string{"v1.0.0"}, nil },
				},
			},
			wantStatus: check.StatusOK,
		},

		// Status command error
		{
			name: "status command fails",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return "", errors.New("git status failed")
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// Multiple untracked/uncommitted files
		{
			name: "clean fails with multiple uncommitted files",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return " M file1.go\n M file2.go\nA  file3.go", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "clean fails with mixed changes",
			check: Check{
				Clean: true,
				Runner: &mockGitRunner{
					IsGitRepoFunc: func() (bool, error) { return true, nil },
					StatusFunc: func() (string, error) {
						return " M modified.go\n?? untracked.txt", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v (details: %v)", result.Status, tt.wantStatus, result.Details)
			}

			if result.Name != "git" {
				t.Errorf("name = %q, want %q", result.Name, "git")
			}
		})
	}
}
