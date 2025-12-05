package main

import (
	"os"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/output"
)

// Checker is implemented by all check types.
type Checker interface {
	Run() check.Result
}

// runCheck executes a check, prints the result, and exits with code 1 if failed.
func runCheck(c Checker) error {
	result := c.Run()
	output.PrintResult(result)

	if !result.OK() {
		os.Exit(1)
	}
	return nil
}
