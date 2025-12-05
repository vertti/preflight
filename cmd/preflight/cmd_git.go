package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/gitcheck"
)

var (
	gitClean         bool
	gitNoUncommitted bool
	gitNoUntracked   bool
	gitBranch        string
	gitTagMatch      string
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Check git repository state (clean, branch, tags)",
	Long: `Check git repository state.

Examples:
  preflight git --clean                    # No uncommitted or untracked files
  preflight git --no-uncommitted           # Allow untracked, forbid uncommitted
  preflight git --branch main              # Must be on 'main' branch
  preflight git --tag-match "v*"           # HEAD must have tag starting with 'v'
  preflight git --clean --branch release   # Combined checks`,
	RunE: runGitCheck,
}

func init() {
	gitCmd.Flags().BoolVar(&gitClean, "clean", false,
		"working directory must be clean (no uncommitted or untracked)")
	gitCmd.Flags().BoolVar(&gitNoUncommitted, "no-uncommitted", false,
		"no uncommitted (staged/modified) changes allowed")
	gitCmd.Flags().BoolVar(&gitNoUntracked, "no-untracked", false,
		"no untracked files allowed")
	gitCmd.Flags().StringVar(&gitBranch, "branch", "",
		"must be on specified branch")
	gitCmd.Flags().StringVar(&gitTagMatch, "tag-match", "",
		"HEAD must have tag matching glob pattern (e.g., 'v*')")
	rootCmd.AddCommand(gitCmd)
}

func runGitCheck(_ *cobra.Command, _ []string) error {
	// Require at least one check flag
	if !gitClean && !gitNoUncommitted && !gitNoUntracked &&
		gitBranch == "" && gitTagMatch == "" {
		return fmt.Errorf("at least one check flag required: --clean, --no-uncommitted, --no-untracked, --branch, or --tag-match")
	}

	c := &gitcheck.Check{
		Clean:         gitClean,
		NoUncommitted: gitNoUncommitted,
		NoUntracked:   gitNoUntracked,
		Branch:        gitBranch,
		TagMatch:      gitTagMatch,
		Runner:        &gitcheck.RealGitRunner{},
	}

	return runCheck(c)
}
