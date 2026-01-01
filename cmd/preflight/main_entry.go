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

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	// Checks passed - exec into command if args were provided
	if err := runExec(execArgs); err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}
}
