package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestKnownSubcommandsMatchesCobra(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		name := cmd.Name()
		t.Run(name, func(t *testing.T) {
			if !isKnownSubcommand(name) {
				t.Errorf("%q is a registered command but not recognized; a file named %q would hijack it", name, name)
			}
		})
	}

	t.Run("does not claim commands that do not exist", func(t *testing.T) {
		registered := map[string]bool{}
		for _, cmd := range rootCmd.Commands() {
			registered[cmd.Name()] = true
		}
		// "version" was listed by hand but has never been a subcommand.
		if isKnownSubcommand("version") && !registered["version"] {
			t.Error(`"version" is recognized as a subcommand but is not registered with cobra`)
		}
	})

	t.Run("help and flags still recognized", func(t *testing.T) {
		for _, name := range []string{"help", "--help", "-h"} {
			if !isKnownSubcommand(name) {
				t.Errorf("%q should be recognized", name)
			}
		}
	})

	t.Run("an arbitrary word is not a subcommand", func(t *testing.T) {
		if isKnownSubcommand("notes.txt") {
			t.Error("notes.txt should not be treated as a subcommand")
		}
	})
}

func TestReportExecuteError(t *testing.T) {
	newCmd := func() *cobra.Command {
		return &cobra.Command{Use: "demo", Short: "a demo", Run: func(*cobra.Command, []string) {}}
	}

	t.Run("a failed check prints nothing extra", func(t *testing.T) {
		var buf bytes.Buffer
		reportExecuteError(newCmd(), ErrCheckFailed, &buf)
		if buf.Len() != 0 {
			t.Errorf("got %q, want no output: [FAIL] was already printed", buf.String())
		}
	})

	t.Run("a wrapped check failure also prints nothing", func(t *testing.T) {
		var buf bytes.Buffer
		reportExecuteError(newCmd(), fmt.Errorf("running check: %w", ErrCheckFailed), &buf)
		if buf.Len() != 0 {
			t.Errorf("got %q, want no output", buf.String())
		}
	})

	t.Run("a usage error prints the message and the usage", func(t *testing.T) {
		var buf bytes.Buffer
		reportExecuteError(newCmd(), errors.New("unknown flag: --nope"), &buf)
		out := buf.String()
		if !strings.Contains(out, "unknown flag: --nope") {
			t.Errorf("got %q, want the error message", out)
		}
		if !strings.Contains(out, "Usage:") {
			t.Errorf("got %q, want usage for a usage error", out)
		}
	})

	t.Run("survives a nil command", func(t *testing.T) {
		var buf bytes.Buffer
		reportExecuteError(nil, errors.New("boom"), &buf)
		if !strings.Contains(buf.String(), "boom") {
			t.Errorf("got %q, want the error message", buf.String())
		}
	})
}
