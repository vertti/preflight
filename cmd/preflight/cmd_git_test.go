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

	expectedMsg := "at least one check flag required"
	if err != nil && err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("error = %q, want prefix %q", err.Error(), expectedMsg)
	}
}
