package main

import (
	"errors"
	"fmt"
	"io"
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

// SilenceUsage and SilenceErrors are set because a failing check is the tool's
// normal operating mode, not a usage mistake. Cobra treats any error from RunE
// as a usage error and dumps the full flag list, which buried the one line that
// mattered. reportExecuteError puts the usage output back for real usage errors.
var rootCmd = &cobra.Command{
	Use:           "preflight",
	Short:         "Docker preflight checks for your runtime environment",
	Long:          "Preflight is a CLI tool for running sanity checks on container and CI environments.",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// reportExecuteError writes diagnostics for a failed Execute. A failed check has
// already printed its [FAIL] line and needs nothing further; anything else is
// the user getting the invocation wrong, where usage is what helps.
func reportExecuteError(cmd *cobra.Command, err error, w io.Writer) {
	if errors.Is(err, ErrCheckFailed) {
		return
	}

	_, _ = fmt.Fprintln(w, "Error:", err)
	if cmd == nil {
		return
	}
	cmd.SetOut(w)
	cmd.SetErr(w)
	_ = cmd.Usage()
}
