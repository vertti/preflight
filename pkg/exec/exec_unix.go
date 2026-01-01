//go:build unix

package exec

import (
	"syscall"
)

// execFunc is the function used to exec into a new process.
// Can be overridden for testing.
var execFunc = syscall.Exec

// Exec replaces the current process with the specified command.
// This is the Unix implementation using syscall.Exec.
func (e *RealExecutor) Exec(name string, args []string) error {
	binary, err := lookPath(name)
	if err != nil {
		return err
	}

	// syscall.Exec replaces the current process.
	// argv[0] must be the program name by convention.
	argv := append([]string{name}, args...)
	env := environ()

	// #nosec G204 -- This is intentional: exec mode allows users to specify the command to run after checks pass.
	// The command comes from CLI args which are under user control.
	return execFunc(binary, argv, env)
}
