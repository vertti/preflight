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
	RunE:  runRun,
}

func init() {
	runCmd.Flags().StringVar(&runFile, "file", "", "path to .preflight file (default: search up from current directory)")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
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

	for _, command := range commands {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			continue
		}

		if parts[0] == "preflight" {
			parts[0] = executable
		}

		execCmd := exec.Command(parts[0], parts[1:]...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				os.Exit(exitError.ExitCode())
			}
			return fmt.Errorf("failed to execute command %q: %w", command, err)
		}
	}

	return nil
}
