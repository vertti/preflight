package cmdcheck

import (
	"bytes"
	"os/exec"
)

type Runner interface {
	LookPath(file string) (string, error)
	RunCommand(name string, args ...string) (stdout, stderr string, err error)
}

type RealRunner struct{}

func (r *RealRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (r *RealRunner) RunCommand(name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}
