package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/preflightfile"
)

var runFile string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run checks from a .preflight file",
	Args:  cobra.NoArgs,
	RunE:  runPreflightFile,
}

func init() {
	runCmd.Flags().StringVar(&runFile, "file", "", "path to .preflight file (default: search up from current directory)")
	rootCmd.AddCommand(runCmd)
}

func runPreflightFile(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	preflightPath, err := preflightfile.FindFile(wd, runFile)
	if err != nil {
		return err
	}

	commands, err := preflightfile.ParseFile(preflightPath)
	if err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	exitCode, err := runCommands(commands, executable, spawn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// spawn runs one command with the parent's streams attached.
func spawn(name string, args []string) error {
	cmd := exec.Command(name, args...) //nolint:gosec // intentional: executing commands from .preflight file
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// runCommands executes each .preflight command, returning the exit code to
// propagate. run is injected so the loop — including the substitution below —
// can be tested without spawning processes or calling os.Exit.
func runCommands(commands []string, executable string, run func(name string, args []string) error) (exitCode int, err error) {
	for _, command := range commands {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			continue
		}

		// ParseFile guarantees the first token is exactly "preflight", so this
		// always fires. Substituting unconditionally means a parser change can
		// never turn a .preflight line into a path to some other binary.
		parts[0] = executable

		if err := run(parts[0], parts[1:]); err != nil {
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				return exitError.ExitCode(), nil
			}
			return 0, fmt.Errorf("failed to execute command %q: %w", command, err)
		}
		checkRan = true
	}

	return 0, nil
}
