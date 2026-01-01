package main

import (
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/exec"
)

// Version is set at build time via ldflags
var Version = "dev"

// knownSubcommands lists all valid preflight subcommands
var knownSubcommands = []string{"cmd", "env", "file", "git", "hash", "http", "json", "prometheus", "resource", "sys", "tcp", "user", "run", "version", "help", "--help", "-h"}

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
	if slices.Contains(knownSubcommands, firstArg) {
		return args, ""
	}

	// Check if it's a file - if so, treat as hashbang invocation
	if checkFile(firstArg) {
		newArgs := append([]string{args[0], "run"}, args[2:]...)
		return newArgs, firstArg
	}

	return args, ""
}

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
// runExec invokes the configured executor with the provided command and arguments.
// It returns the error returned by the executor, or nil if no execArgs are provided.
func runExec(execArgs []string) error {
	if len(execArgs) == 0 {
		return nil
	}
	return executor.Exec(execArgs[0], execArgs[1:])
}

var rootCmd = &cobra.Command{
	Use:     "preflight",
	Short:   "Docker preflight checks for your runtime environment",
	Long:    "Preflight is a CLI tool for running sanity checks on container and CI environments.",
	Version: Version,
}