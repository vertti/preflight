package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The loop that executes .preflight lines was untestable in-process because it
// called os.Exit directly, so the substitution that keeps a line from naming
// some other binary had no test at all.
func TestRunCommands(t *testing.T) {
	const exe = "/path/to/preflight"

	t.Run("always runs the preflight binary, never a path from the file", func(t *testing.T) {
		var gotName string
		var gotArgs []string
		_, err := runCommands([]string{"preflight env HOME"}, exe, func(name string, args []string) error {
			gotName, gotArgs = name, args
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, exe, gotName)
		assert.Equal(t, []string{"env", "HOME"}, gotArgs)
	})

	t.Run("substitutes even when the first token is a path", func(t *testing.T) {
		// ParseFile should never produce this, so the assertion is that the
		// second layer holds independently of the first.
		var gotName string
		_, err := runCommands([]string{"preflight/../evil.sh --pwn"}, exe, func(name string, _ []string) error {
			gotName = name
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, exe, gotName, "a path in the file must not become the program")
	})

	t.Run("propagates a child's exit code", func(t *testing.T) {
		failing := exec.Command("sh", "-c", "exit 42")
		exitErr := failing.Run()
		require.Error(t, exitErr)

		code, err := runCommands([]string{"preflight env NOPE"}, exe, func(string, []string) error {
			return exitErr
		})
		require.NoError(t, err)
		assert.Equal(t, 42, code)
	})

	t.Run("a non-exit failure is an error, not an exit code", func(t *testing.T) {
		code, err := runCommands([]string{"preflight env HOME"}, exe, func(string, []string) error {
			return errors.New("fork failed")
		})
		require.Error(t, err)
		assert.Equal(t, 0, code)
		assert.Contains(t, err.Error(), "preflight env HOME")
	})

	t.Run("stops at the first failing command", func(t *testing.T) {
		calls := 0
		failing := exec.Command("sh", "-c", "exit 3")
		exitErr := failing.Run()

		code, err := runCommands([]string{"preflight env A", "preflight env B", "preflight env C"}, exe,
			func(string, []string) error {
				calls++
				if calls == 2 {
					return exitErr
				}
				return nil
			})
		require.NoError(t, err)
		assert.Equal(t, 3, code)
		assert.Equal(t, 2, calls, "must not keep running after a failure")
	})

	t.Run("blank commands are skipped without running anything", func(t *testing.T) {
		calls := 0
		code, err := runCommands([]string{"", "   "}, exe, func(string, []string) error {
			calls++
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, 0, code)
		assert.Equal(t, 0, calls)
	})

	t.Run("records that a check ran, but not for an empty file", func(t *testing.T) {
		original := checkRan
		defer func() { checkRan = original }()

		checkRan = false
		_, err := runCommands(nil, exe, func(string, []string) error { return nil })
		require.NoError(t, err)
		assert.False(t, checkRan, "an empty .preflight ran no check, so exec must still be refused")

		checkRan = false
		_, err = runCommands([]string{"preflight env HOME"}, exe, func(string, []string) error { return nil })
		require.NoError(t, err)
		assert.True(t, checkRan)
	})
}

// Reaches the wiring in runRun that the "nonexistent file" case returns before:
// discovery, parsing, resolving the executable, and the zero-exit path. An
// empty file runs no commands, so nothing is spawned.
func TestRunRun_EmptyFileIsASuccessfulNoOp(t *testing.T) {
	originalFile, originalRan := runFile, checkRan
	defer func() { runFile, checkRan = originalFile, originalRan }()
	runFile, checkRan = "", false

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".preflight"), []byte("# only a comment\n\n"), 0o600))
	t.Chdir(dir)

	require.NoError(t, runRun(nil, nil))
	assert.False(t, checkRan, "no check ran, so exec mode must still refuse")
}

func TestRunRun_ReportsAMissingFile(t *testing.T) {
	originalFile := runFile
	defer func() { runFile = originalFile }()
	runFile = filepath.Join(t.TempDir(), "absent")

	require.Error(t, runRun(nil, nil))
}
