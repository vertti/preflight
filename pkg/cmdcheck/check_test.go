package cmdcheck

import (
	"errors"
	"testing"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/version"
)

func TestCommandCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
		wantName   string
	}{
		// Basic existence tests
		{
			name: "command not found",
			check: Check{
				Name: "nonexistent",
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "", errors.New("executable file not found in $PATH")
					},
				},
			},
			wantStatus: check.StatusFail,
			wantName:   "cmd: nonexistent",
		},
		{
			name: "version command fails",
			check: Check{
				Name: "broken",
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/broken", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "", "error loading shared library", errors.New("exit 1")
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "command found and runs",
			check: Check{
				Name: "node",
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v18.17.0", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
			wantName:   "cmd: node",
		},

		// --min version tests
		{
			name: "min version passes",
			check: Check{
				Name:       "node",
				MinVersion: &version.Version{Major: 18, Minor: 0, Patch: 0},
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v18.17.0", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "min version fails",
			check: Check{
				Name:       "node",
				MinVersion: &version.Version{Major: 18, Minor: 0, Patch: 0},
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v16.0.0", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// --max version tests
		{
			name: "max version passes",
			check: Check{
				Name:       "node",
				MaxVersion: &version.Version{Major: 22, Minor: 0, Patch: 0},
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v18.17.0", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "max version fails (at boundary)",
			check: Check{
				Name:       "node",
				MaxVersion: &version.Version{Major: 22, Minor: 0, Patch: 0},
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v22.0.0", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// --exact version tests
		{
			name: "exact version passes",
			check: Check{
				Name:         "node",
				ExactVersion: &version.Version{Major: 18, Minor: 17, Patch: 0},
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v18.17.0", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "exact version fails",
			check: Check{
				Name:         "node",
				ExactVersion: &version.Version{Major: 18, Minor: 17, Patch: 0},
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v18.17.1", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},

		// --match pattern tests
		{
			name: "match pattern passes",
			check: Check{
				Name:         "node",
				MatchPattern: `^v18\.`,
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v18.17.0", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "match pattern fails",
			check: Check{
				Name:         "node",
				MatchPattern: `^v18\.`,
				Runner: &MockRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandFunc: func(name string, args ...string) (string, string, error) {
						return "v16.0.0", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", result.Status, tt.wantStatus)
			}

			if tt.wantName != "" && result.Name != tt.wantName {
				t.Errorf("name = %q, want %q", result.Name, tt.wantName)
			}
		})
	}
}
