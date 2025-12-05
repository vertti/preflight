package gitcheck

import (
	"bytes"
	"os/exec"
	"strings"
)

// GitRunner abstracts git command execution for testability.
type GitRunner interface {
	// IsGitRepo returns true if the current directory is inside a git repository.
	IsGitRepo() (bool, error)

	// Status returns the output of 'git status --porcelain'.
	// Lines starting with '??' are untracked, others are staged/modified.
	Status() (string, error)

	// CurrentBranch returns the name of the current branch.
	// Returns "HEAD" if in detached HEAD state.
	CurrentBranch() (string, error)

	// TagsAtHead returns all tags pointing at the current HEAD commit.
	TagsAtHead() ([]string, error)
}

// RealGitRunner executes actual git commands.
type RealGitRunner struct{}

func (r *RealGitRunner) IsGitRepo() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	if err != nil {
		// Exit code 128 = not a git repository
		return false, nil
	}
	return true, nil
}

func (r *RealGitRunner) Status() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (r *RealGitRunner) CurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (r *RealGitRunner) TagsAtHead() ([]string, error) {
	cmd := exec.Command("git", "tag", "--points-at", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	output := strings.TrimSpace(out.String())
	if output == "" {
		return nil, nil
	}
	return strings.Split(output, "\n"), nil
}
