package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/cmdcheck"
	"github.com/vertti/preflight/pkg/version"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "preflight",
	Short: "Docker preflight checks for your runtime environment",
	Long:  "Preflight is a CLI tool for running sanity checks on container and CI environments.",
}

var cmdCmd = &cobra.Command{
	Use:   "cmd <command>",
	Short: "Check that a command exists and can run",
	Args:  cobra.ExactArgs(1),
	RunE:  runCmdCheck,
}

var (
	minVersion string
	maxVersion string
	versionCmd string
)

func init() {
	cmdCmd.Flags().StringVar(&minVersion, "min", "", "minimum version required (inclusive)")
	cmdCmd.Flags().StringVar(&maxVersion, "max", "", "maximum version allowed (exclusive)")
	cmdCmd.Flags().StringVar(&versionCmd, "version-cmd", "--version", "command to get version")
	rootCmd.AddCommand(cmdCmd)
}

func runCmdCheck(cmd *cobra.Command, args []string) error {
	commandName := args[0]

	c := &cmdcheck.Check{
		Name:        commandName,
		VersionArgs: parseVersionArgs(versionCmd),
		Runner:      &cmdcheck.RealRunner{},
	}

	// Parse min version
	if minVersion != "" {
		v, err := version.Parse(minVersion)
		if err != nil {
			return fmt.Errorf("invalid --min version: %w", err)
		}
		c.MinVersion = &v
	}

	// Parse max version
	if maxVersion != "" {
		v, err := version.Parse(maxVersion)
		if err != nil {
			return fmt.Errorf("invalid --max version: %w", err)
		}
		c.MaxVersion = &v
	}

	result := c.Run()
	printResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}

func parseVersionArgs(s string) []string {
	return strings.Fields(s)
}

func printResult(r check.Result) {
	if r.OK() {
		fmt.Printf("[OK] %s\n", r.Name)
	} else {
		fmt.Printf("[FAIL] %s\n", r.Name)
	}
	for _, d := range r.Details {
		fmt.Printf("      %s\n", d)
	}
}
