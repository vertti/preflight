package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/cmdcheck"
	"github.com/vertti/preflight/pkg/output"
	"github.com/vertti/preflight/pkg/version"
)

var (
	minVersion   string
	maxVersion   string
	exactVersion string
	matchPattern string
	versionCmd   string
)

var cmdCmd = &cobra.Command{
	Use:   "cmd <command>",
	Short: "Check that a command exists and can run",
	Args:  cobra.ExactArgs(1),
	RunE:  runCmdCheck,
}

func init() {
	cmdCmd.Flags().StringVar(&minVersion, "min", "", "minimum version required (inclusive)")
	cmdCmd.Flags().StringVar(&maxVersion, "max", "", "maximum version allowed (exclusive)")
	cmdCmd.Flags().StringVar(&exactVersion, "exact", "", "exact version required")
	cmdCmd.Flags().StringVar(&matchPattern, "match", "", "regex pattern to match against version output")
	cmdCmd.Flags().StringVar(&versionCmd, "version-cmd", "--version", "command to get version")
	rootCmd.AddCommand(cmdCmd)
}

func runCmdCheck(cmd *cobra.Command, args []string) error {
	commandName := args[0]

	c := &cmdcheck.Check{
		Name:         commandName,
		VersionArgs:  parseVersionArgs(versionCmd),
		MatchPattern: matchPattern,
		Runner:       &cmdcheck.RealRunner{},
	}

	var err error
	if c.MinVersion, err = version.ParseOptional(minVersion); err != nil {
		return fmt.Errorf("invalid --min version: %w", err)
	}
	if c.MaxVersion, err = version.ParseOptional(maxVersion); err != nil {
		return fmt.Errorf("invalid --max version: %w", err)
	}
	if c.ExactVersion, err = version.ParseOptional(exactVersion); err != nil {
		return fmt.Errorf("invalid --exact version: %w", err)
	}

	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}

func parseVersionArgs(s string) []string {
	return strings.Fields(s)
}
