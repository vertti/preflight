package cmdcheck

import (
	"bytes"
	"os/exec"
)

// CmdRunner abstracts command execution for testability.
type CmdRunner interface {
	LookPath(file string) (string, error)
	RunCommand(name string, args ...string) (stdout, stderr string, err error)
}

// RealCmdRunner implements CmdRunner using actual os/exec.
type RealCmdRunner struct{}

func (r *RealCmdRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (r *RealCmdRunner) RunCommand(name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
