package main

import (
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
