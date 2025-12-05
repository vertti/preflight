package main

import (
	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/envcheck"
)

var (
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
	envMinLen     int
	envMaxLen     int
)

var envCmd = &cobra.Command{
	Use:   "env <variable>",
	Short: "Check that an environment variable is set",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvCheck,
}

func init() {
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
	envCmd.Flags().IntVar(&envMinLen, "min-len", 0, "minimum string length")
	envCmd.Flags().IntVar(&envMaxLen, "max-len", 0, "maximum string length")
	rootCmd.AddCommand(envCmd)
}

func runEnvCheck(_ *cobra.Command, args []string) error {
	varName := args[0]

	c := &envcheck.Check{
		Name:       varName,
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
		MinLen:     envMinLen,
		MaxLen:     envMaxLen,
		Getter:     &envcheck.RealEnvGetter{},
	}

	return runCheck(c)
}
