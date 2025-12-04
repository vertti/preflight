package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/output"
	"github.com/vertti/preflight/pkg/usercheck"
)

var (
	userUID  string
	userGID  string
	userHome string
)

var userCmd = &cobra.Command{
	Use:   "user <username>",
	Short: "Check that a user exists and meets requirements",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserCheck,
}

func init() {
	userCmd.Flags().StringVar(&userUID, "uid", "", "expected user ID")
	userCmd.Flags().StringVar(&userGID, "gid", "", "expected primary group ID")
	userCmd.Flags().StringVar(&userHome, "home", "", "expected home directory")
	rootCmd.AddCommand(userCmd)
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
