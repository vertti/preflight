package check

// Status represents the outcome of a check.
type Status string

const (
	StatusOK   Status = "OK"
	StatusFail Status = "FAIL"
)

// Result holds the outcome of a single check.
type Result struct {
	Name    string   // e.g., "cmd:node", "env:DATABASE_URL"
	Status  Status   // OK or FAIL
	Details []string // human-readable details
	Err     error    // underlying error for failures
}

// OK returns true if the check passed.
func (r Result) OK() bool {
	return r.Status == StatusOK
}
