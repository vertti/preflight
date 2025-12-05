package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/output"
	"github.com/vertti/preflight/pkg/syscheck"
)

var (
	sysOS   string
	sysArch string
)

var sysCmd = &cobra.Command{
	Use:   "sys",
	Short: "Check system OS and architecture",
	Long: `Verify the system matches expected OS and/or architecture.

Examples:
  preflight sys --os linux
  preflight sys --arch amd64
  preflight sys --os linux --arch arm64`,
	RunE: runSysCheck,
}

func init() {
	sysCmd.Flags().StringVar(&sysOS, "os", "", "required OS (linux, darwin, windows)")
	sysCmd.Flags().StringVar(&sysArch, "arch", "", "required architecture (amd64, arm64, 386)")
	rootCmd.AddCommand(sysCmd)
}

func runSysCheck(cmd *cobra.Command, args []string) error {
	c := &syscheck.Check{
		ExpectedOS:   sysOS,
		ExpectedArch: sysArch,
		Info:         &syscheck.RealSysInfo{},
	}

	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}
