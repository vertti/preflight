package main

import (
	"errors"
	"reflect"
	"testing"
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

func TestRunExec(t *testing.T) {
	// Save original executor and restore after test
	originalExecutor := executor
	defer func() { executor = originalExecutor }()

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
