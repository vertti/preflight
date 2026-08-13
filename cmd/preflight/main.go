package main

import (
	"fmt"
	"os"
)

func main() {
	var file string
	os.Args, file = transformArgsForHashbang(os.Args, realFileChecker)
	if file != "" {
		runFile = file
	}

	// Extract exec args (everything after "--")
	execArgs := extractExecArgs(&os.Args)

	// ExecuteC returns the command that actually ran, so usage errors can show
	// that command's usage rather than the root's.
	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		reportExecuteError(cmd, err, os.Stderr)
		os.Exit(1)
	}

	// Checks passed - exec into command if args were provided
	if err := runExec(execArgs); err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}
}
