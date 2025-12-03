package check

import "testing"

func TestStatus(t *testing.T) {
	if StatusOK != "OK" {
		t.Errorf("StatusOK = %q, want %q", StatusOK, "OK")
	}
	if StatusFail != "FAIL" {
		t.Errorf("StatusFail = %q, want %q", StatusFail, "FAIL")
	}
}

func TestCheckResult(t *testing.T) {
	result := Result{
		Name:    "cmd:node",
		Status:  StatusOK,
		Details: []string{"path: /usr/bin/node", "version: 18.0.0"},
	}

	if result.Name != "cmd:node" {
		t.Errorf("Name = %q, want %q", result.Name, "cmd:node")
	}
	if result.Status != StatusOK {
		t.Errorf("Status = %q, want %q", result.Status, StatusOK)
	}
	if len(result.Details) != 2 {
		t.Errorf("len(Details) = %d, want 2", len(result.Details))
	}
}

func TestResultOK(t *testing.T) {
	result := Result{Status: StatusOK}
	if !result.OK() {
		t.Error("OK() = false, want true for StatusOK")
	}

	result.Status = StatusFail
	if result.OK() {
		t.Error("OK() = true, want false for StatusFail")
	}
}
