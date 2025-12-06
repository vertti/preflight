package cmdcheck

import (
	"context"
	"errors"
	"testing"
	"time"

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
				Runner: &mockCmdRunner{
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/broken", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
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
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "v16.0.0", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "invalid regex pattern",
			check: Check{
				Name:         "node",
				MatchPattern: `[invalid`,
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/node", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "v18.0.0", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "version parse fails",
			check: Check{
				Name:       "myapp",
				MinVersion: &version.Version{Major: 1, Minor: 0, Patch: 0},
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/myapp", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "no version info here", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "version from stderr when stdout empty",
			check: Check{
				Name: "java",
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/java", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "", "openjdk 17.0.1 2021-10-19", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "custom version args",
			check: Check{
				Name:        "ffmpeg",
				VersionArgs: []string{"-version"},
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/ffmpeg", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						if len(args) == 1 && args[0] == "-version" {
							return "ffmpeg version 5.1.2", "", nil
						}
						return "", "", errors.New("wrong args")
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --timeout tests
		{
			name: "timeout triggers on slow command",
			check: Check{
				Name:    "slowcmd",
				Timeout: 10 * time.Millisecond,
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/slowcmd", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						// Simulate slow command by waiting for context cancellation
						<-ctx.Done()
						return "", "", ctx.Err()
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "custom timeout passes when command is fast",
			check: Check{
				Name:    "fastcmd",
				Timeout: 5 * time.Second,
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/fastcmd", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "v1.0.0", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --version-regex tests
		{
			name: "version-regex extracts version with capture group",
			check: Check{
				Name:           "myapp",
				VersionPattern: `version[:\s]+(\d+\.\d+\.\d+)`,
				MinVersion:     &version.Version{Major: 2, Minor: 0, Patch: 0},
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/myapp", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "myapp version: 2.5.3 (built 2024-01-01)", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "version-regex extraction fails min version",
			check: Check{
				Name:           "myapp",
				VersionPattern: `version[:\s]+(\d+\.\d+\.\d+)`,
				MinVersion:     &version.Version{Major: 3, Minor: 0, Patch: 0},
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/myapp", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "myapp version: 2.5.3", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "version-regex pattern does not match",
			check: Check{
				Name:           "myapp",
				VersionPattern: `version[:\s]+(\d+\.\d+\.\d+)`,
				MinVersion:     &version.Version{Major: 1, Minor: 0, Patch: 0},
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/myapp", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "myapp v2.5.3", "", nil
					},
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "version-regex without capture group uses full match",
			check: Check{
				Name:           "myapp",
				VersionPattern: `\d+\.\d+\.\d+`,
				MinVersion:     &version.Version{Major: 2, Minor: 0, Patch: 0},
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/myapp", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "version 2.5.3 ready", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "version-regex extracts and displays version without constraints",
			check: Check{
				Name:           "myapp",
				VersionPattern: `v(\d+\.\d+\.\d+)`,
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/myapp", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "myapp v18.17.0-beta", "", nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "version-regex invalid regex",
			check: Check{
				Name:           "myapp",
				VersionPattern: `[invalid`,
				MinVersion:     &version.Version{Major: 1, Minor: 0, Patch: 0},
				Runner: &mockCmdRunner{
					LookPathFunc: func(file string) (string, error) {
						return "/usr/bin/myapp", nil
					},
					RunCommandContextFunc: func(ctx context.Context, name string, args ...string) (string, string, error) {
						return "v1.0.0", "", nil
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
