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
