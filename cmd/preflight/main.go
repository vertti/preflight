package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/preflightfile"
)

// Version is set at build time via ldflags
var Version = "dev"

func main() {
	if len(os.Args) > 1 {
		firstArg := os.Args[1]

		if !strings.HasPrefix(firstArg, "-") {
			knownSubcommands := []string{"cmd", "env", "file", "tcp", "user", "run", "version", "help", "--help", "-h"}
			isSubcommand := false
			for _, subcmd := range knownSubcommands {
				if firstArg == subcmd {
					isSubcommand = true
					break
				}
			}

			if !isSubcommand {
				if info, err := os.Stat(firstArg); err == nil && !info.IsDir() {
					runFile = firstArg
					os.Args = append([]string{os.Args[0], "run"}, os.Args[2:]...)
				}
			}
		}
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

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run checks from a .preflight file",
	Args:  cobra.NoArgs,
	RunE:  runRun,
}

var runFile string

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
			if exitError, ok := err.(*exec.ExitError); ok {
				os.Exit(exitError.ExitCode())
			}
			return fmt.Errorf("failed to execute command %q: %w", command, err)
		}
	}

	return nil
}
