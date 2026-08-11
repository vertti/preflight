//go:build unix

package cmdcheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// A command that exits while a background child still holds the stdout pipe.
// os/exec waits for the pipe to reach EOF, and CommandContext kills only the
// direct child, so without WaitDelay the context deadline is unenforceable and
// RunCommandContext blocks for as long as the grandchild lives.
func TestRunCommandContext_ReturnsWhenGrandchildHoldsThePipe(t *testing.T) {
	script := filepath.Join(t.TempDir(), "daemonize.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nsleep 60 &\necho v1.0.0\n"), 0o700)) //nolint:gosec // the script must be executable to run

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r := &RealCmdRunner{}
		_, _, _ = r.RunCommandContext(ctx, script)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("RunCommandContext did not return; the 200ms timeout was not enforced")
	}
}

// The ordinary path must be unaffected: a command that finishes normally still
// returns its output rather than being cut off by WaitDelay.
func TestRunCommandContext_NormalCommandStillReturnsOutput(t *testing.T) {
	script := filepath.Join(t.TempDir(), "version.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho v1.2.3\n"), 0o700)) //nolint:gosec // the script must be executable to run

	r := &RealCmdRunner{}
	stdout, _, err := r.RunCommandContext(context.Background(), script)
	require.NoError(t, err)
	require.Contains(t, stdout, "v1.2.3")
}
