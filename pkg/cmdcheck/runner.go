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

func (r *RealCmdRunner) RunCommandContext(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
