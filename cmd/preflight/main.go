package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/output"
	"github.com/vertti/preflight/pkg/preflightfile"
	"github.com/vertti/preflight/pkg/tcpcheck"
	"github.com/vertti/preflight/pkg/usercheck"
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

var tcpCmd = &cobra.Command{
	Use:   "tcp <host:port>",
	Short: "Check TCP connectivity to a host:port",
	Args:  cobra.ExactArgs(1),
	RunE:  runTCPCheck,
}

var userCmd = &cobra.Command{
	Use:   "user <username>",
	Short: "Check that a user exists and meets requirements",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserCheck,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run checks from a .preflight file",
	Args:  cobra.NoArgs,
	RunE:  runRun,
}

var (
	// tcp flags
	tcpTimeout time.Duration

	// user flags
	userUID  string
	userGID  string
	userHome string

	// run flags
	runFile string
)

func init() {
	// tcp subcommand
	tcpCmd.Flags().DurationVar(&tcpTimeout, "timeout", 5*time.Second, "connection timeout")
	rootCmd.AddCommand(tcpCmd)

	// user subcommand
	userCmd.Flags().StringVar(&userUID, "uid", "", "expected user ID")
	userCmd.Flags().StringVar(&userGID, "gid", "", "expected primary group ID")
	userCmd.Flags().StringVar(&userHome, "home", "", "expected home directory")
	rootCmd.AddCommand(userCmd)

	// run subcommand
	runCmd.Flags().StringVar(&runFile, "file", "", "path to .preflight file (default: search up from current directory)")
	rootCmd.AddCommand(runCmd)
}

func runTCPCheck(cmd *cobra.Command, args []string) error {
	address := args[0]

	c := &tcpcheck.Check{
		Address: address,
		Timeout: tcpTimeout,
		Dialer:  &tcpcheck.RealDialer{},
	}

	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}

func runUserCheck(cmd *cobra.Command, args []string) error {
	username := args[0]

	c := &usercheck.Check{
		Username: username,
		UID:      userUID,
		GID:      userGID,
		Home:     userHome,
		Lookup:   &usercheck.RealUserLookup{},
	}

	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
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
