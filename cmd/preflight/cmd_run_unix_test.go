//go:build unix

package main

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCommands distinguishes "the child failed" from "the child could not be
// started" by testing for *exec.ExitError, so spawn has to produce that
// distinction faithfully. Unix-tagged because it needs a real shell.
func TestSpawn(t *testing.T) {
	t.Run("a successful command returns nil", func(t *testing.T) {
		require.NoError(t, spawn("/bin/sh", []string{"-c", "exit 0"}))
	})

	t.Run("a failing command returns its exit code", func(t *testing.T) {
		err := spawn("/bin/sh", []string{"-c", "exit 42"})

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr, "runCommands relies on this to propagate the code")
		assert.Equal(t, 42, exitErr.ExitCode())
	})

	t.Run("a command that cannot start is not an exit error", func(t *testing.T) {
		err := spawn("/nonexistent/binary", nil)

		var exitErr *exec.ExitError
		require.Error(t, err)
		assert.NotErrorAs(t, err, &exitErr, "must surface as an error, not as exit code 0")
	})
}
