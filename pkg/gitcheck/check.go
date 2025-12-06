package gitcheck

import (
	"errors"
	"path"
	"strings"

	"github.com/vertti/preflight/pkg/check"
)

// Check verifies git repository state.
type Check struct {
	Clean         bool   // --clean: no uncommitted or untracked changes
	NoUncommitted bool   // --no-uncommitted: no staged/modified files
	NoUntracked   bool   // --no-untracked: no untracked files
	Branch        string // --branch: must be on this branch
	TagMatch      string // --tag-match: HEAD must have tag matching glob
	Runner        GitRunner
}

// Run executes the git check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: "git",
	}

	// First verify we're in a git repository
	isRepo, err := c.Runner.IsGitRepo()
	if err != nil {
		return result.Failf("failed to check git repository: %v", err)
	}
	if !isRepo {
		return result.Failf("not a git repository")
	}

	// Expand --clean to its component checks
	noUncommitted := c.NoUncommitted || c.Clean
	noUntracked := c.NoUntracked || c.Clean

	// Check for uncommitted/untracked changes
	if noUncommitted || noUntracked {
		if err := c.checkStatus(noUncommitted, noUntracked, &result); err != nil {
			return result
		}
	}

	// Check branch
	if c.Branch != "" {
		if err := c.checkBranch(&result); err != nil {
			return result
		}
	}

	// Check tag match
	if c.TagMatch != "" {
		if err := c.checkTagMatch(&result); err != nil {
			return result
		}
	}

	result.Status = check.StatusOK
	return result
}

func (c *Check) checkStatus(noUncommitted, noUntracked bool, result *check.Result) error {
	status, err := c.Runner.Status()
	if err != nil {
		result.Failf("failed to get git status: %v", err)
		return err
	}

	if status == "" {
		result.AddDetail("working directory clean")
		return nil
	}

	// Parse porcelain output
	lines := strings.Split(status, "\n")
	var uncommitted, untracked []string

	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		// Porcelain format: XY PATH
		// X = index status, Y = worktree status
		// '??' = untracked, anything else = staged/modified
		// Path starts after XY, trim leading space(s)
		if strings.HasPrefix(line, "??") {
			untracked = append(untracked, strings.TrimSpace(line[2:]))
		} else {
			uncommitted = append(uncommitted, strings.TrimSpace(line[2:]))
		}
	}

	if noUncommitted && len(uncommitted) > 0 {
		result.AddDetailf("uncommitted: %d file(s)", len(uncommitted))
		for _, f := range uncommitted {
			result.AddDetailf("  %s", f)
		}
		result.Failf("found %d uncommitted file(s)", len(uncommitted))
		return errors.New("uncommitted files")
	}

	if noUntracked && len(untracked) > 0 {
		result.AddDetailf("untracked: %d file(s)", len(untracked))
		for _, f := range untracked {
			result.AddDetailf("  %s", f)
		}
		result.Failf("found %d untracked file(s)", len(untracked))
		return errors.New("untracked files")
	}

	result.AddDetail("working directory clean")
	return nil
}

func (c *Check) checkBranch(result *check.Result) error {
	branch, err := c.Runner.CurrentBranch()
	if err != nil {
		result.Failf("failed to get current branch: %v", err)
		return err
	}

	result.AddDetailf("branch: %s", branch)

	if branch != c.Branch {
		result.Failf("on branch %q, expected %q", branch, c.Branch)
		return errors.New("wrong branch")
	}

	return nil
}

func (c *Check) checkTagMatch(result *check.Result) error {
	tags, err := c.Runner.TagsAtHead()
	if err != nil {
		result.Failf("failed to get tags: %v", err)
		return err
	}

	if len(tags) == 0 {
		result.Failf("no tags at HEAD, expected match for %q", c.TagMatch)
		return errors.New("no tags")
	}

	// Check if any tag matches the glob pattern
	for _, tag := range tags {
		matched, err := path.Match(c.TagMatch, tag)
		if err != nil {
			result.Failf("invalid tag pattern %q: %v", c.TagMatch, err)
			return err
		}
		if matched {
			result.AddDetailf("tag: %s (matches %q)", tag, c.TagMatch)
			return nil
		}
	}

	result.AddDetailf("tags at HEAD: %s", strings.Join(tags, ", "))
	result.Failf("no tag matches pattern %q", c.TagMatch)
	return errors.New("no matching tag")
}
