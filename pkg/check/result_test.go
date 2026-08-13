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

// Eleven call sites across httpcheck and promcheck each wrote out the attempt
// count alongside their own failure message, so the two could drift apart —
// and two of them worded it differently from the other nine.
func TestResult_FailAfter(t *testing.T) {
	t.Run("notes the count when the check retried", func(t *testing.T) {
		r := &Result{Name: "http: /health"}

		result := r.FailAfter(3, "status %d, expected %d", 503, 200)

		if result.Status != StatusFail {
			t.Errorf("Status = %v, want %v", result.Status, StatusFail)
		}
		want := "status 503, expected 200 (after 3 attempts)"
		if len(result.Details) != 1 || result.Details[0] != want {
			t.Errorf("Details = %v, want [%s]", result.Details, want)
		}
	})

	// A single attempt is not a retry, and saying "after 1 attempts" would be
	// noise on every check that never asked for one.
	t.Run("stays quiet when there was only one attempt", func(t *testing.T) {
		r := &Result{Name: "http: /health"}

		result := r.FailAfter(1, "status %d, expected %d", 503, 200)

		want := "status 503, expected 200"
		if len(result.Details) != 1 || result.Details[0] != want {
			t.Errorf("Details = %v, want [%s]", result.Details, want)
		}
	})

	t.Run("works without arguments", func(t *testing.T) {
		r := &Result{Name: "x"}

		result := r.FailAfter(2, "no data")

		want := "no data (after 2 attempts)"
		if len(result.Details) != 1 || result.Details[0] != want {
			t.Errorf("Details = %v, want [%s]", result.Details, want)
		}
	})

	t.Run("does not disturb the caller's arguments", func(t *testing.T) {
		r := &Result{Name: "x"}
		args := make([]any, 1, 4) // spare capacity an append could scribble on
		args[0] = "value"

		r.FailAfter(2, "got %v", args...)

		if len(args) != 1 || args[0] != "value" {
			t.Errorf("args = %v, want [value]", args)
		}
	})
}
