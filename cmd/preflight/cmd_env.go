package main

import (
	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/envcheck"
)

var (
	envNotSet     bool
	envAllowEmpty bool
	envMatch      string
	envExact      string
	envOneOf      []string
	envHideValue  bool
	envMaskValue  bool
	envStartsWith string
	envEndsWith   string
	envContains   string
	envIsNumeric  bool
	envIsPort     bool
	envIsURL      bool
	envIsJSON     bool
	envIsBool     bool
	envIsFile     bool
	envIsDir      bool
	envMinLen     int
	envMaxLen     int
	envMinValue   float64
	envMaxValue   float64
)

var envCmd = &cobra.Command{
	Use:   "env <variable>",
	Short: "Check that an environment variable is set",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvCheck,
}

func init() {
	envCmd.Flags().BoolVar(&envNotSet, "not-set", false, "verify variable is NOT defined")
	envCmd.Flags().BoolVar(&envAllowEmpty, "allow-empty", false, "pass if defined but empty")
	envCmd.Flags().StringVar(&envMatch, "match", "", "regex pattern to match value")
	envCmd.Flags().StringVar(&envExact, "exact", "", "exact value required")
	envCmd.Flags().StringSliceVar(&envOneOf, "one-of", nil, "value must be one of these (comma-separated)")
	envCmd.Flags().BoolVar(&envHideValue, "hide-value", false, "don't show value in output")
	envCmd.Flags().BoolVar(&envMaskValue, "mask-value", false, "show masked value (first/last 3 chars)")
	envCmd.Flags().StringVar(&envStartsWith, "starts-with", "", "value must start with this string")
	envCmd.Flags().StringVar(&envEndsWith, "ends-with", "", "value must end with this string")
	envCmd.Flags().StringVar(&envContains, "contains", "", "value must contain this string")
	envCmd.Flags().BoolVar(&envIsNumeric, "is-numeric", false, "value must be a valid number")
	envCmd.Flags().BoolVar(&envIsPort, "is-port", false, "value must be valid TCP port (1-65535)")
	envCmd.Flags().BoolVar(&envIsURL, "is-url", false, "value must be valid URL")
	envCmd.Flags().BoolVar(&envIsJSON, "is-json", false, "value must be valid JSON")
	envCmd.Flags().BoolVar(&envIsBool, "is-bool", false, "value must be boolean (true/false/1/0/yes/no/on/off)")
	envCmd.Flags().BoolVar(&envIsFile, "is-file", false, "value must be path to existing file")
	envCmd.Flags().BoolVar(&envIsDir, "is-dir", false, "value must be path to existing directory")
	envCmd.Flags().IntVar(&envMinLen, "min-len", 0, "minimum string length")
	envCmd.Flags().IntVar(&envMaxLen, "max-len", 0, "maximum string length")
	envCmd.Flags().Float64Var(&envMinValue, "min-value", 0, "minimum numeric value (use with --is-numeric)")
	envCmd.Flags().Float64Var(&envMaxValue, "max-value", 0, "maximum numeric value (use with --is-numeric)")
	rootCmd.AddCommand(envCmd)
}

func runEnvCheck(cmd *cobra.Command, args []string) error {
	varName := args[0]

	c := &envcheck.Check{
		Name:       varName,
		NotSet:     envNotSet,
		AllowEmpty: envAllowEmpty,
		Match:      envMatch,
		Exact:      envExact,
		OneOf:      envOneOf,
		HideValue:  envHideValue,
		MaskValue:  envMaskValue,
		StartsWith: envStartsWith,
		EndsWith:   envEndsWith,
		Contains:   envContains,
		IsNumeric:  envIsNumeric,
		IsPort:     envIsPort,
		IsURL:      envIsURL,
		IsJSON:     envIsJSON,
		IsBool:     envIsBool,
		IsFile:     envIsFile,
		IsDir:      envIsDir,
		MinLen:     envMinLen,
		MaxLen:     envMaxLen,
		Getter:     &envcheck.RealEnvGetter{},
		Stater:     &envcheck.RealFileStater{},
	}

	// Only set MinValue/MaxValue if the flags were explicitly provided
	if cmd.Flags().Changed("min-value") {
		c.MinValue = &envMinValue
	}
	if cmd.Flags().Changed("max-value") {
		c.MaxValue = &envMaxValue
	}

	return runCheck(c)
}
