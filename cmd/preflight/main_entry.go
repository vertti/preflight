package main

import (
	"fmt"
	"os"
)

// main parses and transforms command-line arguments, runs the root Cobra command,
// and optionally execs a target command specified after a "--" delimiter.
//
// It updates global runFile when a hashbang-determined file is found, extracts
// exec arguments (everything after "--"), executes rootCmd and exits with status
// 1 on failure, then calls runExec with the extracted arguments and prints any
// exec error to stderr before exiting non-zero.
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