package cmdcheck

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// DefaultTimeout is the default timeout for version commands.
const DefaultTimeout = 30 * time.Second

// CmdRunner abstracts command execution for testability.
type CmdRunner interface {
	LookPath(file string) (string, error)
	RunCommandContext(ctx context.Context, name string, args ...string) (stdout, stderr string, err error)
}

// RealCmdRunner implements CmdRunner using actual os/exec.
type RealCmdRunner struct{}

func (r *RealCmdRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// waitDelay bounds how long Wait blocks after the context is cancelled. Because
// Stdout and Stderr are buffers rather than files, os/exec pipes them and waits
// for EOF, but CommandContext kills only the direct child — a grandchild holding
// the write end keeps the pipe open, so without this the timeout is
// unenforceable and a version command that daemonizes hangs the check forever.
const waitDelay = time.Second

func (r *RealCmdRunner) RunCommandContext(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // intentional: executing user-specified version check commands
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	cmd.WaitDelay = waitDelay
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
