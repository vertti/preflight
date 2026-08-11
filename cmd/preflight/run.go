package main

import (
	"errors"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/output"
)

// Checker is implemented by all check types.
type Checker interface {
	Run() check.Result
}

// ErrCheckFailed is returned when a check fails.
var ErrCheckFailed = errors.New("check failed")

// checkRan records that a check actually executed. Exec mode is gated on it:
// the root command has no RunE, so `preflight -- ./app` and
// `preflight --help -- ./app` reach the end of Execute() with a nil error,
// which would otherwise be indistinguishable from "every check passed".
var checkRan bool

// runCheck executes a check, prints the result, and returns an error if failed.
// The returned error causes Cobra to exit with code 1.
func runCheck(c Checker) error {
	checkRan = true
	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		return ErrCheckFailed
	}
	return nil
}
