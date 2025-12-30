package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

func TestFormatLabel(t *testing.T) {
	// Save and restore color codes
	oldDim, oldReset := dim, reset
	defer func() { dim, reset = oldDim, oldReset }()

	// Test without colors
	dim, reset = "", ""

	tests := []struct {
		input string
		want  string
	}{
		{"cmd: go", "cmd: go"},
		{"path: /usr/bin", "path: /usr/bin"},
		{"no colon here", "no colon here"},
		{"multiple: colons: here", "multiple: colons: here"},
		{"", ""},
	}

	for _, tt := range tests {
		got := formatLabel(tt.input)
		if got != tt.want {
			t.Errorf("formatLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatLabelWithColors(t *testing.T) {
	// Save and restore color codes
	oldDim, oldReset := dim, reset
	defer func() { dim, reset = oldDim, oldReset }()

	// Set test colors
	dim, reset = "[DIM]", "[RESET]"

	tests := []struct {
		input string
		want  string
	}{
		{"cmd: go", "[DIM]cmd:[RESET] go"},
		{"path: /usr/bin", "[DIM]path:[RESET] /usr/bin"},
		{"no colon here", "no colon here"},
	}

	for _, tt := range tests {
		got := formatLabel(tt.input)
		if got != tt.want {
			t.Errorf("formatLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPrintResultOK(t *testing.T) {
	// Capture stdout
	output := captureOutput(func() {
		// Save and restore color codes
		oldGreen, oldReset, oldDim := green, reset, dim
		green, reset, dim = "", "", ""
		defer func() { green, reset, dim = oldGreen, oldReset, oldDim }()

		PrintResult(check.Result{
			Name:    "cmd: go",
			Status:  check.StatusOK,
			Details: []string{"path: /usr/bin/go", "version: 1.21"},
		})
	})

	expected := "[OK] cmd: go\n     path: /usr/bin/go\n     version: 1.21\n"
	if output != expected {
		t.Errorf("PrintResult output = %q, want %q", output, expected)
	}
}

func TestPrintResultFail(t *testing.T) {
	// Capture stdout
	output := captureOutput(func() {
		// Save and restore color codes
		oldRed, oldReset, oldDim := red, reset, dim
		red, reset, dim = "", "", ""
		defer func() { red, reset, dim = oldRed, oldReset, oldDim }()

		PrintResult(check.Result{
			Name:    "file: /missing",
			Status:  check.StatusFail,
			Details: []string{"not found"},
		})
	})

	expected := "[FAIL] file: /missing\n       not found\n"
	if output != expected {
		t.Errorf("PrintResult output = %q, want %q", output, expected)
	}
}

func TestPrintResultIndentation(t *testing.T) {
	// Test that OK and FAIL have correct indentation for alignment
	okOutput := captureOutput(func() {
		oldGreen, oldReset, oldDim := green, reset, dim
		green, reset, dim = "", "", ""
		defer func() { green, reset, dim = oldGreen, oldReset, oldDim }()

		PrintResult(check.Result{
			Name:    "test",
			Status:  check.StatusOK,
			Details: []string{"detail"},
		})
	})

	failOutput := captureOutput(func() {
		oldRed, oldReset, oldDim := red, reset, dim
		red, reset, dim = "", "", ""
		defer func() { red, reset, dim = oldRed, oldReset, oldDim }()

		PrintResult(check.Result{
			Name:    "test",
			Status:  check.StatusFail,
			Details: []string{"detail"},
		})
	})

	// OK: "[OK] " is 5 chars, so detail should have 5 space indent
	if !strings.Contains(okOutput, "\n     detail\n") {
		t.Errorf("OK output should have 5-space indent for details, got: %q", okOutput)
	}

	// FAIL: "[FAIL] " is 7 chars, so detail should have 7 space indent
	if !strings.Contains(failOutput, "\n       detail\n") {
		t.Errorf("FAIL output should have 7-space indent for details, got: %q", failOutput)
	}
}

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// envVars is a list of environment variables used by color detection
var envVars = []string{
	"PREFLIGHT_COLOR",
	"NO_COLOR",
	"CLICOLOR_FORCE",
	"GITHUB_ACTIONS",
	"GITLAB_CI",
	"CIRCLECI",
	"TRAVIS",
	"JENKINS_URL",
	"TERM",
	"TF_BUILD",
	"BUILDKITE",
	"TEAMCITY_VERSION",
	"DRONE",
	"BITBUCKET_BUILD_NUMBER",
	"CODEBUILD_BUILD_ID",
	"CI",
}

// clearColorEnvVars clears all color-related environment variables for testing
func clearColorEnvVars(t *testing.T) {
	t.Helper()
	for _, v := range envVars {
		t.Setenv(v, "")
		os.Unsetenv(v) //nolint:errcheck // test helper, errors don't matter
	}
}

func TestShouldEnableColor(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "PREFLIGHT_COLOR=1 forces colors on",
			env:  map[string]string{"PREFLIGHT_COLOR": "1"},
			want: true,
		},
		{
			name: "PREFLIGHT_COLOR=0 forces colors off",
			env:  map[string]string{"PREFLIGHT_COLOR": "0"},
			want: false,
		},
		{
			name: "PREFLIGHT_COLOR=0 overrides GITHUB_ACTIONS",
			env:  map[string]string{"PREFLIGHT_COLOR": "0", "GITHUB_ACTIONS": "true"},
			want: false,
		},
		{
			name: "NO_COLOR disables colors",
			env:  map[string]string{"NO_COLOR": "1"},
			want: false,
		},
		{
			name: "CLICOLOR_FORCE=1 enables colors",
			env:  map[string]string{"CLICOLOR_FORCE": "1"},
			want: true,
		},
		{
			name: "NO_COLOR takes precedence over CLICOLOR_FORCE",
			env:  map[string]string{"NO_COLOR": "1", "CLICOLOR_FORCE": "1"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearColorEnvVars(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got := shouldEnableColor()
			if got != tt.want {
				t.Errorf("shouldEnableColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCIEnvironment(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "GitHub Actions",
			env:  map[string]string{"GITHUB_ACTIONS": "true"},
			want: true,
		},
		{
			name: "GitLab CI",
			env:  map[string]string{"GITLAB_CI": "true"},
			want: true,
		},
		{
			name: "CircleCI",
			env:  map[string]string{"CIRCLECI": "true"},
			want: true,
		},
		{
			name: "Travis CI",
			env:  map[string]string{"TRAVIS": "true"},
			want: true,
		},
		{
			name: "Jenkins with TERM",
			env:  map[string]string{"JENKINS_URL": "http://jenkins.example.com", "TERM": "xterm"},
			want: true,
		},
		{
			name: "Jenkins without TERM",
			env:  map[string]string{"JENKINS_URL": "http://jenkins.example.com"},
			want: false,
		},
		{
			name: "Azure Pipelines",
			env:  map[string]string{"TF_BUILD": "True"},
			want: true,
		},
		{
			name: "Buildkite",
			env:  map[string]string{"BUILDKITE": "true"},
			want: true,
		},
		{
			name: "TeamCity",
			env:  map[string]string{"TEAMCITY_VERSION": "2023.1"},
			want: true,
		},
		{
			name: "Drone CI",
			env:  map[string]string{"DRONE": "true"},
			want: true,
		},
		{
			name: "Bitbucket Pipelines",
			env:  map[string]string{"BITBUCKET_BUILD_NUMBER": "123"},
			want: true,
		},
		{
			name: "AWS CodeBuild",
			env:  map[string]string{"CODEBUILD_BUILD_ID": "build-123"},
			want: true,
		},
		{
			name: "Woodpecker CI",
			env:  map[string]string{"CI": "woodpecker"},
			want: true,
		},
		{
			name: "Generic CI env var does not match",
			env:  map[string]string{"CI": "true"},
			want: false,
		},
		{
			name: "No CI environment",
			env:  map[string]string{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearColorEnvVars(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got := isCIEnvironment()
			if got != tt.want {
				t.Errorf("isCIEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}
