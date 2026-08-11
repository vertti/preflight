package main

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/vertti/preflight/pkg/check"
)

func TestTransformArgsForHashbang(t *testing.T) {
	// Mock file checker that returns true for specific paths
	mockFileChecker := func(existingFiles map[string]bool) fileChecker {
		return func(path string) bool {
			return existingFiles[path]
		}
	}

	tests := []struct {
		name          string
		args          []string
		existingFiles map[string]bool
		wantArgs      []string
		wantFile      string
	}{
		{
			name:          "no args",
			args:          []string{"preflight"},
			existingFiles: map[string]bool{},
			wantArgs:      []string{"preflight"},
			wantFile:      "",
		},
		{
			name:          "known subcommand cmd",
			args:          []string{"preflight", "cmd", "node"},
			existingFiles: map[string]bool{},
			wantArgs:      []string{"preflight", "cmd", "node"},
			wantFile:      "",
		},
		{
			name:          "known subcommand env",
			args:          []string{"preflight", "env", "PATH"},
			existingFiles: map[string]bool{},
			wantArgs:      []string{"preflight", "env", "PATH"},
			wantFile:      "",
		},
		{
			name:          "flag arg",
			args:          []string{"preflight", "--help"},
			existingFiles: map[string]bool{},
			wantArgs:      []string{"preflight", "--help"},
			wantFile:      "",
		},
		{
			name:          "hashbang invocation with file",
			args:          []string{"preflight", "/path/to/script.pf"},
			existingFiles: map[string]bool{"/path/to/script.pf": true},
			wantArgs:      []string{"preflight", "run"},
			wantFile:      "/path/to/script.pf",
		},
		{
			name:          "hashbang with extra args",
			args:          []string{"preflight", "script.pf", "--verbose"},
			existingFiles: map[string]bool{"script.pf": true},
			wantArgs:      []string{"preflight", "run", "--verbose"},
			wantFile:      "script.pf",
		},
		{
			name:          "non-existent file treated as unknown command",
			args:          []string{"preflight", "nonexistent.pf"},
			existingFiles: map[string]bool{},
			wantArgs:      []string{"preflight", "nonexistent.pf"},
			wantFile:      "",
		},
		{
			name:          "help flag",
			args:          []string{"preflight", "-h"},
			existingFiles: map[string]bool{},
			wantArgs:      []string{"preflight", "-h"},
			wantFile:      "",
		},
		{
			name:          "version subcommand",
			args:          []string{"preflight", "version"},
			existingFiles: map[string]bool{},
			wantArgs:      []string{"preflight", "version"},
			wantFile:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := mockFileChecker(tt.existingFiles)
			gotArgs, gotFile := transformArgsForHashbang(tt.args, checker)

			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
			if gotFile != tt.wantFile {
				t.Errorf("file = %q, want %q", gotFile, tt.wantFile)
			}
		})
	}
}

// mockExecutor is a test implementation of exec.Executor
type mockExecutor struct {
	execFunc func(name string, args []string) error
}

func (m *mockExecutor) Exec(name string, args []string) error {
	if m.execFunc != nil {
		return m.execFunc(name, args)
	}
	return nil
}

type passingChecker struct{}

func (passingChecker) Run() check.Result {
	return check.Result{Name: "stub", Status: check.StatusOK}
}

type failingChecker struct{}

func (failingChecker) Run() check.Result {
	return check.Result{Name: "stub", Status: check.StatusFail}
}

func TestRunExec_RequiresACheckToHaveRun(t *testing.T) {
	originalExecutor := executor
	originalCheckRan := checkRan
	defer func() { executor = originalExecutor; checkRan = originalCheckRan }()

	t.Run("refuses to exec when no check ran", func(t *testing.T) {
		execCalled := false
		executor = &mockExecutor{execFunc: func(string, []string) error {
			execCalled = true
			return nil
		}}
		checkRan = false

		err := runExec([]string{"./myapp"})
		if err == nil {
			t.Fatal("runExec() = nil, want error when no check ran")
		}
		if execCalled {
			t.Error("executor was called despite no check having run")
		}
	})

	t.Run("execs when a check ran", func(t *testing.T) {
		execCalled := false
		executor = &mockExecutor{execFunc: func(string, []string) error {
			execCalled = true
			return nil
		}}
		checkRan = true

		if err := runExec([]string{"./myapp"}); err != nil {
			t.Fatalf("runExec() = %v, want nil", err)
		}
		if !execCalled {
			t.Error("executor was not called after a successful check")
		}
	})

	t.Run("no exec args is not an error even without a check", func(t *testing.T) {
		checkRan = false
		if err := runExec(nil); err != nil {
			t.Errorf("runExec(nil) = %v, want nil", err)
		}
	})
}

func TestRunCheckRecordsThatItRan(t *testing.T) {
	original := checkRan
	defer func() { checkRan = original }()

	checkRan = false
	_ = runCheck(passingChecker{})
	if !checkRan {
		t.Error("runCheck did not record that a check ran")
	}

	checkRan = false
	_ = runCheck(failingChecker{})
	if !checkRan {
		t.Error("runCheck did not record a failing check as having run")
	}
}

func TestRunExec(t *testing.T) {
	// Save original executor and restore after test
	originalExecutor := executor
	originalCheckRan := checkRan
	checkRan = true
	defer func() { executor = originalExecutor; checkRan = originalCheckRan }()

	t.Run("empty args returns nil", func(t *testing.T) {
		err := runExec([]string{})
		if err != nil {
			t.Errorf("runExec([]) = %v, want nil", err)
		}
	})

	t.Run("nil args returns nil", func(t *testing.T) {
		err := runExec(nil)
		if err != nil {
			t.Errorf("runExec(nil) = %v, want nil", err)
		}
	})

	t.Run("calls executor with correct args", func(t *testing.T) {
		var calledName string
		var calledArgs []string

		executor = &mockExecutor{
			execFunc: func(name string, args []string) error {
				calledName = name
				calledArgs = args
				return nil
			},
		}

		err := runExec([]string{"./myapp", "arg1", "arg2"})
		if err != nil {
			t.Errorf("runExec() = %v, want nil", err)
		}
		if calledName != "./myapp" {
			t.Errorf("name = %q, want %q", calledName, "./myapp")
		}
		if !reflect.DeepEqual(calledArgs, []string{"arg1", "arg2"}) {
			t.Errorf("args = %v, want %v", calledArgs, []string{"arg1", "arg2"})
		}
	})

	t.Run("single arg no extra args", func(t *testing.T) {
		var calledName string
		var calledArgs []string

		executor = &mockExecutor{
			execFunc: func(name string, args []string) error {
				calledName = name
				calledArgs = args
				return nil
			},
		}

		err := runExec([]string{"./myapp"})
		if err != nil {
			t.Errorf("runExec() = %v, want nil", err)
		}
		if calledName != "./myapp" {
			t.Errorf("name = %q, want %q", calledName, "./myapp")
		}
		if len(calledArgs) != 0 {
			t.Errorf("args = %v, want empty", calledArgs)
		}
	})

	t.Run("returns executor error", func(t *testing.T) {
		expectedErr := errors.New("exec failed")
		executor = &mockExecutor{
			execFunc: func(name string, args []string) error {
				return expectedErr
			},
		}

		err := runExec([]string{"./myapp"})
		if !errors.Is(err, expectedErr) {
			t.Errorf("runExec() = %v, want %v", err, expectedErr)
		}
	})
}

// The set of subcommands must come from cobra rather than a hand-kept list.
// The list had already drifted: it named "version", which is not a command, and
// a command added without updating it would be mistaken for a script path
// whenever a file of that name sat in the working directory.
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

func TestExtractExecArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantArgs     []string
		wantExecArgs []string
	}{
		{
			name:         "no double dash",
			args:         []string{"preflight", "cmd", "node"},
			wantArgs:     []string{"preflight", "cmd", "node"},
			wantExecArgs: nil,
		},
		{
			name:         "double dash with exec command",
			args:         []string{"preflight", "tcp", "localhost:5432", "--", "./myapp"},
			wantArgs:     []string{"preflight", "tcp", "localhost:5432"},
			wantExecArgs: []string{"./myapp"},
		},
		{
			name:         "double dash with exec command and args",
			args:         []string{"preflight", "http", "localhost:8080", "--", "./myapp", "arg1", "arg2"},
			wantArgs:     []string{"preflight", "http", "localhost:8080"},
			wantExecArgs: []string{"./myapp", "arg1", "arg2"},
		},
		{
			name:         "double dash at end (no exec args)",
			args:         []string{"preflight", "env", "PATH", "--"},
			wantArgs:     []string{"preflight", "env", "PATH"},
			wantExecArgs: []string{},
		},
		{
			name:         "double dash with flags before",
			args:         []string{"preflight", "tcp", "localhost:5432", "--retry", "5", "--", "./app"},
			wantArgs:     []string{"preflight", "tcp", "localhost:5432", "--retry", "5"},
			wantExecArgs: []string{"./app"},
		},
		{
			name:         "only preflight and double dash",
			args:         []string{"preflight", "--", "./app"},
			wantArgs:     []string{"preflight"},
			wantExecArgs: []string{"./app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := make([]string, len(tt.args))
			copy(args, tt.args)

			gotExecArgs := extractExecArgs(&args)

			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
			if !reflect.DeepEqual(gotExecArgs, tt.wantExecArgs) {
				t.Errorf("execArgs = %v, want %v", gotExecArgs, tt.wantExecArgs)
			}
		})
	}
}
