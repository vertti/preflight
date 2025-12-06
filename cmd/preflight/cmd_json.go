package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/jsoncheck"
)

var (
	jsonHasKey string
	jsonKey    string
	jsonExact  string
	jsonMatch  string
)

var jsonCmd = &cobra.Command{
	Use:   "json <file>",
	Short: "Validate JSON file and check values",
	Args:  cobra.ExactArgs(1),
	RunE:  runJSONCheck,
}

func init() {
	jsonCmd.Flags().StringVar(&jsonHasKey, "has-key", "", "check that key exists (dot notation for nested)")
	jsonCmd.Flags().StringVar(&jsonKey, "key", "", "key to check value of (dot notation for nested)")
	jsonCmd.Flags().StringVar(&jsonExact, "exact", "", "exact value required (requires --key)")
	jsonCmd.Flags().StringVar(&jsonMatch, "match", "", "regex pattern for value (requires --key)")
	rootCmd.AddCommand(jsonCmd)
}

func runJSONCheck(_ *cobra.Command, args []string) error {
	file := args[0]

	// Validate flag combinations
	if (jsonExact != "" || jsonMatch != "") && jsonKey == "" {
		return errors.New("--exact and --match require --key to be set")
	}

	c := &jsoncheck.Check{
		File:   file,
		HasKey: jsonHasKey,
		Key:    jsonKey,
		Exact:  jsonExact,
		Match:  jsonMatch,
		FS:     &jsoncheck.RealFileSystem{},
	}

	return runCheck(c)
}
