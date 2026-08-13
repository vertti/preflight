package main

import (
	"os"
	"strings"
)

// fileChecker abstracts file existence checks for testing
type fileChecker func(path string) (isFile bool)

// realFileChecker checks if a path exists and is a file (not a directory)
func realFileChecker(path string) bool {
	info, err := os.Stat(path) //nolint:gosec // intentional: checking hashbang script path from user invocation
	return err == nil && !info.IsDir()
}

// transformArgsForHashbang detects hashbang invocation and transforms args.
// When preflight is invoked as a hashbang interpreter (e.g., #!/usr/bin/env preflight),
// the first arg is the script file path. This transforms ["preflight", "script.pf"]
// into ["preflight", "run"] and sets runFile to "script.pf".
func transformArgsForHashbang(args []string, checkFile fileChecker) (newArgs []string, filePath string) {
	if len(args) <= 1 {
		return args, ""
	}

	firstArg := args[1]

	// Skip if it's a flag
	if strings.HasPrefix(firstArg, "-") {
		return args, ""
	}

	// Skip if it's a known subcommand
	if isKnownSubcommand(firstArg) {
		return args, ""
	}

	// Check if it's a file - if so, treat as hashbang invocation
	if checkFile(firstArg) {
		newArgs := append([]string{args[0], "run"}, args[2:]...)
		return newArgs, firstArg
	}

	return args, ""
}
