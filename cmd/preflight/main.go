package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

// knownSubcommands lists all valid preflight subcommands
var knownSubcommands = []string{"cmd", "env", "file", "hash", "http", "sys", "tcp", "user", "run", "version", "help", "--help", "-h"}

// fileChecker abstracts file existence checks for testing
type fileChecker func(path string) (isFile bool)

// realFileChecker checks if a path exists and is a file (not a directory)
func realFileChecker(path string) bool {
	info, err := os.Stat(path)
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
	for _, subcmd := range knownSubcommands {
		if firstArg == subcmd {
			return args, ""
		}
	}

	// Check if it's a file - if so, treat as hashbang invocation
	if checkFile(firstArg) {
		newArgs := append([]string{args[0], "run"}, args[2:]...)
		return newArgs, firstArg
	}

	return args, ""
}

func main() {
	var file string
	os.Args, file = transformArgsForHashbang(os.Args, realFileChecker)
	if file != "" {
		runFile = file
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "preflight",
	Short:   "Docker preflight checks for your runtime environment",
	Long:    "Preflight is a CLI tool for running sanity checks on container and CI environments.",
	Version: Version,
}
