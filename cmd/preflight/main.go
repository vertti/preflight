package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

func main() {
	if len(os.Args) > 1 {
		firstArg := os.Args[1]

		if !strings.HasPrefix(firstArg, "-") {
			knownSubcommands := []string{"cmd", "env", "file", "tcp", "user", "run", "version", "help", "--help", "-h"}
			isSubcommand := false
			for _, subcmd := range knownSubcommands {
				if firstArg == subcmd {
					isSubcommand = true
					break
				}
			}

			if !isSubcommand {
				if info, err := os.Stat(firstArg); err == nil && !info.IsDir() {
					runFile = firstArg
					os.Args = append([]string{os.Args[0], "run"}, os.Args[2:]...)
				}
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "preflight",
	Short:   "Docker preflight checks for your runtime environment",
	Long:    "Preflight is a CLI tool for running sanity checks on container and CI environments.",
	Version: Version,
}
