package check

import (
	"errors"
	"testing"
)

func TestResult_Fail(t *testing.T) {
	r := &Result{Name: "test"}
	err := errors.New("test error")

	result := r.Fail("something failed", err)

	if result.Status != StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, StatusFail)
	}
	if len(result.Details) != 1 || result.Details[0] != "something failed" {
		t.Errorf("Details = %v, want [something failed]", result.Details)
	}
	if result.Err != err {
		t.Errorf("Err = %v, want %v", result.Err, err)
	}
}

func TestResult_Failf(t *testing.T) {
	r := &Result{Name: "test"}

	result := r.Failf("value %d is invalid", 42)

	if result.Status != StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, StatusFail)
	}
	if len(result.Details) != 1 || result.Details[0] != "value 42 is invalid" {
		t.Errorf("Details = %v, want [value 42 is invalid]", result.Details)
	}
	if result.Err == nil || result.Err.Error() != "value 42 is invalid" {
		t.Errorf("Err = %v, want error with message 'value 42 is invalid'", result.Err)
	}
}

func TestResult_AddDetail(t *testing.T) {
	r := &Result{Name: "test"}

	result := r.AddDetail("first detail").AddDetail("second detail")

	if len(result.Details) != 2 {
		t.Errorf("len(Details) = %d, want 2", len(result.Details))
	}
	if result.Details[0] != "first detail" || result.Details[1] != "second detail" {
		t.Errorf("Details = %v, want [first detail, second detail]", result.Details)
	}
	if result != r {
		t.Error("AddDetail should return the same Result pointer")
	}
}

func TestResult_AddDetailf(t *testing.T) {
	r := &Result{Name: "test"}

	result := r.AddDetailf("path: %s", "/usr/bin/go")

	if len(result.Details) != 1 || result.Details[0] != "path: /usr/bin/go" {
		t.Errorf("Details = %v, want [path: /usr/bin/go]", result.Details)
	}
}

func TestCompileRegex(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		wantNil   bool
		wantError bool
	}{
		{
			name:    "empty pattern returns nil",
			pattern: "",
			wantNil: true,
		},
		{
			name:    "valid pattern compiles",
			pattern: "^foo.*bar$",
			wantNil: false,
		},
		{
			name:      "invalid pattern returns error",
			pattern:   "[invalid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := CompileRegex(tt.pattern)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.wantNil && re != nil {
				t.Errorf("expected nil regex, got %v", re)
			}
			if !tt.wantNil && re == nil {
				t.Error("expected non-nil regex, got nil")
			}
		})
	}
}
