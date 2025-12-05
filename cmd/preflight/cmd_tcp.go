package main

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/tcpcheck"
)

var tcpTimeout time.Duration

var tcpCmd = &cobra.Command{
	Use:   "tcp <host:port>",
	Short: "Check TCP connectivity to a host:port",
	Args:  cobra.ExactArgs(1),
	RunE:  runTCPCheck,
}

func init() {
	tcpCmd.Flags().DurationVar(&tcpTimeout, "timeout", 5*time.Second, "connection timeout")
	rootCmd.AddCommand(tcpCmd)
}

func runTCPCheck(_ *cobra.Command, args []string) error {
	address := args[0]

	c := &tcpcheck.Check{
		Address: address,
		Timeout: tcpTimeout,
		Dialer:  &tcpcheck.RealDialer{},
	}

	return runCheck(c)
}
