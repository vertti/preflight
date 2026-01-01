package cmdcheck

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/version"
)

func TestCommandCheck_Run(t *testing.T) {
	found := func(string) (string, error) { return "/usr/bin/cmd", nil }
	notFound := func(string) (string, error) { return "", errors.New("not found") }

	run := func(output string) func(context.Context, string, ...string) (string, string, error) {
		return func(context.Context, string, ...string) (string, string, error) { return output, "", nil }
	}
	runStderr := func(output string) func(context.Context, string, ...string) (string, string, error) {
		return func(context.Context, string, ...string) (string, string, error) { return "", output, nil }
	}
	runErr := func(context.Context, string, ...string) (string, string, error) {
		return "", "error", errors.New("exit 1")
	}
	runSlow := func(ctx context.Context, _ string, _ ...string) (string, string, error) {
		<-ctx.Done()
		return "", "", ctx.Err()
	}

	v := func(major, minor, patch int) *version.Version {
		return &version.Version{Major: major, Minor: minor, Patch: patch}
	}

	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
	}{
		// Basic
		{"command not found", Check{Name: "x", Runner: &mockCmdRunner{LookPathFunc: notFound}}, check.StatusFail},
		{"version fails", Check{Name: "x", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: runErr}}, check.StatusFail},
		{"command found", Check{Name: "node", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v18.17.0")}}, check.StatusOK},

		// --min
		{"min passes", Check{Name: "n", MinVersion: v(18, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v18.17.0")}}, check.StatusOK},
		{"min fails", Check{Name: "n", MinVersion: v(18, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v16.0.0")}}, check.StatusFail},

		// --max
		{"max passes", Check{Name: "n", MaxVersion: v(22, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v18.17.0")}}, check.StatusOK},
		{"max fails at boundary", Check{Name: "n", MaxVersion: v(22, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v22.0.0")}}, check.StatusFail},

		// --exact
		{"exact passes", Check{Name: "n", ExactVersion: v(18, 17, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v18.17.0")}}, check.StatusOK},
		{"exact fails", Check{Name: "n", ExactVersion: v(18, 17, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v18.17.1")}}, check.StatusFail},

		// --match
		{"match passes", Check{Name: "n", MatchPattern: `^v18\.`, Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v18.17.0")}}, check.StatusOK},
		{"match fails", Check{Name: "n", MatchPattern: `^v18\.`, Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v16.0.0")}}, check.StatusFail},
		{"match invalid regex", Check{Name: "n", MatchPattern: `[invalid`, Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v18.0.0")}}, check.StatusFail},

		// Parse failures
		{"version parse fails", Check{Name: "n", MinVersion: v(1, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("no version")}}, check.StatusFail},
		{"version from stderr", Check{Name: "java", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: runStderr("openjdk 17.0.1")}}, check.StatusOK},
		{"custom version args", Check{Name: "ff", VersionArgs: []string{"-version"}, Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: func(_ context.Context, _ string, args ...string) (string, string, error) {
			if len(args) == 1 && args[0] == "-version" {
				return "ffmpeg version 5.1.2", "", nil
			}
			return "", "", errors.New("wrong args")
		}}}, check.StatusOK},

		// --timeout
		{"timeout triggers", Check{Name: "slow", Timeout: 10 * time.Millisecond, Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: runSlow}}, check.StatusFail},
		{"timeout passes fast cmd", Check{Name: "fast", Timeout: 5 * time.Second, Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v1.0.0")}}, check.StatusOK},

		// --version-regex
		{"version-regex extracts", Check{Name: "a", VersionPattern: `version[:\s]+(\d+\.\d+\.\d+)`, MinVersion: v(2, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("myapp version: 2.5.3")}}, check.StatusOK},
		{"version-regex extraction fails min", Check{Name: "a", VersionPattern: `version[:\s]+(\d+\.\d+\.\d+)`, MinVersion: v(3, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("myapp version: 2.5.3")}}, check.StatusFail},
		{"version-regex no match", Check{Name: "a", VersionPattern: `version[:\s]+(\d+\.\d+\.\d+)`, MinVersion: v(1, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("myapp v2.5.3")}}, check.StatusFail},
		{"version-regex no capture group", Check{Name: "a", VersionPattern: `\d+\.\d+\.\d+`, MinVersion: v(2, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("version 2.5.3 ready")}}, check.StatusOK},
		{"version-regex display only", Check{Name: "a", VersionPattern: `v(\d+\.\d+\.\d+)`, Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("myapp v18.17.0-beta")}}, check.StatusOK},
		{"version-regex invalid", Check{Name: "a", VersionPattern: `[invalid`, MinVersion: v(1, 0, 0), Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v1.0.0")}}, check.StatusFail},

		// --range semver
		{"range >= passes", Check{Name: "a", VersionRange: ">=2.0.0", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v2.5.3")}}, check.StatusOK},
		{"range >= fails", Check{Name: "a", VersionRange: ">=3.0.0", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v2.5.3")}}, check.StatusFail},
		{"range multiple conditions", Check{Name: "a", VersionRange: ">=2.0.0, <3.0.0", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v2.5.3")}}, check.StatusOK},
		{"range caret", Check{Name: "a", VersionRange: "^1.5.0", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v1.9.0")}}, check.StatusOK},
		{"range tilde", Check{Name: "a", VersionRange: "~1.5.0", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v1.5.9")}}, check.StatusOK},
		{"range tilde fails minor", Check{Name: "a", VersionRange: "~1.5.0", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v1.6.0")}}, check.StatusFail},
		{"range invalid", Check{Name: "a", VersionRange: "invalid constraint", Runner: &mockCmdRunner{LookPathFunc: found, RunCommandContextFunc: run("v1.0.0")}}, check.StatusFail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
		})
	}
}
