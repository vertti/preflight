package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/cmdcheck"
	"github.com/vertti/preflight/pkg/version"
)

var (
	minVersion     string
	maxVersion     string
	exactVersion   string
	matchPattern   string
	versionCmd     string
	cmdTimeout     time.Duration
	versionPattern string
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
	cmdCmd.Flags().StringVar(&versionPattern, "version-regex", "", "regex with capture group to extract version")
	cmdCmd.Flags().StringVar(&versionCmd, "version-cmd", "--version", "command to get version")
	cmdCmd.Flags().DurationVar(&cmdTimeout, "timeout", cmdcheck.DefaultTimeout, "timeout for version command")
	rootCmd.AddCommand(cmdCmd)
}

func runCmdCheck(_ *cobra.Command, args []string) error {
	commandName := args[0]

	c := &cmdcheck.Check{
		Name:           commandName,
		VersionArgs:    parseVersionArgs(versionCmd),
		MatchPattern:   matchPattern,
		VersionPattern: versionPattern,
		Timeout:        cmdTimeout,
		Runner:         &cmdcheck.RealCmdRunner{},
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

	return runCheck(c)
}

func parseVersionArgs(s string) []string {
	return strings.Fields(s)
}
