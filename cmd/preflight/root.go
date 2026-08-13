package main

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

// SilenceUsage and SilenceErrors are set because a failing check is the tool's
// normal operating mode, not a usage mistake. Cobra treats any error from RunE
// as a usage error and dumps the full flag list, which buried the one line that
// mattered. reportExecuteError puts the usage output back for real usage errors.
var rootCmd = &cobra.Command{
	Use:           "preflight",
	Short:         "Docker preflight checks for your runtime environment",
	Long:          "Preflight is a CLI tool for running sanity checks on container and CI environments.",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// isKnownSubcommand reports whether name is one of preflight's own commands.
// Derived from cobra rather than a hand-kept list: the list had drifted (it
// named "version", which is not a command), and adding a command without
// updating it meant a file of that name in the working directory would be
// mistaken for a hashbang script.
func isKnownSubcommand(name string) bool {
	switch name {
	case "help", "--help", "-h":
		return true
	}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return true
		}
		if slices.Contains(cmd.Aliases, name) {
			return true
		}
	}
	return false
}

// reportExecuteError writes diagnostics for a failed Execute. A failed check has
// already printed its [FAIL] line and needs nothing further; anything else is
// the user getting the invocation wrong, where usage is what helps.
func reportExecuteError(cmd *cobra.Command, err error, w io.Writer) {
	if errors.Is(err, ErrCheckFailed) {
		return
	}

	_, _ = fmt.Fprintln(w, "Error:", err)
	if cmd == nil {
		return
	}
	cmd.SetOut(w)
	cmd.SetErr(w)
	_ = cmd.Usage()
}
