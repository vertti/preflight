//go:build windows

package exec

import "errors"

// ErrExecNotSupported indicates exec mode is not available on Windows.
var ErrExecNotSupported = errors.New("exec mode not supported on Windows; use a shell script instead")

// Exec is not supported on Windows.
// Windows does not have a true exec syscall that replaces the current process.
func (e *RealExecutor) Exec(name string, args []string) error {
	return ErrExecNotSupported
}
