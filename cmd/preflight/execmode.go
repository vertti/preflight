package main

import (
	"errors"

	"github.com/vertti/preflight/pkg/exec"
)

// extractExecArgs finds "--" in args and returns everything after it.
// It modifies the input slice to remove the "--" and everything after.
// This allows syntax like: preflight tcp postgres:5432 -- ./myapp
func extractExecArgs(args *[]string) []string {
	for i, arg := range *args {
		if arg == "--" {
			execArgs := (*args)[i+1:]
			*args = (*args)[:i]
			return execArgs
		}
	}
	return nil
}

// executor is the default exec implementation, can be overridden for testing.
var executor exec.Executor = &exec.RealExecutor{}

// runExec executes the command specified in execArgs.
// Returns an error if the exec fails, or if no check ran — handing control to
// the target when nothing was verified would turn the gate into a no-op that
// reports success.
func runExec(execArgs []string) error {
	if len(execArgs) == 0 {
		return nil
	}
	if !checkRan {
		return errors.New("refusing to exec: no check ran before --")
	}
	return executor.Exec(execArgs[0], execArgs[1:])
}
