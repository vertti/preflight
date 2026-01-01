// Package exec provides process replacement functionality for entrypoint mode.
package exec

import (
	"os"
	"os/exec"
)

// Executor handles process replacement after successful checks.
type Executor interface {
	// Exec replaces the current process with the specified command.
	// On Unix, this uses syscall.Exec. On Windows, returns an error.
	Exec(name string, args []string) error
}

// RealExecutor is the production implementation.
type RealExecutor struct{}

// lookPath finds the executable in PATH.
func lookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// environ returns the current environment.
func environ() []string {
	return os.Environ()
}
