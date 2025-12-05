package main

import (
	"testing"
)

func TestRunGitCheck_RequiresFlag(t *testing.T) {
	// Reset all flags to defaults
	gitClean = false
	gitNoUncommitted = false
	gitNoUntracked = false
	gitBranch = ""
	gitTagMatch = ""

	err := runGitCheck(nil, nil)
	if err == nil {
		t.Error("expected error when no flags provided")
	}

	expectedPrefix := "at least one of"
	if err != nil && len(err.Error()) >= len(expectedPrefix) && err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("error = %q, want prefix %q", err.Error(), expectedPrefix)
	}
}
