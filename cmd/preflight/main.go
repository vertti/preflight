package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/cmdcheck"
	"github.com/vertti/preflight/pkg/envcheck"
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

var envCmd = &cobra.Command{
	Use:   "env <variable>",
	Short: "Check that an environment variable is set",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvCheck,
}

var (
	// cmd flags
	minVersion   string
	maxVersion   string
	exactVersion string
	matchPattern string
	versionCmd   string

	// env flags
	envRequired  bool
	envMatch     string
	envExact     string
	envHideValue bool
	envMaskValue bool
)

func init() {
	// cmd subcommand
	cmdCmd.Flags().StringVar(&minVersion, "min", "", "minimum version required (inclusive)")
	cmdCmd.Flags().StringVar(&maxVersion, "max", "", "maximum version allowed (exclusive)")
	cmdCmd.Flags().StringVar(&exactVersion, "exact", "", "exact version required")
	cmdCmd.Flags().StringVar(&matchPattern, "match", "", "regex pattern to match against version output")
	cmdCmd.Flags().StringVar(&versionCmd, "version-cmd", "--version", "command to get version")
	rootCmd.AddCommand(cmdCmd)

	// env subcommand
	envCmd.Flags().BoolVar(&envRequired, "required", false, "fail if not set (allows empty)")
	envCmd.Flags().StringVar(&envMatch, "match", "", "regex pattern to match value")
	envCmd.Flags().StringVar(&envExact, "exact", "", "exact value required")
	envCmd.Flags().BoolVar(&envHideValue, "hide-value", false, "don't show value in output")
	envCmd.Flags().BoolVar(&envMaskValue, "mask-value", false, "show masked value (first/last 3 chars)")
	rootCmd.AddCommand(envCmd)
}

func runCmdCheck(cmd *cobra.Command, args []string) error {
	commandName := args[0]

	c := &cmdcheck.Check{
		Name:         commandName,
		VersionArgs:  parseVersionArgs(versionCmd),
		MatchPattern: matchPattern,
		Runner:       &cmdcheck.RealRunner{},
	}

	if minVersion != "" {
		v, err := version.Parse(minVersion)
		if err != nil {
			return fmt.Errorf("invalid --min version: %w", err)
		}
		c.MinVersion = &v
	}

	if maxVersion != "" {
		v, err := version.Parse(maxVersion)
		if err != nil {
			return fmt.Errorf("invalid --max version: %w", err)
		}
		c.MaxVersion = &v
	}

	if exactVersion != "" {
		v, err := version.Parse(exactVersion)
		if err != nil {
			return fmt.Errorf("invalid --exact version: %w", err)
		}
		c.ExactVersion = &v
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

func runEnvCheck(cmd *cobra.Command, args []string) error {
	varName := args[0]

	c := &envcheck.Check{
		Name:      varName,
		Required:  envRequired,
		Match:     envMatch,
		Exact:     envExact,
		HideValue: envHideValue,
		MaskValue: envMaskValue,
		Getter:    &envcheck.RealEnvGetter{},
	}

	result := c.Run()
	printResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
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
