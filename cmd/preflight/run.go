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

// runCheck executes a check, prints the result, and returns an error if failed.
// The returned error causes Cobra to exit with code 1.
func runCheck(c Checker) error {
	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		return ErrCheckFailed
	}
	return nil
}
