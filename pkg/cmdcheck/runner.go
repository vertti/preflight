package cmdcheck

import (
	"bytes"
	"os/exec"
)

// Runner abstracts command execution for testability.
type Runner interface {
	LookPath(file string) (string, error)
	RunCommand(name string, args ...string) (stdout, stderr string, err error)
}

// RealRunner implements Runner using actual OS commands.
type RealRunner struct{}

// LookPath searches for an executable in PATH.
func (r *RealRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// RunCommand executes a command and returns its output.
func (r *RealRunner) RunCommand(name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// MockRunner is a test double for Runner.
type MockRunner struct {
	LookPathFunc   func(file string) (string, error)
	RunCommandFunc func(name string, args ...string) (string, string, error)
}

// LookPath calls the mock function.
func (m *MockRunner) LookPath(file string) (string, error) {
	return m.LookPathFunc(file)
}

// RunCommand calls the mock function.
func (m *MockRunner) RunCommand(name string, args ...string) (stdout, stderr string, err error) {
	return m.RunCommandFunc(name, args...)
}
